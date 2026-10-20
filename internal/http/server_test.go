package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/db"
	"mldl-exam-trainer/internal/quiz"
)

func testHandler(t *testing.T) http.Handler {
	handler, _, _ := testEnvironment(t)
	return handler
}

func jsonRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func testEnvironment(t *testing.T) (http.Handler, *db.Store, *content.Catalog) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	files := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>Test trainer</title>")},
		"styles.css": {Data: []byte("body{}")},
		"app.js":     {Data: []byte("void 0")},
	}
	catalog := &content.Catalog{
		Exams: []content.Exam{{ID: "machine-learning", Name: "Machine Learning", SortOrder: 1}, {ID: "deep-learning-data-science", Name: "Deep Learning", SortOrder: 2}},
		Modules: []content.Module{
			{ID: "ml-foundations", ExamID: "machine-learning", Name: "Grundlagen", SortOrder: 1},
			{ID: "ml-models", ExamID: "machine-learning", Name: "Modelle", SortOrder: 2},
			{ID: "dl-training", ExamID: "deep-learning-data-science", Name: "Training", SortOrder: 1},
		},
		Topics: []content.Topic{
			{ID: "example", ExamID: "machine-learning", ModuleID: "ml-foundations", Title: "Beispiel", Priority: content.PriorityCore, Summary: "Kern", Explanation: []string{"Text"}, Sources: []content.Source{{File: "source.pdf", Path: "source.pdf", Pages: []int{1}}}, InitialStatus: "unlearned", SortOrder: 1},
			{ID: "support-example", ExamID: "machine-learning", ModuleID: "ml-models", Title: "Support-Beispiel", Priority: content.PrioritySupport, Summary: "Ergänzung", Explanation: []string{"Text"}, Sources: []content.Source{{File: "source.pdf", Path: "source.pdf", Pages: []int{2}}}, InitialStatus: "unlearned", SortOrder: 1},
			{ID: "dl-example", ExamID: "deep-learning-data-science", ModuleID: "dl-training", Title: "DL-Beispiel", Priority: content.PriorityCore, Summary: "DL-Kern", Explanation: []string{"Text"}, Sources: []content.Source{{File: "source.pdf", Path: "source.pdf", Pages: []int{3}}}, InitialStatus: "unlearned", SortOrder: 1},
		},
		Questions: []content.Question{
			{ID: "enabled-question", TopicID: "example", Type: content.QuestionSingle, Prompt: "Frage", Explanation: "Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectAnswer: "a"},
			{ID: "spacing-one", TopicID: "example", Type: content.QuestionSingle, Prompt: "Abstand 1", Explanation: "Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectAnswer: "a"},
			{ID: "spacing-two", TopicID: "example", Type: content.QuestionSingle, Prompt: "Abstand 2", Explanation: "Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectAnswer: "a"},
			{ID: "spacing-three", TopicID: "example", Type: content.QuestionSingle, Prompt: "Abstand 3", Explanation: "Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectAnswer: "a"},
			{ID: "disabled-question", TopicID: "example", Type: content.QuestionSingle, Prompt: "Aus", Explanation: "Erklärung", Difficulty: 1, Enabled: false, Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectAnswer: "a"},
			{ID: "support-question", TopicID: "support-example", Type: content.QuestionSingle, Prompt: "Support-Frage", Explanation: "Support-Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "yes", Text: "Ja"}, {ID: "no", Text: "Nein"}}, CorrectAnswer: "yes"},
			{ID: "dl-question", TopicID: "dl-example", Type: content.QuestionSingle, Prompt: "DL-Frage", Explanation: "DL-Erklärung", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "yes", Text: "Ja"}, {ID: "no", Text: "Nein"}}, CorrectAnswer: "yes"},
		},
	}
	if err := store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	return NewHandler(store, catalog, t.TempDir(), files), store, catalog
}

func TestQuestionsExcludeSolutionsAndDisabledEntries(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/questions?exam=machine-learning", nil)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "enabled-question") || strings.Contains(body, "disabled-question") || strings.Contains(body, "correct_answer") {
		t.Fatalf("status=%d body=%s", response.Code, body)
	}
}

func TestQuestionsRejectUnknownExamAndMismatchedModule(t *testing.T) {
	handler := testHandler(t)
	for _, target := range []string{
		"/api/questions?exam=unknown",
		"/api/questions?exam=machine-learning&module=dl-training",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "error") {
			t.Fatalf("target=%s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}
}

func TestSubmitAnswerEvaluatesAndStoresReview(t *testing.T) {
	handler := testHandler(t)
	request := jsonRequest(http.MethodPost, "/api/answers", `{"question_id":"enabled-question","response":{"option_ids":["a"]}}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"correct":true`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSubmitAnswerRejectsDisabledQuestion(t *testing.T) {
	request := jsonRequest(http.MethodPost, "/api/answers", `{"question_id":"disabled-question","response":{"option_ids":["a"]}}`)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestWrongAnswerReturnsDelayedRetryPlan(t *testing.T) {
	handler := testHandler(t)
	body := `{"question_id":"enabled-question","response":{"option_ids":["b"]},"session":{"question_ids":["spacing-one","spacing-two","spacing-three","enabled-question"],"current_index":3}}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, jsonRequest(http.MethodPost, "/api/answers", body))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, id := range []string{"spacing-three", "spacing-two", "spacing-one", "enabled-question"} {
		if !strings.Contains(response.Body.String(), id) {
			t.Fatalf("retry plan missing %s: %s", id, response.Body.String())
		}
	}
}

func TestCatalogAPI(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"machine-learning"`) || strings.Contains(response.Body.String(), `"path":"source.pdf"`) {
		t.Fatalf("unexpected public catalog: %s", response.Body.String())
	}
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestServesEmbeddedFrontend(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "Test trainer") {
		t.Fatalf("frontend was not served: %q", response.Body.String())
	}
}

func TestThemeAPI(t *testing.T) {
	handler := testHandler(t)
	request := jsonRequest(http.MethodPost, "/api/settings/theme", `{"theme":"dark"}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("POST status = %d, body = %s", response.Code, body)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/settings/theme", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `"theme":"dark"`) {
		t.Fatalf("GET body = %q", response.Body.String())
	}
}

func TestThemeAPIRejectsInvalidInput(t *testing.T) {
	request := jsonRequest(http.MethodPost, "/api/settings/theme", `{"theme":"neon"}`)
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestProgressAndTodayAPIsKeepExamScope(t *testing.T) {
	handler := testHandler(t)
	for _, path := range []string{"/api/progress", "/api/progress/machine-learning", "/api/reviews/due?exam=machine-learning", "/api/today?exam=machine-learning&minutes=10"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
		if path != "/api/progress" && strings.Contains(response.Body.String(), "deep-learning-data-science") {
			t.Fatalf("%s mixed exams: %s", path, response.Body.String())
		}
	}
}

func TestTodayTimeModelsAndUnknownExam(t *testing.T) {
	handler := testHandler(t)
	for _, test := range []struct {
		minutes int
		wantNew bool
	}{{10, false}, {20, false}, {30, true}, {45, true}} {
		response := httptest.NewRecorder()
		path := fmt.Sprintf("/api/today?exam=machine-learning&minutes=%d", test.minutes)
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, response.Code)
		}
		hasNew := !strings.Contains(response.Body.String(), `"new_topic_count":0`)
		if hasNew != test.wantNew {
			t.Fatalf("minutes=%d body=%s", test.minutes, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/today?exam=unknown&minutes=30", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown exam status=%d", response.Code)
	}
}

func TestTopicViewOnlyMarksSeen(t *testing.T) {
	handler := testHandler(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/topics/example/view", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("view status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/progress/machine-learning", nil))
	if !strings.Contains(response.Body.String(), `"status":"seen"`) || strings.Contains(response.Body.String(), `"status":"mastered"`) {
		t.Fatalf("unexpected progress: %s", response.Body.String())
	}
}

func pdfHandler(t *testing.T) http.Handler {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	root := t.TempDir()
	pdfRoot := filepath.Join(root, "content", "pdfs")
	if err = os.MkdirAll(pdfRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"Course Notes.pdf", "Lecture(1).pdf"} {
		if err = os.WriteFile(filepath.Join(pdfRoot, file), []byte("%PDF-1.4\ntest\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog := &content.Catalog{
		Exams: []content.Exam{{ID: "machine-learning", Name: "ML"}, {ID: "deep-learning-data-science", Name: "DL"}},
		PDFs: []content.PDFSource{
			{File: "Course Notes.pdf", Title: "Course Notes", ExamID: "machine-learning", Available: true},
			{File: "Lecture(1).pdf", Title: "Lecture", ExamID: "deep-learning-data-science", Available: true},
			{File: "Missing.pdf", Title: "Missing", ExamID: "machine-learning"},
		},
	}
	return NewHandler(store, catalog, root, fstest.MapFS{"index.html": {Data: []byte("test")}})
}

func TestPDFEndpointServesKnownEncodedNamesAndRanges(t *testing.T) {
	handler := pdfHandler(t)
	for _, file := range []string{"Course Notes.pdf", "Lecture(1).pdf"} {
		request := httptest.NewRequest(http.MethodGet, "/pdf/"+url.PathEscape(file), nil)
		request.Header.Set("Range", "bytes=0-3")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusPartialContent || response.Body.String() != "%PDF" || response.Header().Get("Content-Type") != "application/pdf" {
			t.Fatalf("file=%q status=%d type=%q body=%q", file, response.Code, response.Header().Get("Content-Type"), response.Body.String())
		}
	}
}

func TestPDFEndpointRejectsUnknownMissingAndTraversal(t *testing.T) {
	handler := pdfHandler(t)
	for _, path := range []string{
		"/pdf/Unknown.pdf",
		"/pdf/Missing.pdf",
		"/pdf/..%2F..%2Fsecret.txt",
		"/pdf/%2e%2e%5csecret.txt",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("unknown, missing, or unsafe path %q returned status %d, want 404", path, response.Code)
		}
	}
}

func TestExamQuestionsWithholdSolutionsAndRespectScope(t *testing.T) {
	handler := testHandler(t)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=core", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"prompt":"Frage"`) {
		t.Fatalf("core status=%d body=%s", response.Code, body)
	}
	for _, forbidden := range []string{`"prompt":"Support-Frage"`, `"prompt":"DL-Frage"`, "disabled-question", "enabled-question", "correct_answer", "correct_answers", "correct_order", "correct_option_id", "explanation"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("core response leaked or mixed %q: %s", forbidden, body)
		}
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=all", nil))
	body = response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"prompt":"Support-Frage"`) || strings.Contains(body, `"prompt":"DL-Frage"`) {
		t.Fatalf("all scope status=%d body=%s", response.Code, body)
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=deep-learning-data-science&count=10&scope=all", nil))
	body = response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"prompt":"DL-Frage"`) || strings.Contains(body, `"prompt":"Support-Frage"`) || strings.Contains(body, "dl-question") {
		t.Fatalf("DL scope status=%d body=%s", response.Code, body)
	}
}

func TestProjectExamSimulationsProvideTenTwentyAndThirtyQuestions(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	catalog, err := content.Load(filepath.Join(projectRoot, "content"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store, catalog, projectRoot, fstest.MapFS{"index.html": {Data: []byte("test")}})
	for _, exam := range catalog.Exams {
		for _, count := range []int{10, 20, 30} {
			response := httptest.NewRecorder()
			target := fmt.Sprintf("/api/exam/questions?exam=%s&count=%d&scope=all", url.QueryEscape(exam.ID), count)
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
			var prepared struct {
				AttemptID string                `json:"attempt_id"`
				Questions []quiz.PublicQuestion `json:"questions"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &prepared); err != nil {
				t.Fatalf("exam=%s count=%d: %v body=%s", exam.ID, count, err, response.Body.String())
			}
			if response.Code != http.StatusOK || prepared.AttemptID == "" || len(prepared.Questions) != count {
				t.Fatalf("exam=%s count=%d status=%d attempt=%q questions=%d body=%s", exam.ID, count, response.Code, prepared.AttemptID, len(prepared.Questions), response.Body.String())
			}
		}
	}
}

func TestExamQuestionAliasCannotUseLearningAnswerEndpoint(t *testing.T) {
	handler, store, _ := testEnvironment(t)
	preparedResponse := httptest.NewRecorder()
	handler.ServeHTTP(preparedResponse, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=core", nil))
	var prepared struct {
		Questions []quiz.PublicQuestion `json:"questions"`
	}
	if err := json.Unmarshal(preparedResponse.Body.Bytes(), &prepared); err != nil {
		t.Fatal(err)
	}
	if len(prepared.Questions) == 0 {
		t.Fatal("exam returned no questions")
	}
	body, err := json.Marshal(map[string]any{"question_id": prepared.Questions[0].ID, "response": quiz.Answer{OptionIDs: []string{"a"}}})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, jsonRequest(http.MethodPost, "/api/answers", string(body)))
	if response.Code != http.StatusNotFound {
		t.Fatalf("opaque exam ID leaked into learning API: status=%d body=%s", response.Code, response.Body.String())
	}
	var reviews int
	if err = store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if reviews != 0 {
		t.Fatalf("learning endpoint stored %d exam reviews", reviews)
	}
}

func TestExamWeakScopeUsesCurrentMastery(t *testing.T) {
	handler := testHandler(t)
	wrong := httptest.NewRecorder()
	handler.ServeHTTP(wrong, jsonRequest(http.MethodPost, "/api/answers", `{"question_id":"support-question","response":{"option_ids":["no"]}}`))
	if wrong.Code != http.StatusOK {
		t.Fatalf("prepare weak status=%d body=%s", wrong.Code, wrong.Body.String())
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=weak", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"prompt":"Support-Frage"`) || strings.Contains(body, `"prompt":"Frage"`) {
		t.Fatalf("weak scope status=%d body=%s", response.Code, body)
	}
}

func TestExamFinishEvaluatesWithQuizEngineAndStoresAtFinish(t *testing.T) {
	handler, store, _ := testEnvironment(t)
	var before int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	questions := httptest.NewRecorder()
	handler.ServeHTTP(questions, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=all", nil))
	var prepared struct {
		AttemptID string                `json:"attempt_id"`
		Questions []quiz.PublicQuestion `json:"questions"`
	}
	if err := json.Unmarshal(questions.Body.Bytes(), &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.AttemptID == "" || len(prepared.Questions) != 5 {
		t.Fatalf("unexpected prepared exam: %+v body=%s", prepared, questions.Body.String())
	}
	var afterQuestions int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&afterQuestions); err != nil {
		t.Fatal(err)
	}
	if before != 0 || afterQuestions != 0 {
		t.Fatalf("loading exam questions persisted reviews: before=%d after=%d", before, afterQuestions)
	}

	answers := make([]map[string]any, 0, len(prepared.Questions))
	for _, question := range prepared.Questions {
		optionID := "a"
		if question.TopicID == "support-example" {
			optionID = "no"
		}
		answers = append(answers, map[string]any{"question_id": question.ID, "response": quiz.Answer{OptionIDs: []string{optionID}}})
	}
	submissionBytes, err := json.Marshal(map[string]any{"attempt_id": prepared.AttemptID, "exam_id": "machine-learning", "answers": answers})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, jsonRequest(http.MethodPost, "/api/exam/finish", string(submissionBytes)))
	if response.Code != http.StatusOK {
		t.Fatalf("finish status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Correct int `json:"correct"`
		Wrong   int `json:"wrong"`
		Total   int `json:"total"`
		Percent int `json:"percent"`
		Results []struct {
			QuestionID         string `json:"question_id"`
			PracticeQuestionID string `json:"practice_question_id"`
			Correct            bool   `json:"correct"`
			Explanation        string `json:"explanation"`
		} `json:"results"`
		Topics  []map[string]any `json:"topics"`
		Modules []map[string]any `json:"modules"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Correct != 4 || result.Wrong != 1 || result.Total != 5 || result.Percent != 80 || len(result.Results) != 5 || len(result.Topics) != 2 || len(result.Modules) != 2 {
		t.Fatalf("unexpected exam result: %+v body=%s", result, response.Body.String())
	}
	for _, item := range result.Results {
		if item.Explanation == "" || item.PracticeQuestionID == "" {
			t.Fatalf("finish did not return final feedback: %+v", result.Results)
		}
	}
	var reviews int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if reviews != 5 {
		t.Fatalf("reviews=%d want=5", reviews)
	}
}

func TestExamAttemptSurvivesHandlerRestartAndCannotBeReused(t *testing.T) {
	handler, store, catalog := testEnvironment(t)
	preparedResponse := httptest.NewRecorder()
	handler.ServeHTTP(preparedResponse, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=all", nil))
	var prepared struct {
		AttemptID string                `json:"attempt_id"`
		Questions []quiz.PublicQuestion `json:"questions"`
	}
	if err := json.Unmarshal(preparedResponse.Body.Bytes(), &prepared); err != nil {
		t.Fatal(err)
	}
	answers := make([]map[string]any, 0, len(prepared.Questions))
	for _, question := range prepared.Questions {
		optionID := "a"
		if question.TopicID == "support-example" {
			optionID = "yes"
		}
		answers = append(answers, map[string]any{"question_id": question.ID, "response": quiz.Answer{OptionIDs: []string{optionID}}})
	}
	submission, err := json.Marshal(map[string]any{"attempt_id": prepared.AttemptID, "exam_id": "machine-learning", "answers": answers})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewHandler(store, catalog, t.TempDir(), fstest.MapFS{"index.html": {Data: []byte("test")}})
	first := httptest.NewRecorder()
	restarted.ServeHTTP(first, jsonRequest(http.MethodPost, "/api/exam/finish", string(submission)))
	if first.Code != http.StatusOK {
		t.Fatalf("finish after handler restart status=%d body=%s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	restarted.ServeHTTP(second, jsonRequest(http.MethodPost, "/api/exam/finish", string(submission)))
	if second.Code == http.StatusOK {
		t.Fatalf("exam attempt was accepted twice: %s", second.Body.String())
	}
}

func TestExamFinishRejectsMixedExamAndDuplicateQuestions(t *testing.T) {
	for _, name := range []string{"mixed exam", "duplicate", "incomplete"} {
		t.Run(name, func(t *testing.T) {
			handler, store, _ := testEnvironment(t)
			preparedResponse := httptest.NewRecorder()
			handler.ServeHTTP(preparedResponse, httptest.NewRequest(http.MethodGet, "/api/exam/questions?exam=machine-learning&count=10&scope=all", nil))
			var prepared struct {
				AttemptID string                `json:"attempt_id"`
				Questions []quiz.PublicQuestion `json:"questions"`
			}
			if err := json.Unmarshal(preparedResponse.Body.Bytes(), &prepared); err != nil {
				t.Fatal(err)
			}
			answers := make([]map[string]any, 0, len(prepared.Questions))
			for _, question := range prepared.Questions {
				answers = append(answers, map[string]any{"question_id": question.ID, "response": quiz.Answer{OptionIDs: []string{"a"}}})
			}
			switch name {
			case "mixed exam":
				answers[0]["question_id"] = "dl-question"
			case "duplicate":
				answers[1]["question_id"] = answers[0]["question_id"]
			case "incomplete":
				answers = answers[:len(answers)-1]
			}
			submission, err := json.Marshal(map[string]any{"attempt_id": prepared.AttemptID, "exam_id": "machine-learning", "answers": answers})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, jsonRequest(http.MethodPost, "/api/exam/finish", string(submission)))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var reviews int
			if err := store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&reviews); err != nil {
				t.Fatal(err)
			}
			if reviews != 0 {
				t.Fatalf("invalid submission stored %d reviews", reviews)
			}
		})
	}
}

func TestBackupRestoreAPIExportsAndRestoresState(t *testing.T) {
	handler, store, _ := testEnvironment(t)
	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, jsonRequest(http.MethodPost, "/api/answers", `{"question_id":"enabled-question","response":{"option_ids":["a"]}}`))
	if answer.Code != http.StatusOK {
		t.Fatalf("prepare answer status=%d body=%s", answer.Code, answer.Body.String())
	}

	download := httptest.NewRecorder()
	handler.ServeHTTP(download, httptest.NewRequest(http.MethodGet, "/api/backup", nil))
	if download.Code != http.StatusOK || !strings.Contains(download.Header().Get("Content-Disposition"), "mldl-exam-trainer-backup-") {
		t.Fatalf("backup status=%d disposition=%q body=%s", download.Code, download.Header().Get("Content-Disposition"), download.Body.String())
	}
	backup, err := db.DecodeBackup(strings.NewReader(download.Body.String()))
	if err != nil || backup.Data == nil || len(backup.Data.Reviews) != 1 {
		t.Fatalf("exported backup=%+v err=%v", backup, err)
	}
	if err := store.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}

	restore := httptest.NewRecorder()
	handler.ServeHTTP(restore, jsonRequest(http.MethodPost, "/api/restore", download.Body.String()))
	if restore.Code != http.StatusOK || !strings.Contains(restore.Body.String(), `"status":"ok"`) {
		t.Fatalf("restore status=%d body=%s", restore.Code, restore.Body.String())
	}
	theme, err := store.Theme()
	if err != nil || theme != "light" {
		t.Fatalf("restored theme=%q err=%v", theme, err)
	}
	var reviews int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&reviews); err != nil || reviews != 1 {
		t.Fatalf("restored reviews=%d err=%v", reviews, err)
	}
}

func TestRestoreAPIRejectsInvalidFormatVersionAndJSON(t *testing.T) {
	handler, store, _ := testEnvironment(t)
	valid, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"invalid JSON": `{broken`,
		"format":       strings.Replace(mustJSON(t, valid), db.BackupFormat, "other-format", 1),
		"version":      strings.Replace(mustJSON(t, valid), `"version":1`, `"version":99`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, jsonRequest(http.MethodPost, "/api/restore", body))
			if response.Code != http.StatusBadRequest || strings.Contains(strings.ToLower(response.Body.String()), "sqlite") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestJSONEndpointsRequireContentTypeAndLimitBody(t *testing.T) {
	handler := testHandler(t)
	missingType := httptest.NewRecorder()
	handler.ServeHTTP(missingType, httptest.NewRequest(http.MethodPost, "/api/settings/theme", strings.NewReader(`{"theme":"dark"}`)))
	if missingType.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("missing content type status=%d", missingType.Code)
	}

	tooLargeRequest := httptest.NewRequest(http.MethodPost, "/api/restore", strings.NewReader(`{"padding":"`+strings.Repeat("x", 8*1024*1024)+`"}`))
	tooLargeRequest.Header.Set("Content-Type", "application/json")
	tooLarge := httptest.NewRecorder()
	handler.ServeHTTP(tooLarge, tooLargeRequest)
	if tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized restore status=%d body=%s", tooLarge.Code, tooLarge.Body.String())
	}

	tooLargeAnswer := httptest.NewRecorder()
	handler.ServeHTTP(tooLargeAnswer, jsonRequest(http.MethodPost, "/api/answers", `{"padding":"`+strings.Repeat("x", 16*1024)+`"}`))
	if tooLargeAnswer.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized answer status=%d body=%s", tooLargeAnswer.Code, tooLargeAnswer.Body.String())
	}

	for _, body := range []string{`{"theme":`, `{"theme":"dark"}{}`, `{"theme":"dark","extra":true}`} {
		invalid := httptest.NewRecorder()
		handler.ServeHTTP(invalid, jsonRequest(http.MethodPost, "/api/settings/theme", body))
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid JSON %q status=%d body=%s", body, invalid.Code, invalid.Body.String())
		}
	}

	wrongMethod := httptest.NewRecorder()
	handler.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodGet, "/api/restore", nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status=%d", wrongMethod.Code)
	}
}

func TestUnknownAPIEndpointReturnsFriendlyJSON404(t *testing.T) {
	response := httptest.NewRecorder()
	testHandler(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Header().Get("Content-Type"), "application/json") || !strings.Contains(response.Body.String(), "Unbekannter API-Endpunkt") {
		t.Fatalf("status=%d content-type=%q body=%s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
}

func TestSecurityHeadersCoverAPIAndCrossSiteWritesAreBlocked(t *testing.T) {
	handler := testHandler(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	for _, header := range []string{"X-Content-Type-Options", "Referrer-Policy", "Content-Security-Policy", "X-Frame-Options", "Permissions-Policy"} {
		if response.Header().Get(header) == "" {
			t.Fatalf("API response missing %s", header)
		}
	}
	if response.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("local HTTP app must not send HSTS")
	}
	if csp := response.Header().Get("Content-Security-Policy"); strings.Contains(csp, "unsafe-inline") || !strings.Contains(csp, "default-src 'self'") {
		t.Fatalf("unexpected CSP %q", csp)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("local app must not expose open CORS")
	}

	crossSite := jsonRequest(http.MethodPost, "/api/settings/theme", `{"theme":"dark"}`)
	crossSite.Header.Set("Origin", "http://attacker.example")
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, crossSite)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("cross-site write status=%d body=%s", blocked.Code, blocked.Body.String())
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
