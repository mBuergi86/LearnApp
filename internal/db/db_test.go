package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/progress"
	"mldl-exam-trainer/internal/quiz"
)

type fixedClock struct{ value time.Time }

func (c fixedClock) Now() time.Time { return c.value }

func TestOpenInitializesDatabaseAndDefaultTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "exam.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	var version string
	if err := store.DB.QueryRow(`SELECT value FROM app_meta WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatalf("schema version lesen: %v", err)
	}
	if version != "7" {
		t.Fatalf("schema version = %q, want 7", version)
	}
	var foreignKeys int
	if err := store.DB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys lesen: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	theme, err := store.Theme()
	if err != nil {
		t.Fatalf("Theme() error = %v", err)
	}
	if theme != "light" {
		t.Fatalf("Theme() = %q, want light", theme)
	}
}

func TestSaveReviewStoresRawHistoryWithoutMastery(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.SaveReview("question-one", true, []byte(`{"option_ids":["a"]}`)); err != nil {
		t.Fatal(err)
	}
	var questionID string
	var correct int
	var response string
	if err := store.DB.QueryRow(`SELECT question_id, correct, response_data FROM reviews`).Scan(&questionID, &correct, &response); err != nil {
		t.Fatal(err)
	}
	if questionID != "question-one" || correct != 1 || response == "" {
		t.Fatalf("stored review=%q %d %q", questionID, correct, response)
	}
}

func TestSyncCatalogStoresSeparateExamsModulesAndTopics(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{
		Exams:   []content.Exam{{ID: "machine-learning", Name: "Machine Learning", SortOrder: 1}, {ID: "deep-learning-data-science", Name: "Deep Learning", SortOrder: 2}},
		Modules: []content.Module{{ID: "ml-foundations", ExamID: "machine-learning", Name: "Grundlagen", SortOrder: 1}},
		Topics:  []content.Topic{{ID: "example", ExamID: "machine-learning", ModuleID: "ml-foundations", Title: "Beispiel", Priority: content.PriorityCore, Summary: "Kern", Explanation: []string{"Text"}, Sources: []content.Source{{File: "a.pdf", Path: "a.pdf", Pages: []int{1}}}, InitialStatus: "unlearned", SortOrder: 1}},
	}
	if err := store.SyncCatalog(catalog); err != nil {
		t.Fatalf("SyncCatalog() error = %v", err)
	}
	for table, expected := range map[string]int{"exams": 2, "modules": 1, "topics": 1} {
		var count int
		if err := store.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != expected {
			t.Fatalf("%s count = %d, want %d", table, count, expected)
		}
	}
}

func TestThemePersists(t *testing.T) {
	for _, preference := range []string{"light", "dark", "system"} {
		t.Run(preference, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "exam.db")
			store, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.SetTheme(preference); err != nil {
				t.Fatalf("SetTheme() error = %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}

			reopened, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			theme, err := reopened.Theme()
			if err != nil {
				t.Fatal(err)
			}
			if theme != preference {
				t.Fatalf("persisted theme = %q, want %q", theme, preference)
			}
		})
	}
}

func TestExamAttemptPersistsAndExpires(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exam.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(`INSERT INTO exams(id,name,sort_order) VALUES('ml','ML',1)`); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	want := ExamAttempt{ID: "attempt", ExamID: "ml", CreatedAt: created, QuestionAliases: map[string]string{"public-q": "internal-q"}}
	if err = store.SaveExamAttempt(want); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, found, err := reopened.ExamAttempt("attempt")
	if err != nil || !found {
		t.Fatalf("ExamAttempt() found=%v err=%v", found, err)
	}
	if got.ExamID != want.ExamID || !got.CreatedAt.Equal(created) || got.QuestionAliases["public-q"] != "internal-q" {
		t.Fatalf("persisted attempt=%+v", got)
	}
	if err = reopened.DeleteExpiredExamAttempts(created.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, found, err = reopened.ExamAttempt("attempt"); err != nil || found {
		t.Fatalf("expired attempt found=%v err=%v", found, err)
	}
}

func TestSetThemeRejectsUnknownValue(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.SetTheme("neon"); err == nil {
		t.Fatal("SetTheme() accepted an invalid value")
	}
}

func TestRecordAnswerUsesInjectedClockAndKeepsExamProgressSeparate(t *testing.T) {
	now := time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)
	store, err := OpenWithClock(":memory:", fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{Exams: []content.Exam{{ID: "ml", Name: "ML"}, {ID: "dl", Name: "DL"}}, Modules: []content.Module{{ID: "m1", ExamID: "ml", Name: "M1"}, {ID: "m2", ExamID: "dl", Name: "M2"}}, Topics: []content.Topic{{ID: "t1", ExamID: "ml", ModuleID: "m1", Title: "T1", Priority: content.PriorityCore, Summary: "S"}, {ID: "t2", ExamID: "dl", ModuleID: "m2", Title: "T2", Priority: content.PriorityCore, Summary: "S"}}, Questions: []content.Question{{ID: "q1", TopicID: "t1", Enabled: true}}}
	if err = store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordAnswer(catalog, catalog.Questions[0], true, quiz.Answer{OptionIDs: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	var answered, next string
	if err = store.DB.QueryRow(`SELECT last_answered_at,next_review_at FROM question_progress WHERE question_id='q1'`).Scan(&answered, &next); err != nil {
		t.Fatal(err)
	}
	if answered != now.Format(time.RFC3339) || next != now.Add(48*time.Hour).Format(time.RFC3339) {
		t.Fatalf("times %s %s", answered, next)
	}
	all, err := store.Progress(catalog, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[1].Percent != 0 {
		t.Fatalf("exam progress mixed: %#v", all)
	}
}

func TestMigrationBackfillsLegacyReviewsExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE app_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL)`,
		`INSERT INTO app_meta(key,value) VALUES('schema_version','3')`,
		`CREATE TABLE reviews(id INTEGER PRIMARY KEY AUTOINCREMENT,question_id TEXT NOT NULL,answered_at TEXT NOT NULL,correct INTEGER NOT NULL,response_data TEXT NOT NULL)`,
		`INSERT INTO reviews(question_id,answered_at,correct,response_data) VALUES('q1','2026-09-01T08:00:00Z',1,'{"option_ids":["a"]}')`,
	} {
		if _, err = legacy.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err = legacy.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenWithClock(path, fixedClock{time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{Exams: []content.Exam{{ID: "ml", Name: "ML"}}, Modules: []content.Module{{ID: "m1", ExamID: "ml", Name: "M1"}}, Topics: []content.Topic{{ID: "t1", ExamID: "ml", ModuleID: "m1", Title: "T1", Priority: content.PriorityCore, Summary: "S"}}, Questions: []content.Question{{ID: "q1", TopicID: "t1", Enabled: true}}}
	if err = store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if err = store.BackfillReviews(catalog); err != nil {
		t.Fatal(err)
	}
	if err = store.BackfillReviews(catalog); err != nil {
		t.Fatal(err)
	}
	var reviews, correct, applied int
	if err = store.DB.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if err = store.DB.QueryRow(`SELECT correct_count FROM question_progress WHERE question_id='q1'`).Scan(&correct); err != nil {
		t.Fatal(err)
	}
	if err = store.DB.QueryRow(`SELECT COUNT(*) FROM progress_applied_reviews`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if reviews != 1 || correct != 1 || applied != 1 {
		t.Fatalf("reviews=%d correct=%d applied=%d", reviews, correct, applied)
	}
}

func TestIntegratedWeightedProgressExcludesReferenceTopics(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{Exams: []content.Exam{{ID: "exam", Name: "Exam"}}, Modules: []content.Module{{ID: "module", ExamID: "exam", Name: "Module"}}, Topics: []content.Topic{
		{ID: "core", ExamID: "exam", ModuleID: "module", Title: "Core", Priority: content.PriorityCore, Summary: "S"},
		{ID: "support", ExamID: "exam", ModuleID: "module", Title: "Support", Priority: content.PrioritySupport, Summary: "S"},
		{ID: "reference", ExamID: "exam", ModuleID: "module", Title: "Reference", Priority: content.PriorityReference, Summary: "S"},
	}, Questions: []content.Question{{ID: "qc", TopicID: "core", Enabled: true}, {ID: "qs", TopicID: "support", Enabled: true}, {ID: "qr", TopicID: "reference", Enabled: true}}}
	if err = store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id, status string
		score      float64
	}{{"core", "mastered", 1}, {"support", "unlearned", 0}, {"reference", "mastered", 1}} {
		if _, err = store.DB.Exec(`INSERT INTO topic_progress(topic_id,mastery_status,mastery_score) VALUES(?,?,?)`, row.id, row.status, row.score); err != nil {
			t.Fatal(err)
		}
	}
	result, err := store.Progress(catalog, "exam")
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Percent != 67 {
		t.Fatalf("percent=%d, want 67", result[0].Percent)
	}
	if result[0].Counts[progress.Mastered] != 1 || result[0].Counts[progress.Unlearned] != 1 {
		t.Fatalf("REFERENCE leaked into counts: %v", result[0].Counts)
	}
	if _, err = store.DB.Exec(`UPDATE topic_progress SET mastery_status='unlearned',mastery_score=0 WHERE topic_id='core'`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(`UPDATE topic_progress SET mastery_status='mastered',mastery_score=1 WHERE topic_id='support'`); err != nil {
		t.Fatal(err)
	}
	result, err = store.Progress(catalog, "exam")
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Percent != 33 {
		t.Fatalf("CORE weighting percent=%d, want 33", result[0].Percent)
	}
	candidates, err := store.Candidates(catalog, "exam")
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.TopicID == "reference" {
			t.Fatal("REFERENCE topic created a review candidate")
		}
	}
}

func TestIntegratedProgressCapsMultipleWeakCoreTopics(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{Exams: []content.Exam{{ID: "exam", Name: "Exam"}}, Modules: []content.Module{{ID: "module", ExamID: "exam", Name: "Module"}}}
	for index := 0; index < 32; index++ {
		id := fmt.Sprintf("topic-%02d", index)
		catalog.Topics = append(catalog.Topics, content.Topic{ID: id, ExamID: "exam", ModuleID: "module", Title: id, Priority: content.PriorityCore, Summary: "S"})
	}
	if err = store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	for index, topic := range catalog.Topics {
		status, score := "mastered", 1.0
		if index >= 30 {
			status, score = "weak", 1
		}
		if _, err = store.DB.Exec(`INSERT INTO topic_progress(topic_id,mastery_status,mastery_score) VALUES(?,?,?)`, topic.ID, status, score); err != nil {
			t.Fatal(err)
		}
	}
	result, err := store.Progress(catalog, "exam")
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Percent != 89 {
		t.Fatalf("percent=%d, want capped 89", result[0].Percent)
	}
}

func TestRefreshTopicsIncludesNewQuestions(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := &content.Catalog{
		Exams:     []content.Exam{{ID: "exam", Name: "Exam"}},
		Modules:   []content.Module{{ID: "module", ExamID: "exam", Name: "Module"}},
		Topics:    []content.Topic{{ID: "topic", ExamID: "exam", ModuleID: "module", Title: "Topic", Priority: content.PriorityCore, Summary: "S"}},
		Questions: []content.Question{{ID: "old-question", TopicID: "topic", Enabled: true}},
	}
	if err = store.SyncCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(`INSERT INTO topic_progress(topic_id,mastery_status,mastery_score) VALUES('topic','mastered',1)`); err != nil {
		t.Fatal(err)
	}
	catalog.Questions = append(catalog.Questions, content.Question{ID: "new-question", TopicID: "topic", Enabled: true})
	if err = store.RefreshTopics(catalog); err != nil {
		t.Fatal(err)
	}
	var status string
	if err = store.DB.QueryRow(`SELECT mastery_status FROM topic_progress WHERE topic_id='topic'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status == "mastered" {
		t.Fatal("new unanswered question did not refresh cached topic mastery")
	}
}
