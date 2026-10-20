package httpserver

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mldl-exam-trainer/internal/auth"
	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/db"
	"mldl-exam-trainer/internal/progress"
	"mldl-exam-trainer/internal/quiz"
)

type server struct {
	store       *db.Store
	catalog     *content.Catalog
	projectRoot string
	random      *rand.Rand
	randomMu    sync.Mutex
	auth        *auth.Service
}

func NewHandler(store *db.Store, catalog *content.Catalog, projectRoot string, staticFiles fs.FS) http.Handler {
	return NewHandlerWithAuth(store, catalog, projectRoot, staticFiles, nil)
}

// NewHandlerWithAuth enables the production single-user login when login is
// non-nil. NewHandler remains unauthenticated solely for existing local tests.
func NewHandlerWithAuth(store *db.Store, catalog *content.Catalog, projectRoot string, staticFiles fs.FS, login *auth.Service) http.Handler {
	s := &server{store: store, catalog: catalog, projectRoot: projectRoot, random: rand.New(rand.NewSource(time.Now().UnixNano())), auth: login}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("POST /api/auth/password", s.changePassword)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/catalog", s.getCatalog)
	mux.HandleFunc("GET /api/questions", s.getQuestions)
	mux.HandleFunc("GET /api/comprehension", s.getComprehension)
	mux.HandleFunc("POST /api/comprehension/answers", s.answerComprehension)
	mux.HandleFunc("POST /api/answers", s.submitAnswer)
	mux.HandleFunc("POST /api/topics/{topicID}/view", s.viewTopic)
	mux.HandleFunc("GET /api/progress", s.getProgress)
	mux.HandleFunc("GET /api/progress/{examID}", s.getExamProgress)
	mux.HandleFunc("GET /api/reviews/due", s.getDue)
	mux.HandleFunc("GET /api/today", s.getToday)
	mux.HandleFunc("GET /api/exam/questions", s.getExamQuestions)
	mux.HandleFunc("POST /api/exam/finish", s.finishExam)
	mux.HandleFunc("GET /api/backup", s.downloadBackup)
	mux.HandleFunc("POST /api/restore", s.restoreBackup)
	mux.HandleFunc("GET /pdf/{file}", s.getPDF)
	mux.HandleFunc("GET /api/settings/theme", s.getTheme)
	mux.HandleFunc("POST /api/settings/theme", s.setTheme)
	for pattern, allow := range map[string]string{
		"/healthz":                   http.MethodGet,
		"/api/catalog":               http.MethodGet,
		"/api/questions":             http.MethodGet,
		"/api/comprehension":         http.MethodGet,
		"/api/comprehension/answers": http.MethodPost,
		"/api/answers":               http.MethodPost,
		"/api/topics/{topicID}/view": http.MethodPost,
		"/api/progress":              http.MethodGet,
		"/api/progress/{examID}":     http.MethodGet,
		"/api/reviews/due":           http.MethodGet,
		"/api/today":                 http.MethodGet,
		"/api/exam/questions":        http.MethodGet,
		"/api/exam/finish":           http.MethodPost,
		"/api/backup":                http.MethodGet,
		"/api/restore":               http.MethodPost,
		"/pdf/{file}":                http.MethodGet,
		"/api/settings/theme":        http.MethodGet + ", " + http.MethodPost,
		"/api/auth/status":           http.MethodGet,
		"/api/auth/login":            http.MethodPost,
		"/api/auth/logout":           http.MethodPost,
		"/api/auth/password":         http.MethodPost,
		"/api/admin/users":           http.MethodGet,
	} {
		mux.HandleFunc(pattern, methodNotAllowed(allow))
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unbekannter API-Endpunkt."})
	})
	mux.Handle("/", http.FileServerFS(staticFiles))
	return withSecurityHeaders(withSameOrigin(withRecovery(withAuthentication(mux, login)), login), login)
}

func (s *server) authStatus(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
		return
	}
	_, authenticated := s.auth.Authenticate(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": authenticated})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Login ist nicht aktiviert."})
		return
	}
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSONRequest(w, r, 8*1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Anmeldedaten.")
		return
	}
	if _, err := s.auth.Login(w, r, request.Username, request.Password); err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	} else if limited, ok := err.(*auth.RateLimitError); ok {
		seconds := int(limited.RetryAfter.Round(time.Second).Seconds())
		w.Header().Set("Retry-After", strconv.Itoa(max(1, seconds)))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Benutzername oder Passwort ungültig."})
	} else {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Benutzername oder Passwort ungültig."})
	}
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Logout(w, principalFromContext(r.Context())); err != nil {
		s.internalError(w, "Abmelden konnte nicht abgeschlossen werden.", "Logout", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) changePassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSONRequest(w, r, 8*1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Passwortänderung.")
		return
	}
	if err := s.auth.ChangePassword(principalFromContext(r.Context()), request.CurrentPassword, request.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Passwort konnte nicht geändert werden."})
		return
	}
	// The password change invalidates even the current session.
	s.auth.ClearCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) adminUsers(w http.ResponseWriter, r *http.Request) {
	p := principalFromContext(r.Context())
	if p.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Admin-Berechtigung erforderlich."})
		return
	}
	rows, err := s.store.DB.Query(`SELECT id, username, role, created_at, updated_at, disabled FROM users ORDER BY id`)
	if err != nil {
		s.internalError(w, "Benutzer konnten nicht geladen werden.", "Admin-Benutzer laden", err)
		return
	}
	defer rows.Close()
	type user struct {
		ID        int64     `json:"id"`
		Username  string    `json:"username"`
		Role      auth.Role `json:"role"`
		CreatedAt string    `json:"created_at"`
		UpdatedAt string    `json:"updated_at"`
		Disabled  bool      `json:"disabled"`
	}
	users := []user{}
	for rows.Next() {
		var item user
		if err := rows.Scan(&item.ID, &item.Username, &item.Role, &item.CreatedAt, &item.UpdatedAt, &item.Disabled); err != nil {
			s.internalError(w, "Benutzer konnten nicht geladen werden.", "Admin-Benutzer lesen", err)
			return
		}
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		s.internalError(w, "Benutzer konnten nicht geladen werden.", "Admin-Benutzer lesen", err)
		return
	}
	log.Printf("security event=admin_users_listed user_id=%d", p.ID)
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "HTTP-Methode ist für diesen Endpunkt nicht erlaubt."})
	}
}

func (s *server) getQuestions(w http.ResponseWriter, r *http.Request) {
	examID := r.URL.Query().Get("exam")
	moduleID := r.URL.Query().Get("module")
	if !s.hasExam(examID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekannter Prüfungsbereich."})
		return
	}
	if moduleID != "" && !s.hasModule(examID, moduleID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekanntes Modul für diesen Prüfungsbereich."})
		return
	}
	questions := quiz.Filter(s.catalog, examID, moduleID)
	s.randomMu.Lock()
	s.random.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	public := make([]quiz.PublicQuestion, len(questions))
	for index, question := range questions {
		public[index] = quiz.ToPublic(question, s.random)
	}
	s.randomMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"questions": public})
}

func (s *server) submitAnswer(w http.ResponseWriter, r *http.Request) {
	var request struct {
		QuestionID string      `json:"question_id"`
		Response   quiz.Answer `json:"response"`
		Session    *struct {
			QuestionIDs  []string `json:"question_ids"`
			CurrentIndex int      `json:"current_index"`
		} `json:"session,omitempty"`
	}
	if err := decodeJSONRequest(w, r, 16*1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Antwortdaten.")
		return
	}
	if strings.TrimSpace(request.QuestionID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Antwortdaten."})
		return
	}
	question, found := content.FindQuestion(s.catalog, request.QuestionID)
	if !found || !question.Enabled {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unbekannte Frage."})
		return
	}
	result, err := quiz.Evaluate(*question, request.Response)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.store.RecordAnswer(s.catalog, *question, result.Correct, request.Response); err != nil {
		s.internalError(w, "Antwort konnte nicht gespeichert werden.", "Antwort speichern", err)
		return
	}
	var retryIDs []string
	if !result.Correct && request.Session != nil {
		retryIDs = progress.PlanRetry(question.ID, request.Session.QuestionIDs, request.Session.CurrentIndex, 3)
	}
	writeJSON(w, http.StatusOK, struct {
		quiz.Result
		RetryQuestionIDs []string `json:"retry_question_ids,omitempty"`
	}{Result: result, RetryQuestionIDs: retryIDs})
}

func (s *server) viewTopic(w http.ResponseWriter, r *http.Request) {
	if err := s.store.MarkTopicSeen(s.catalog, r.PathValue("topicID")); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unbekanntes Topic."})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *server) getProgress(w http.ResponseWriter, _ *http.Request) {
	v, err := s.store.Progress(s.catalog, "")
	if err != nil {
		s.internalError(w, "Fortschritt konnte nicht geladen werden.", "Fortschritt laden", err)
		return
	}
	writeJSON(w, 200, map[string]any{"exams": v})
}
func (s *server) getExamProgress(w http.ResponseWriter, r *http.Request) {
	v, err := s.store.Progress(s.catalog, r.PathValue("examID"))
	if err != nil {
		s.internalError(w, "Fortschritt konnte nicht geladen werden.", "Prüfungsfortschritt laden", err)
		return
	}
	if len(v) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unbekannter Prüfungsbereich."})
		return
	}
	writeJSON(w, 200, v[0])
}
func (s *server) getDue(w http.ResponseWriter, r *http.Request) {
	exam := r.URL.Query().Get("exam")
	if !s.hasExam(exam) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekannter Prüfungsbereich."})
		return
	}
	items, err := s.store.Candidates(s.catalog, exam)
	if err != nil {
		s.internalError(w, "Fälligkeiten konnten nicht geladen werden.", "Fälligkeiten laden", err)
		return
	}
	now := s.store.Clock.Now().UTC()
	localNow := now.In(s.store.Location)
	weekEnd := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, s.store.Location).AddDate(0, 0, 7)
	overdue, today, week := 0, 0, 0
	for _, item := range items {
		if item.New {
			continue
		}
		switch item.DueState {
		case progress.DueOverdue:
			overdue++
		case progress.DueToday:
			today++
		case progress.DueLater:
			if item.Due.In(s.store.Location).Before(weekEnd) {
				week++
			}
		}
	}
	writeJSON(w, 200, map[string]any{"exam_id": exam, "overdue": overdue, "today": today, "this_week": week, "items": items})
}
func (s *server) getToday(w http.ResponseWriter, r *http.Request) {
	exam := r.URL.Query().Get("exam")
	if !s.hasExam(exam) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekannter Prüfungsbereich."})
		return
	}
	minutes, err := strconv.Atoi(r.URL.Query().Get("minutes"))
	if err != nil || !map[int]bool{10: true, 20: true, 30: true, 45: true}[minutes] {
		writeJSON(w, 400, map[string]string{"error": "Zeit muss 10, 20, 30 oder 45 Minuten sein."})
		return
	}
	items, err := s.store.Candidates(s.catalog, exam)
	if err != nil {
		s.internalError(w, "Session konnte nicht erstellt werden.", "Heute-Session erstellen", err)
		return
	}
	selected := progress.SelectSession(items, minutes, s.store.Clock.Now().UTC())
	ids := make([]string, len(selected))
	reviews, weak, newCount := 0, 0, 0
	newTopics := map[string]bool{}
	newTopicIDs := make([]string, 0, 2)
	for i, item := range selected {
		ids[i] = item.QuestionID
		if item.New {
			newCount++
			if !newTopics[item.TopicID] {
				newTopics[item.TopicID] = true
				newTopicIDs = append(newTopicIDs, item.TopicID)
			}
		} else {
			reviews++
		}
		if item.Weak {
			weak++
		}
	}
	estimated := len(ids)*2 + len(newTopics)*8
	writeJSON(w, 200, map[string]any{"exam_id": exam, "minutes": minutes, "question_ids": ids, "question_count": len(ids), "review_count": reviews, "weak_count": weak, "new_count": newCount, "new_topic_count": len(newTopics), "new_topic_ids": newTopicIDs, "estimated_minutes": min(minutes, estimated)})
}

func (s *server) hasExam(id string) bool {
	for _, exam := range s.catalog.Exams {
		if exam.ID == id {
			return true
		}
	}
	return false
}

func (s *server) hasModule(examID, moduleID string) bool {
	for _, module := range s.catalog.Modules {
		if module.ID == moduleID && module.ExamID == examID {
			return true
		}
	}
	return false
}

func (s *server) getExamQuestions(w http.ResponseWriter, r *http.Request) {
	examID := r.URL.Query().Get("exam")
	if !s.hasExam(examID) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekannter Prüfungsbereich."})
		return
	}
	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || (count != 10 && count != 20 && count != 30) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Fragenzahl muss 10, 20 oder 30 sein."})
		return
	}
	scope := r.URL.Query().Get("scope")
	if scope != "all" && scope != "core" && scope != "weak" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Umfang muss all, core oder weak sein."})
		return
	}
	allowedTopics := map[string]bool{}
	if scope == "weak" {
		items, progressErr := s.store.Progress(s.catalog, examID)
		if progressErr != nil {
			s.internalError(w, "Schwachstellen konnten nicht geladen werden.", "Schwachstellen laden", progressErr)
			return
		}
		for _, topic := range items[0].Topics {
			if topic.Status == progress.Weak {
				allowedTopics[topic.TopicID] = true
			}
		}
	} else {
		for _, topic := range s.catalog.Topics {
			if topic.ExamID == examID && (scope == "all" || topic.Priority == content.PriorityCore) {
				allowedTopics[topic.ID] = true
			}
		}
	}
	questions := make([]content.Question, 0, count)
	for _, question := range s.catalog.Questions {
		if question.Enabled && allowedTopics[question.TopicID] {
			questions = append(questions, question)
		}
	}
	s.randomMu.Lock()
	s.random.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
	if len(questions) > count {
		questions = questions[:count]
	}
	public := make([]quiz.PublicQuestion, len(questions))
	for index, question := range questions {
		public[index] = quiz.ToPublic(question, s.random)
	}
	s.randomMu.Unlock()
	attemptID := ""
	if len(questions) > 0 {
		attemptID, err = newAttemptID()
		if err != nil {
			s.internalError(w, "Prüfung konnte nicht vorbereitet werden.", "Prüfungs-ID erzeugen", err)
			return
		}
		aliases := make(map[string]string, len(questions))
		for index, question := range questions {
			alias, aliasErr := newAttemptID()
			if aliasErr != nil {
				s.internalError(w, "Prüfung konnte nicht vorbereitet werden.", "Prüfungsalias erzeugen", aliasErr)
				return
			}
			public[index].ID = alias
			aliases[alias] = question.ID
		}
		now := s.store.Clock.Now().UTC()
		if err = s.store.DeleteExpiredExamAttempts(now.Add(-4 * time.Hour)); err != nil {
			s.internalError(w, "Prüfung konnte nicht vorbereitet werden.", "Abgelaufene Prüfungsversuche löschen", err)
			return
		}
		if err = s.store.SaveExamAttempt(db.ExamAttempt{ID: attemptID, ExamID: examID, CreatedAt: now, QuestionAliases: aliases}); err != nil {
			s.internalError(w, "Prüfung konnte nicht vorbereitet werden.", "Prüfungsversuch speichern", err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"attempt_id": attemptID, "exam_id": examID, "scope": scope, "requested_count": count, "actual_count": len(public), "questions": public})
}

func (s *server) finishExam(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AttemptID string `json:"attempt_id"`
		ExamID    string `json:"exam_id"`
		Answers   []struct {
			QuestionID string      `json:"question_id"`
			Response   quiz.Answer `json:"response"`
		} `json:"answers"`
	}
	if err := decodeJSONRequest(w, r, 256*1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Prüfungsabgabe.")
		return
	}
	if request.AttemptID == "" || !s.hasExam(request.ExamID) || len(request.Answers) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Prüfungsabgabe."})
		return
	}
	attempt, found, err := s.store.ExamAttempt(request.AttemptID)
	if err != nil {
		s.internalError(w, "Prüfungsversuch konnte nicht gelesen werden.", "Prüfungsversuch lesen", err)
		return
	}
	if !found || s.store.Clock.Now().UTC().Sub(attempt.CreatedAt) > 4*time.Hour || attempt.ExamID != request.ExamID || len(request.Answers) != len(attempt.QuestionAliases) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Prüfung ist unbekannt, abgelaufen oder unvollständig."})
		return
	}
	type itemResult struct {
		QuestionID         string      `json:"question_id"`
		PracticeQuestionID string      `json:"practice_question_id"`
		TopicID            string      `json:"topic_id"`
		ModuleID           string      `json:"module_id"`
		Response           quiz.Answer `json:"response"`
		quiz.Result
	}
	type breakdown struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Correct int    `json:"correct"`
		Total   int    `json:"total"`
	}
	seen := map[string]bool{}
	results := make([]itemResult, 0, len(request.Answers))
	records := make([]db.AnswerRecord, 0, len(request.Answers))
	topicCounts := map[string]*breakdown{}
	moduleCounts := map[string]*breakdown{}
	correctTotal := 0
	for _, answer := range request.Answers {
		internalQuestionID, belongsToAttempt := attempt.QuestionAliases[answer.QuestionID]
		if seen[answer.QuestionID] || !belongsToAttempt {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Eine Frage wurde mehrfach abgegeben."})
			return
		}
		seen[answer.QuestionID] = true
		question, found := content.FindQuestion(s.catalog, internalQuestionID)
		if !found || !question.Enabled {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unbekannte Prüfungsfrage."})
			return
		}
		topic, found := content.FindTopic(s.catalog, question.TopicID)
		if !found || topic.ExamID != request.ExamID {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Frage gehört nicht zu dieser Prüfung."})
			return
		}
		result, err := quiz.Evaluate(*question, answer.Response)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if result.Correct {
			correctTotal++
		}
		results = append(results, itemResult{QuestionID: answer.QuestionID, PracticeQuestionID: question.ID, TopicID: topic.ID, ModuleID: topic.ModuleID, Response: answer.Response, Result: result})
		records = append(records, db.AnswerRecord{Question: *question, Correct: result.Correct, Response: answer.Response})
		if topicCounts[topic.ID] == nil {
			topicCounts[topic.ID] = &breakdown{ID: topic.ID, Name: topic.Title}
		}
		topicCounts[topic.ID].Total++
		if result.Correct {
			topicCounts[topic.ID].Correct++
		}
		moduleName := topic.ModuleID
		for _, module := range s.catalog.Modules {
			if module.ID == topic.ModuleID {
				moduleName = module.Name
				break
			}
		}
		if moduleCounts[topic.ModuleID] == nil {
			moduleCounts[topic.ModuleID] = &breakdown{ID: topic.ModuleID, Name: moduleName}
		}
		moduleCounts[topic.ModuleID].Total++
		if result.Correct {
			moduleCounts[topic.ModuleID].Correct++
		}
	}
	claimed, err := s.store.DeleteExamAttempt(request.AttemptID)
	if err != nil {
		s.internalError(w, "Prüfungsabschluss konnte nicht bestätigt werden.", "Prüfungsversuch beanspruchen", err)
		return
	}
	if !claimed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Diese Prüfung wurde bereits abgegeben."})
		return
	}
	if err := s.store.RecordAnswers(s.catalog, records); err != nil {
		_ = s.store.SaveExamAttempt(attempt)
		s.internalError(w, "Prüfung konnte nicht gespeichert werden.", "Prüfungsantworten speichern", err)
		return
	}
	topics := make([]breakdown, 0, len(topicCounts))
	for _, topic := range s.catalog.Topics {
		if item := topicCounts[topic.ID]; item != nil {
			topics = append(topics, *item)
		}
	}
	modules := make([]breakdown, 0, len(moduleCounts))
	for _, module := range s.catalog.Modules {
		if item := moduleCounts[module.ID]; item != nil {
			modules = append(modules, *item)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"exam_id": request.ExamID, "correct": correctTotal, "wrong": len(results) - correctTotal, "total": len(results), "percent": correctTotal * 100 / len(results), "results": results, "topics": topics, "modules": modules})
}

func newAttemptID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *server) getCatalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.catalog.WithSourceURLs())
}

func (s *server) downloadBackup(w http.ResponseWriter, _ *http.Request) {
	backup, err := s.store.ExportBackup()
	if err != nil {
		s.internalError(w, "Backup konnte nicht erstellt werden.", "Backup exportieren", err)
		return
	}
	filename := "mldl-exam-trainer-backup-" + s.store.Clock.Now().In(s.store.Location).Format("2006-01-02") + ".json"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	writeJSON(w, http.StatusOK, backup)
}

func (s *server) restoreBackup(w http.ResponseWriter, r *http.Request) {
	if err := requireJSONContentType(r); err != nil {
		writeRequestError(w, err, "Backup muss als JSON gesendet werden.")
		return
	}
	limited := http.MaxBytesReader(w, r.Body, 8*1024*1024)
	backup, err := db.DecodeBackup(limited)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Backup ist grösser als 8 MB."})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Backup ist kein gültiges JSON-Dokument."})
		return
	}
	if err := s.store.ValidateBackup(backup, s.catalog); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Backup ist ungültig oder nicht kompatibel: " + err.Error()})
		return
	}
	safetyDirectory := filepath.Join(s.projectRoot, "data", "backups")
	if s.store.DataDir != "" {
		safetyDirectory = filepath.Join(s.store.DataDir, "backups")
	}
	safetyPath, err := s.store.RestoreBackup(backup, s.catalog, safetyDirectory)
	if err != nil {
		s.internalError(w, "Backup konnte nicht wiederhergestellt werden. Der bisherige Lernstand blieb erhalten.", "Backup wiederherstellen", err)
		return
	}
	theme, err := s.store.Theme()
	if err != nil {
		theme = "light"
	}
	log.Printf("Backup erfolgreich wiederhergestellt; Sicherheitskopie: %s", filepath.Base(safetyPath))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "theme": theme})
}

func (s *server) getPDF(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	registered := false
	for _, pdf := range s.catalog.PDFs {
		if pdf.File == file {
			registered = true
			break
		}
	}
	if !registered {
		http.NotFound(w, r)
		return
	}
	pdfRoot, err := filepath.Abs(filepath.Join(s.projectRoot, "content", "pdfs"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	target, err := filepath.Abs(filepath.Join(pdfRoot, file))
	if err != nil || filepath.Dir(target) != pdfRoot {
		http.NotFound(w, r)
		return
	}
	linkInfo, err := os.Lstat(target)
	if err != nil || !linkInfo.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline")
	http.ServeFile(w, r, target)
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	if err := s.store.DB.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) getTheme(w http.ResponseWriter, _ *http.Request) {
	theme, err := s.store.Theme()
	if err != nil {
		s.internalError(w, "Einstellung konnte nicht geladen werden.", "Theme laden", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"theme": theme})
}

func (s *server) setTheme(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Theme string `json:"theme"`
	}
	if err := decodeJSONRequest(w, r, 1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Anfrage.")
		return
	}
	if err := s.store.SetTheme(request.Theme); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Theme muss light, dark oder system sein."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"theme": request.Theme})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

var errJSONContentType = errors.New("Content-Type muss application/json sein")

func requireJSONContentType(r *http.Request) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errJSONContentType
	}
	return nil
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, limit int64, destination any) error {
	if err := requireJSONContentType(r); err != nil {
		return err
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unerwartete weitere JSON-Daten")
		}
		return err
	}
	return nil
}

func writeRequestError(w http.ResponseWriter, err error, fallback string) {
	if errors.Is(err, errJSONContentType) {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type muss application/json sein."})
		return
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Anfrage ist zu gross."})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": fallback})
}

func (s *server) internalError(w http.ResponseWriter, message, operation string, err error) {
	log.Printf("%s fehlgeschlagen: %v", operation, err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": message})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("Panic in %s %s: %v", r.Method, r.URL.Path, recovered)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Interner Serverfehler."})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func withSameOrigin(next http.Handler, login *auth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Cross-Site-Anfrage abgelehnt."})
				return
			}
			if rawOrigin := r.Header.Get("Origin"); rawOrigin != "" {
				origin, err := url.Parse(rawOrigin)
				// A browser-attested same-origin request already covers the scheme, so
				// the host comparison suffices and no forwarded header is consulted.
				sameOrigin := err == nil && strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "same-origin") && (origin.Scheme == "https" || origin.Scheme == "http")
				if err != nil || (!sameOrigin && origin.Scheme != requestScheme(r, login)) || !strings.EqualFold(origin.Host, r.Host) {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "Fremde Herkunft abgelehnt."})
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

type principalContextKey struct{}

func principalFromContext(ctx context.Context) auth.Principal {
	principal, _ := ctx.Value(principalContextKey{}).(auth.Principal)
	return principal
}

func withAuthentication(next http.Handler, login *auth.Service) http.Handler {
	if login == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Static login assets remain public. Every API endpoint except the
		// authentication handshake and every course PDF needs a valid session.
		if r.URL.Path == "/healthz" || r.URL.Path == "/api/auth/status" || r.URL.Path == "/api/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/pdf/") {
			principal, authenticated := login.Authenticate(w, r)
			if !authenticated {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Bitte zuerst anmelden."})
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && !login.ValidCSRF(principal, r) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "CSRF-Prüfung fehlgeschlagen."})
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal))
		}
		next.ServeHTTP(w, r)
	})
}

func requestScheme(r *http.Request, login *auth.Service) string {
	if r.TLS != nil || (login.IsTrustedProxy(r) && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")) {
		return "https"
	}
	return "http"
}

func withSecurityHeaders(next http.Handler, login *auth.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cache-Control", "no-cache")
		if requestScheme(r, login) == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
