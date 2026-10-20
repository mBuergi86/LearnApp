package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/progress"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB       *sql.DB
	Clock    progress.Clock
	Location *time.Location
	DataDir  string
}

func Open(path string) (*Store, error) {
	return OpenWithClock(path, progress.SystemClock{})
}

func OpenWithClock(path string, clock progress.Clock) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("Datenbankpfad darf nicht leer sein")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("Datenverzeichnis erstellen: %w", err)
		}
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("SQLite öffnen: %w", err)
	}
	database.SetMaxOpenConns(1)

	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("Zeitzone laden: %w", err)
	}
	dataDir := ""
	if path != ":memory:" {
		dataDir = filepath.Dir(path)
	}
	store := &Store{DB: database, Clock: clock, Location: location, DataDir: dataDir}
	if err := store.initialize(); err != nil {
		database.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize() error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS app_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS exams (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			exam_date TEXT,
			sort_order INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS modules (
			id TEXT PRIMARY KEY,
			exam_id TEXT NOT NULL REFERENCES exams(id),
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS topics (
			id TEXT PRIMARY KEY,
			exam_id TEXT NOT NULL REFERENCES exams(id),
			module_id TEXT NOT NULL REFERENCES modules(id),
			title TEXT NOT NULL,
			priority TEXT NOT NULL CHECK(priority IN ('core', 'support', 'reference')),
			summary TEXT NOT NULL,
			content_json TEXT NOT NULL,
			initial_status TEXT NOT NULL DEFAULT 'unlearned',
			sort_order INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id TEXT NOT NULL,
			answered_at TEXT NOT NULL,
			correct INTEGER NOT NULL CHECK(correct IN (0, 1)),
			response_data TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_question ON reviews(question_id, answered_at)`,
		`INSERT OR IGNORE INTO app_meta (key, value) VALUES ('schema_version', '3')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('theme', 'light')`,
	}
	for _, statement := range statements {
		if _, err := s.DB.Exec(statement); err != nil {
			return fmt.Errorf("Datenbank initialisieren: %w", err)
		}
	}
	return s.migrate()
}

func (s *Store) migrate() error {
	var rawVersion string
	if err := s.DB.QueryRow(`SELECT value FROM app_meta WHERE key='schema_version'`).Scan(&rawVersion); err != nil {
		return fmt.Errorf("Schemaversion lesen: %w", err)
	}
	version, err := strconv.Atoi(rawVersion)
	if err != nil {
		return fmt.Errorf("ungültige Schemaversion %q", rawVersion)
	}
	migrations := []struct {
		version    int
		statements []string
	}{
		{4, []string{
			`CREATE TABLE IF NOT EXISTS question_progress (
				question_id TEXT PRIMARY KEY, correct_count INTEGER NOT NULL DEFAULT 0, wrong_count INTEGER NOT NULL DEFAULT 0,
				consecutive_correct INTEGER NOT NULL DEFAULT 0, consecutive_wrong INTEGER NOT NULL DEFAULT 0,
				last_answered_at TEXT, last_correct INTEGER NOT NULL DEFAULT 0, next_review_at TEXT,
				interval_stage INTEGER NOT NULL DEFAULT 0, successful_days INTEGER NOT NULL DEFAULT 0, last_success_date TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE TABLE IF NOT EXISTS topic_progress (
				topic_id TEXT PRIMARY KEY REFERENCES topics(id), mastery_status TEXT NOT NULL DEFAULT 'unlearned',
				mastery_score REAL NOT NULL DEFAULT 0, opened_at TEXT, last_reviewed_at TEXT, next_review_at TEXT
			)`,
			`CREATE INDEX IF NOT EXISTS idx_question_progress_due ON question_progress(next_review_at)`,
			`CREATE INDEX IF NOT EXISTS idx_topic_progress_due ON topic_progress(next_review_at)`,
		}},
		{5, []string{
			`CREATE TABLE IF NOT EXISTS progress_applied_reviews (
				review_id INTEGER PRIMARY KEY REFERENCES reviews(id) ON DELETE CASCADE,
				applied_at TEXT NOT NULL
			)`,
		}},
		{6, []string{
			`CREATE TABLE IF NOT EXISTS exam_attempts (
				id TEXT PRIMARY KEY,
				exam_id TEXT NOT NULL REFERENCES exams(id),
				created_at TEXT NOT NULL,
				questions_json TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS idx_exam_attempts_created ON exam_attempts(created_at)`,
		}},
		{7, []string{
			`CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT NOT NULL COLLATE NOCASE UNIQUE,
				password_hash TEXT NOT NULL,
				role TEXT NOT NULL CHECK(role IN ('user', 'admin')),
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				disabled INTEGER NOT NULL DEFAULT 0 CHECK(disabled IN (0, 1))
			)`,
			`CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				session_hash BLOB NOT NULL UNIQUE,
				csrf_hash BLOB NOT NULL,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				created_at TEXT NOT NULL,
				last_seen_at TEXT NOT NULL,
				expires_at TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
			`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expires_at)`,
		}},
	}
	for _, migration := range migrations {
		if version >= migration.version {
			continue
		}
		transaction, err := s.DB.Begin()
		if err != nil {
			return fmt.Errorf("Migration %d starten: %w", migration.version, err)
		}
		for _, statement := range migration.statements {
			if _, err = transaction.Exec(statement); err != nil {
				transaction.Rollback()
				return fmt.Errorf("Migration %d ausführen: %w", migration.version, err)
			}
		}
		if _, err = transaction.Exec(`UPDATE app_meta SET value=? WHERE key='schema_version'`, strconv.Itoa(migration.version)); err != nil {
			transaction.Rollback()
			return fmt.Errorf("Migration %d markieren: %w", migration.version, err)
		}
		if err = transaction.Commit(); err != nil {
			return fmt.Errorf("Migration %d abschliessen: %w", migration.version, err)
		}
		version = migration.version
	}
	return nil
}

func (s *Store) SaveReview(questionID string, correct bool, responseData []byte) error {
	if questionID == "" || len(responseData) == 0 || !json.Valid(responseData) {
		return fmt.Errorf("Review-Daten sind ungültig")
	}
	_, err := s.DB.Exec(`INSERT INTO reviews (question_id, answered_at, correct, response_data) VALUES (?, ?, ?, ?)`, questionID, s.Clock.Now().UTC().Format(time.RFC3339), correct, string(responseData))
	if err != nil {
		return fmt.Errorf("Review speichern: %w", err)
	}
	return nil
}

func (s *Store) SyncCatalog(catalog *content.Catalog) error {
	transaction, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("Content-Synchronisierung starten: %w", err)
	}
	defer transaction.Rollback()

	for _, exam := range catalog.Exams {
		_, err = transaction.Exec(`INSERT INTO exams (id, name, description, sort_order) VALUES (?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, sort_order=excluded.sort_order`, exam.ID, exam.Name, exam.Description, exam.SortOrder)
		if err != nil {
			return fmt.Errorf("Prüfungsbereich %q synchronisieren: %w", exam.ID, err)
		}
	}
	for _, module := range catalog.Modules {
		_, err = transaction.Exec(`INSERT INTO modules (id, exam_id, name, description, sort_order) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET exam_id=excluded.exam_id, name=excluded.name, description=excluded.description, sort_order=excluded.sort_order`, module.ID, module.ExamID, module.Name, module.Description, module.SortOrder)
		if err != nil {
			return fmt.Errorf("Modul %q synchronisieren: %w", module.ID, err)
		}
	}
	for _, topic := range catalog.Topics {
		encoded, err := json.Marshal(topic)
		if err != nil {
			return fmt.Errorf("Thema %q serialisieren: %w", topic.ID, err)
		}
		_, err = transaction.Exec(`INSERT INTO topics (id, exam_id, module_id, title, priority, summary, content_json, initial_status, sort_order, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET exam_id=excluded.exam_id, module_id=excluded.module_id, title=excluded.title,
			priority=excluded.priority, summary=excluded.summary, content_json=excluded.content_json,
			initial_status=excluded.initial_status, sort_order=excluded.sort_order, updated_at=excluded.updated_at`,
			topic.ID, topic.ExamID, topic.ModuleID, topic.Title, topic.Priority, topic.Summary, string(encoded), topic.InitialStatus, topic.SortOrder, s.Clock.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("Thema %q synchronisieren: %w", topic.ID, err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("Content-Synchronisierung abschliessen: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) Theme() (string, error) {
	var theme string
	if err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = 'theme'`).Scan(&theme); err != nil {
		return "", fmt.Errorf("Theme lesen: %w", err)
	}
	return theme, nil
}

func (s *Store) SetTheme(theme string) error {
	if theme != "light" && theme != "dark" && theme != "system" {
		return fmt.Errorf("ungültiges Theme %q", theme)
	}
	_, err := s.DB.Exec(`INSERT INTO settings (key, value, updated_at) VALUES ('theme', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`, theme)
	if err != nil {
		return fmt.Errorf("Theme speichern: %w", err)
	}
	return nil
}
