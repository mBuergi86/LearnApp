package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/quiz"
)

func backupTestCatalog() *content.Catalog {
	return &content.Catalog{
		Exams:   []content.Exam{{ID: "machine-learning", Name: "Machine Learning"}, {ID: "deep-learning-data-science", Name: "Deep Learning"}},
		Modules: []content.Module{{ID: "ml-module", ExamID: "machine-learning", Name: "ML"}, {ID: "dl-module", ExamID: "deep-learning-data-science", Name: "DL"}},
		Topics: []content.Topic{
			{ID: "ml-topic", ExamID: "machine-learning", ModuleID: "ml-module", Title: "ML Topic", Priority: content.PriorityCore, Summary: "ML"},
			{ID: "dl-topic", ExamID: "deep-learning-data-science", ModuleID: "dl-module", Title: "DL Topic", Priority: content.PriorityCore, Summary: "DL"},
		},
		Questions: []content.Question{
			{ID: "ml-question", TopicID: "ml-topic", Enabled: true},
			{ID: "dl-question", TopicID: "dl-topic", Enabled: true},
		},
	}
}

func populatedBackupStore(t *testing.T) (*Store, *content.Catalog) {
	t.Helper()
	now := time.Date(2026, 9, 6, 12, 0, 0, 123, time.UTC)
	store, err := OpenWithClock(":memory:", fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	catalog := backupTestCatalog()
	if err := store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if err := store.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordAnswer(catalog, catalog.Questions[0], true, quiz.Answer{OptionIDs: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`UPDATE topic_progress SET mastery_status='mastered',mastery_score=1 WHERE topic_id='ml-topic'`); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveExamAttempt(ExamAttempt{ID: "attempt-one", ExamID: "machine-learning", CreatedAt: now, QuestionAliases: map[string]string{"public-one": "ml-question"}}); err != nil {
		t.Fatal(err)
	}
	return store, catalog
}

func TestExportBackupContainsCompleteStoredLearningState(t *testing.T) {
	store, _ := populatedBackupStore(t)
	backup, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if backup.Format != BackupFormat || backup.Version != BackupVersion || backup.DatabaseSchema != 7 || backup.Data == nil {
		t.Fatalf("backup header=%+v", backup)
	}
	if len(backup.Data.Settings) != 1 || backup.Data.Settings[0].Value != "dark" || len(backup.Data.Reviews) != 1 || len(backup.Data.QuestionProgress) != 1 || len(backup.Data.TopicProgress) != 1 || len(backup.Data.ExamAttempts) != 1 || len(backup.Data.SchedulerState.AppliedReviews) != 1 {
		t.Fatalf("incomplete backup data=%+v", backup.Data)
	}
	if backup.Data.StudySessions == nil || backup.Data.ExamAnswers == nil {
		t.Fatal("required V1 collections are missing")
	}
	encoded, err := json.Marshal(backup)
	if err != nil || !json.Valid(encoded) {
		t.Fatalf("invalid exported JSON: %v", err)
	}
	decoded, err := DecodeBackup(strings.NewReader(string(encoded)))
	if err != nil || decoded.Format != BackupFormat {
		t.Fatalf("DecodeBackup()=%+v err=%v", decoded, err)
	}
}

func TestBackupRestoreRoundTripRestoresReviewsMasteryAttemptsAndSettings(t *testing.T) {
	store, catalog := populatedBackupStore(t)
	want, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordAnswer(catalog, catalog.Questions[0], false, quiz.Answer{OptionIDs: []string{"b"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DeleteExamAttempt("attempt-one"); err != nil {
		t.Fatal(err)
	}
	safetyDirectory := filepath.Join(t.TempDir(), "backups")
	safetyPath, err := store.RestoreBackup(want, catalog, safetyDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(safetyPath); err != nil || info.Size() == 0 {
		t.Fatalf("safety backup missing: path=%q err=%v", safetyPath, err)
	}
	got, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	want.CreatedAt, got.CreatedAt = "", ""
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("roundtrip mismatch\nwant=%+v\n got=%+v", want, got)
	}
	if got.Data.TopicProgress[0].MasteryStatus != "mastered" || got.Data.Settings[0].Value != "dark" || got.Data.ExamAttempts[0].ID != "attempt-one" {
		t.Fatalf("important state not restored: %+v", got.Data)
	}
}

func TestContentUpgradePreservesHistorySettingsAndBackupCompatibility(t *testing.T) {
	store, catalog := populatedBackupStore(t)
	before, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	// Stable question/topic IDs allow existing personal state to survive a
	// didactic revision, including a question changed to a numeric exercise.
	catalog.Topics[0].Explanation = []string{"Expanded explanation with a worked calculation"}
	catalog.Questions[0].Type = content.QuestionNumeric
	catalog.Questions[0].Options = []content.QuestionOption{{ID: "new-one", Text: "2"}, {ID: "new-two", Text: "4"}}
	catalog.Questions[0].CorrectAnswer = "new-two"
	if err := store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	after, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("content sync changed personal history, mastery, scheduler, attempts or theme")
	}
	if err := store.ValidateBackup(before, catalog); err != nil {
		t.Fatalf("previous backup no longer valid: %v", err)
	}
	if _, err := store.RestoreBackup(before, catalog, filepath.Join(t.TempDir(), "safety")); err != nil {
		t.Fatal(err)
	}
	restored, err := store.ExportBackup()
	if err != nil || !reflect.DeepEqual(before.Data, restored.Data) {
		t.Fatalf("personal state did not round-trip after content upgrade: %v", err)
	}
	version, err := store.SchemaVersion()
	if err != nil || version != 7 {
		t.Fatalf("schema changed: %d %v", version, err)
	}
}

func TestDecodeBackupRejectsInvalidJSONAndUnknownFields(t *testing.T) {
	for name, document := range map[string]string{
		"invalid JSON":  `{broken`,
		"unknown field": `{"format":"mldl-exam-trainer-backup","surprise":true}`,
		"trailing data": `{}` + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeBackup(strings.NewReader(document)); err == nil {
				t.Fatalf("DecodeBackup accepted %s", document)
			}
		})
	}
}

func TestValidateBackupRejectsFormatVersionMissingFieldsAndUnknownIDs(t *testing.T) {
	store, catalog := populatedBackupStore(t)
	valid, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*Backup){
		"format":         func(item *Backup) { item.Format = "other" },
		"version":        func(item *Backup) { item.Version++ },
		"schema":         func(item *Backup) { item.DatabaseSchema++ },
		"required field": func(item *Backup) { item.Data.QuestionProgress = nil },
		"unknown ID":     func(item *Backup) { item.Data.Reviews[0].QuestionID = "unknown-question" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(valid)
			if err != nil {
				t.Fatal(err)
			}
			var item Backup
			if err := json.Unmarshal(encoded, &item); err != nil {
				t.Fatal(err)
			}
			mutate(&item)
			if err := store.ValidateBackup(item, catalog); err == nil {
				t.Fatal("invalid backup was accepted")
			}
		})
	}
}

func TestFailedRestoreRollsBackEveryChange(t *testing.T) {
	store, catalog := populatedBackupStore(t)
	backup, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetTheme("light"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(`CREATE TRIGGER reject_restored_review BEFORE INSERT ON reviews BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	before, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreBackup(backup, catalog, filepath.Join(t.TempDir(), "backups")); err == nil {
		t.Fatal("RestoreBackup unexpectedly succeeded")
	}
	after, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	before.CreatedAt, after.CreatedAt = "", ""
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed restore changed state\nbefore=%+v\n after=%+v", before, after)
	}
}

func TestCheckIntegrityDetectsUnknownPersonalContentReferences(t *testing.T) {
	store, catalog := populatedBackupStore(t)
	if err := store.CheckIntegrity(catalog); err != nil {
		t.Fatalf("valid state failed integrity check: %v", err)
	}
	if _, err := store.DB.Exec(`INSERT INTO reviews(question_id,answered_at,correct,response_data) VALUES('unknown','2026-09-06T12:00:00Z',1,'{}')`); err != nil {
		t.Fatal(err)
	}
	if err := store.CheckIntegrity(catalog); err == nil {
		t.Fatal("unknown review question was not detected")
	}
}
