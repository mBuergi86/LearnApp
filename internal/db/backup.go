package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/progress"
)

const (
	BackupFormat  = "mldl-exam-trainer-backup"
	BackupVersion = 1
)

type Backup struct {
	Format         string      `json:"format"`
	Version        int         `json:"version"`
	CreatedAt      string      `json:"created_at"`
	DatabaseSchema int         `json:"database_schema"`
	Data           *BackupData `json:"data"`
}

type BackupData struct {
	Settings         []BackupSetting          `json:"settings"`
	Reviews          []BackupReview           `json:"reviews"`
	QuestionProgress []BackupQuestionProgress `json:"question_progress"`
	TopicProgress    []BackupTopicProgress    `json:"topic_progress"`
	ExamAttempts     []BackupExamAttempt      `json:"exam_attempts"`
	StudySessions    []BackupStudySession     `json:"study_sessions"`
	ExamAnswers      []BackupExamAnswer       `json:"exam_answers"`
	SchedulerState   *BackupSchedulerState    `json:"scheduler_state"`
}

type BackupSetting struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

type BackupReview struct {
	ID           int64           `json:"id"`
	QuestionID   string          `json:"question_id"`
	AnsweredAt   string          `json:"answered_at"`
	Correct      bool            `json:"correct"`
	ResponseData json.RawMessage `json:"response_data"`
}

type BackupQuestionProgress struct {
	QuestionID         string  `json:"question_id"`
	CorrectCount       int     `json:"correct_count"`
	WrongCount         int     `json:"wrong_count"`
	ConsecutiveCorrect int     `json:"consecutive_correct"`
	ConsecutiveWrong   int     `json:"consecutive_wrong"`
	LastAnsweredAt     *string `json:"last_answered_at"`
	LastCorrect        bool    `json:"last_correct"`
	NextReviewAt       *string `json:"next_review_at"`
	IntervalStage      int     `json:"interval_stage"`
	SuccessfulDays     int     `json:"successful_days"`
	LastSuccessDate    string  `json:"last_success_date"`
}

type BackupTopicProgress struct {
	TopicID       string  `json:"topic_id"`
	MasteryStatus string  `json:"mastery_status"`
	MasteryScore  float64 `json:"mastery_score"`
	OpenedAt      *string `json:"opened_at"`
	LastReviewed  *string `json:"last_reviewed_at"`
	NextReview    *string `json:"next_review_at"`
}

type BackupExamAttempt struct {
	ID              string            `json:"id"`
	ExamID          string            `json:"exam_id"`
	CreatedAt       string            `json:"created_at"`
	QuestionAliases map[string]string `json:"question_aliases"`
}

// The learning-data schema does not persist completed session or exam aggregates separately.
// The required V1 keys remain explicit and empty; completed answers are Reviews.
type BackupStudySession struct{}
type BackupExamAnswer struct{}

type BackupSchedulerState struct {
	AppliedReviews []BackupAppliedReview `json:"applied_reviews"`
}

type BackupAppliedReview struct {
	ReviewID  int64  `json:"review_id"`
	AppliedAt string `json:"applied_at"`
}

type backupQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func DecodeBackup(reader io.Reader) (Backup, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var backup Backup
	if err := decoder.Decode(&backup); err != nil {
		return Backup{}, fmt.Errorf("Backup ist kein gültiges JSON-Dokument: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Backup{}, fmt.Errorf("Backup enthält unerwartete weitere Daten")
	}
	return backup, nil
}

func (s *Store) SchemaVersion() (int, error) {
	return schemaVersion(context.Background(), s.DB)
}

func schemaVersion(ctx context.Context, queryer backupQueryer) (int, error) {
	var raw string
	if err := queryer.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key='schema_version'`).Scan(&raw); err != nil {
		return 0, fmt.Errorf("Schemaversion lesen: %w", err)
	}
	version, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("ungültige Schemaversion")
	}
	return version, nil
}

func (s *Store) ExportBackup() (Backup, error) {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Backup{}, fmt.Errorf("Backup starten: %w", err)
	}
	defer tx.Rollback()
	backup, err := s.exportBackup(ctx, tx)
	if err != nil {
		return Backup{}, err
	}
	if err := tx.Commit(); err != nil {
		return Backup{}, fmt.Errorf("Backup abschliessen: %w", err)
	}
	return backup, nil
}

func (s *Store) exportBackup(ctx context.Context, queryer backupQueryer) (Backup, error) {
	version, err := schemaVersion(ctx, queryer)
	if err != nil {
		return Backup{}, err
	}
	data := &BackupData{
		Settings:         make([]BackupSetting, 0),
		Reviews:          make([]BackupReview, 0),
		QuestionProgress: make([]BackupQuestionProgress, 0),
		TopicProgress:    make([]BackupTopicProgress, 0),
		ExamAttempts:     make([]BackupExamAttempt, 0),
		StudySessions:    make([]BackupStudySession, 0),
		ExamAnswers:      make([]BackupExamAnswer, 0),
		SchedulerState:   &BackupSchedulerState{AppliedReviews: make([]BackupAppliedReview, 0)},
	}
	backup := Backup{Format: BackupFormat, Version: BackupVersion, CreatedAt: s.Clock.Now().UTC().Format(time.RFC3339Nano), DatabaseSchema: version, Data: data}

	if err := readBackupRows(ctx, queryer, `SELECT key,value,updated_at FROM settings ORDER BY key`, func(rows *sql.Rows) error {
		var item BackupSetting
		if err := rows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return err
		}
		data.Settings = append(data.Settings, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Einstellungen exportieren: %w", err)
	}
	if err := readBackupRows(ctx, queryer, `SELECT id,question_id,answered_at,correct,response_data FROM reviews ORDER BY id`, func(rows *sql.Rows) error {
		var item BackupReview
		var correct int
		var response string
		if err := rows.Scan(&item.ID, &item.QuestionID, &item.AnsweredAt, &correct, &response); err != nil {
			return err
		}
		item.Correct = correct == 1
		item.ResponseData = json.RawMessage(response)
		data.Reviews = append(data.Reviews, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Reviews exportieren: %w", err)
	}
	if err := readBackupRows(ctx, queryer, `SELECT question_id,correct_count,wrong_count,consecutive_correct,consecutive_wrong,last_answered_at,last_correct,next_review_at,interval_stage,successful_days,last_success_date FROM question_progress ORDER BY question_id`, func(rows *sql.Rows) error {
		var item BackupQuestionProgress
		var lastAnswered, nextReview sql.NullString
		var lastCorrect int
		if err := rows.Scan(&item.QuestionID, &item.CorrectCount, &item.WrongCount, &item.ConsecutiveCorrect, &item.ConsecutiveWrong, &lastAnswered, &lastCorrect, &nextReview, &item.IntervalStage, &item.SuccessfulDays, &item.LastSuccessDate); err != nil {
			return err
		}
		item.LastAnsweredAt = stringPointer(lastAnswered)
		item.LastCorrect = lastCorrect == 1
		item.NextReviewAt = stringPointer(nextReview)
		data.QuestionProgress = append(data.QuestionProgress, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Fragenfortschritt exportieren: %w", err)
	}
	if err := readBackupRows(ctx, queryer, `SELECT topic_id,mastery_status,mastery_score,opened_at,last_reviewed_at,next_review_at FROM topic_progress ORDER BY topic_id`, func(rows *sql.Rows) error {
		var item BackupTopicProgress
		var opened, last, next sql.NullString
		if err := rows.Scan(&item.TopicID, &item.MasteryStatus, &item.MasteryScore, &opened, &last, &next); err != nil {
			return err
		}
		item.OpenedAt, item.LastReviewed, item.NextReview = stringPointer(opened), stringPointer(last), stringPointer(next)
		data.TopicProgress = append(data.TopicProgress, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Topic-Fortschritt exportieren: %w", err)
	}
	if err := readBackupRows(ctx, queryer, `SELECT id,exam_id,created_at,questions_json FROM exam_attempts ORDER BY id`, func(rows *sql.Rows) error {
		var item BackupExamAttempt
		var aliases string
		if err := rows.Scan(&item.ID, &item.ExamID, &item.CreatedAt, &aliases); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(aliases), &item.QuestionAliases); err != nil {
			return fmt.Errorf("Prüfungsfragen lesen: %w", err)
		}
		data.ExamAttempts = append(data.ExamAttempts, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Prüfungsversuche exportieren: %w", err)
	}
	if err := readBackupRows(ctx, queryer, `SELECT review_id,applied_at FROM progress_applied_reviews ORDER BY review_id`, func(rows *sql.Rows) error {
		var item BackupAppliedReview
		if err := rows.Scan(&item.ReviewID, &item.AppliedAt); err != nil {
			return err
		}
		data.SchedulerState.AppliedReviews = append(data.SchedulerState.AppliedReviews, item)
		return nil
	}); err != nil {
		return Backup{}, fmt.Errorf("Scheduler-Zustand exportieren: %w", err)
	}
	return backup, nil
}

func readBackupRows(ctx context.Context, queryer backupQueryer, query string, scan func(*sql.Rows) error) error {
	rows, err := queryer.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func stringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}

func (s *Store) ValidateBackup(backup Backup, catalog *content.Catalog) error {
	version, err := s.SchemaVersion()
	if err != nil {
		return err
	}
	return validateBackup(backup, catalog, version)
}

func validateBackup(backup Backup, catalog *content.Catalog, databaseSchema int) error {
	if backup.Format != BackupFormat {
		return fmt.Errorf("unbekanntes Backupformat")
	}
	if backup.Version != BackupVersion {
		return fmt.Errorf("Backupversion %d wird nicht unterstützt", backup.Version)
	}
	if backup.DatabaseSchema != databaseSchema {
		return fmt.Errorf("Backup benötigt Datenbankschema %d statt %d", backup.DatabaseSchema, databaseSchema)
	}
	if _, err := time.Parse(time.RFC3339Nano, backup.CreatedAt); err != nil {
		return fmt.Errorf("created_at ist ungültig")
	}
	data := backup.Data
	if data == nil || data.Settings == nil || data.Reviews == nil || data.QuestionProgress == nil || data.TopicProgress == nil || data.ExamAttempts == nil || data.StudySessions == nil || data.ExamAnswers == nil || data.SchedulerState == nil || data.SchedulerState.AppliedReviews == nil {
		return fmt.Errorf("Backup enthält nicht alle Pflichtfelder")
	}
	if len(data.StudySessions) != 0 || len(data.ExamAnswers) != 0 {
		return fmt.Errorf("separat gespeicherte Sessions oder Prüfungsantworten werden in Version 1 nicht unterstützt")
	}

	exams := make(map[string]bool, len(catalog.Exams))
	topics := make(map[string]bool, len(catalog.Topics))
	questions := make(map[string]content.Question, len(catalog.Questions))
	for _, exam := range catalog.Exams {
		exams[exam.ID] = true
	}
	for _, topic := range catalog.Topics {
		topics[topic.ID] = true
	}
	for _, question := range catalog.Questions {
		questions[question.ID] = question
	}

	settings := map[string]bool{}
	for _, item := range data.Settings {
		if item.Key != "theme" || settings[item.Key] || !validTimestamp(item.UpdatedAt) || (item.Value != "light" && item.Value != "dark" && item.Value != "system") {
			return fmt.Errorf("Einstellung %q ist ungültig", item.Key)
		}
		settings[item.Key] = true
	}
	if !settings["theme"] {
		return fmt.Errorf("Theme-Einstellung fehlt")
	}

	reviews := map[int64]bool{}
	for _, item := range data.Reviews {
		if item.ID <= 0 || reviews[item.ID] || questions[item.QuestionID].ID == "" || !json.Valid(item.ResponseData) || !validTimestamp(item.AnsweredAt) {
			return fmt.Errorf("Review %d ist ungültig oder verweist auf unbekannten Content", item.ID)
		}
		reviews[item.ID] = true
	}
	questionProgress := map[string]bool{}
	for _, item := range data.QuestionProgress {
		if questions[item.QuestionID].ID == "" || questionProgress[item.QuestionID] || minInt(item.CorrectCount, item.WrongCount, item.ConsecutiveCorrect, item.ConsecutiveWrong, item.IntervalStage, item.SuccessfulDays) < 0 || item.IntervalStage > len(progress.Intervals) || !validOptionalTimestamp(item.LastAnsweredAt) || !validOptionalTimestamp(item.NextReviewAt) || !validDate(item.LastSuccessDate) {
			return fmt.Errorf("Fragenfortschritt für %q ist ungültig", item.QuestionID)
		}
		questionProgress[item.QuestionID] = true
	}
	topicProgress := map[string]bool{}
	for _, item := range data.TopicProgress {
		status := progress.Status(item.MasteryStatus)
		if !topics[item.TopicID] || topicProgress[item.TopicID] || (status != progress.Unlearned && status != progress.Seen && status != progress.Practicing && status != progress.Mastered && status != progress.Weak) || item.MasteryScore < 0 || item.MasteryScore > 1 || !validOptionalTimestamp(item.OpenedAt) || !validOptionalTimestamp(item.LastReviewed) || !validOptionalTimestamp(item.NextReview) {
			return fmt.Errorf("Topic-Fortschritt für %q ist ungültig", item.TopicID)
		}
		topicProgress[item.TopicID] = true
	}
	attempts := map[string]bool{}
	for _, item := range data.ExamAttempts {
		if strings.TrimSpace(item.ID) == "" || attempts[item.ID] || !exams[item.ExamID] || !validTimestamp(item.CreatedAt) || len(item.QuestionAliases) == 0 || len(item.QuestionAliases) > 30 {
			return fmt.Errorf("Prüfungsversuch %q ist ungültig", item.ID)
		}
		attemptQuestions := map[string]bool{}
		for alias, questionID := range item.QuestionAliases {
			question, found := questions[questionID]
			topic, topicFound := content.FindTopic(catalog, question.TopicID)
			if strings.TrimSpace(alias) == "" || !found || !topicFound || topic.ExamID != item.ExamID || attemptQuestions[questionID] {
				return fmt.Errorf("Prüfungsversuch %q verweist auf unbekannten Content", item.ID)
			}
			attemptQuestions[questionID] = true
		}
		attempts[item.ID] = true
	}
	applied := map[int64]bool{}
	for _, item := range data.SchedulerState.AppliedReviews {
		if !reviews[item.ReviewID] || applied[item.ReviewID] || !validTimestamp(item.AppliedAt) {
			return fmt.Errorf("Scheduler-Eintrag für Review %d ist ungültig", item.ReviewID)
		}
		applied[item.ReviewID] = true
	}
	return nil
}

func minInt(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func validTimestamp(value string) bool {
	if _, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return true
	}
	_, err := time.Parse("2006-01-02 15:04:05", value)
	return err == nil
}

func validOptionalTimestamp(value *string) bool {
	return value == nil || validTimestamp(*value)
}

func validDate(value string) bool {
	if value == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func (s *Store) RestoreBackup(backup Backup, catalog *content.Catalog, safetyDirectory string) (string, error) {
	if err := s.ValidateBackup(backup, catalog); err != nil {
		return "", err
	}
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("Restore starten: %w", err)
	}
	defer tx.Rollback()
	current, err := s.exportBackup(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("Sicherheitskopie erstellen: %w", err)
	}
	safetyPath, err := writeSafetyBackup(safetyDirectory, current)
	if err != nil {
		return "", fmt.Errorf("Sicherheitskopie speichern: %w", err)
	}
	if err := restoreBackupRows(ctx, tx, backup); err != nil {
		return safetyPath, fmt.Errorf("Backup wiederherstellen: %w", err)
	}
	if err := checkIntegrity(ctx, tx, catalog); err != nil {
		return safetyPath, fmt.Errorf("wiederhergestellte Daten sind inkonsistent: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return safetyPath, fmt.Errorf("Restore abschliessen: %w", err)
	}
	return safetyPath, nil
}

func restoreBackupRows(ctx context.Context, tx *sql.Tx, backup Backup) error {
	for _, statement := range []string{
		`DELETE FROM progress_applied_reviews`,
		`DELETE FROM exam_attempts`,
		`DELETE FROM topic_progress`,
		`DELETE FROM question_progress`,
		`DELETE FROM reviews`,
		`DELETE FROM settings`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.Settings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?)`, item.Key, item.Value, item.UpdatedAt); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.Reviews {
		if _, err := tx.ExecContext(ctx, `INSERT INTO reviews(id,question_id,answered_at,correct,response_data) VALUES(?,?,?,?,?)`, item.ID, item.QuestionID, item.AnsweredAt, item.Correct, string(item.ResponseData)); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.QuestionProgress {
		if _, err := tx.ExecContext(ctx, `INSERT INTO question_progress(question_id,correct_count,wrong_count,consecutive_correct,consecutive_wrong,last_answered_at,last_correct,next_review_at,interval_stage,successful_days,last_success_date) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.QuestionID, item.CorrectCount, item.WrongCount, item.ConsecutiveCorrect, item.ConsecutiveWrong, nullableString(item.LastAnsweredAt), item.LastCorrect, nullableString(item.NextReviewAt), item.IntervalStage, item.SuccessfulDays, item.LastSuccessDate); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.TopicProgress {
		if _, err := tx.ExecContext(ctx, `INSERT INTO topic_progress(topic_id,mastery_status,mastery_score,opened_at,last_reviewed_at,next_review_at) VALUES(?,?,?,?,?,?)`, item.TopicID, item.MasteryStatus, item.MasteryScore, nullableString(item.OpenedAt), nullableString(item.LastReviewed), nullableString(item.NextReview)); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.ExamAttempts {
		aliases, err := json.Marshal(item.QuestionAliases)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO exam_attempts(id,exam_id,created_at,questions_json) VALUES(?,?,?,?)`, item.ID, item.ExamID, item.CreatedAt, string(aliases)); err != nil {
			return err
		}
	}
	for _, item := range backup.Data.SchedulerState.AppliedReviews {
		if _, err := tx.ExecContext(ctx, `INSERT INTO progress_applied_reviews(review_id,applied_at) VALUES(?,?)`, item.ReviewID, item.AppliedAt); err != nil {
			return err
		}
	}
	return nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func writeSafetyBackup(directory string, backup Backup) (string, error) {
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("Sicherungsverzeichnis fehlt")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return "", err
	}
	base := "pre-restore-" + strings.NewReplacer(":", "", ".", "-").Replace(backup.CreatedAt)
	for suffix := 0; suffix < 100; suffix++ {
		name := base + ".json"
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d.json", base, suffix)
		}
		path := filepath.Join(directory, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err = file.Write(append(data, '\n')); err != nil {
			file.Close()
			return "", err
		}
		if err = file.Close(); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("kein freier Dateiname für Sicherheitskopie")
}

func (s *Store) CheckIntegrity(catalog *content.Catalog) error {
	return checkIntegrity(context.Background(), s.DB, catalog)
}

func checkIntegrity(ctx context.Context, queryer backupQueryer, catalog *content.Catalog) error {
	rows, err := queryer.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Close()
		return fmt.Errorf("Foreign-Key-Verletzung erkannt")
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	questions := make(map[string]bool, len(catalog.Questions))
	topics := make(map[string]bool, len(catalog.Topics))
	exams := make(map[string]bool, len(catalog.Exams))
	for _, question := range catalog.Questions {
		questions[question.ID] = true
	}
	for _, topic := range catalog.Topics {
		topics[topic.ID] = true
	}
	for _, exam := range catalog.Exams {
		exams[exam.ID] = true
	}
	for _, check := range []struct {
		query string
		known map[string]bool
		name  string
	}{
		{`SELECT DISTINCT question_id FROM reviews`, questions, "Review"},
		{`SELECT question_id FROM question_progress`, questions, "Fragenfortschritt"},
		{`SELECT topic_id FROM topic_progress`, topics, "Topic-Fortschritt"},
		{`SELECT exam_id FROM exam_attempts`, exams, "Prüfungsversuch"},
	} {
		if err := readBackupRows(ctx, queryer, check.query, func(rows *sql.Rows) error {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			if !check.known[id] {
				return fmt.Errorf("%s verweist auf unbekannte ID %q", check.name, id)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	for _, topic := range catalog.Topics {
		moduleFound := false
		for _, module := range catalog.Modules {
			if module.ID == topic.ModuleID && module.ExamID == topic.ExamID {
				moduleFound = true
				break
			}
		}
		if !moduleFound {
			return fmt.Errorf("Topic %q verweist auf unbekanntes Modul", topic.ID)
		}
	}
	for _, question := range catalog.Questions {
		if !topics[question.TopicID] {
			return fmt.Errorf("Frage %q verweist auf unbekanntes Topic", question.ID)
		}
	}
	for _, module := range catalog.Modules {
		if !exams[module.ExamID] {
			return fmt.Errorf("Modul %q verweist auf unbekannte Prüfung", module.ID)
		}
	}
	return nil
}
