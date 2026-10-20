package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ExamAttempt struct {
	ID              string
	ExamID          string
	CreatedAt       time.Time
	QuestionAliases map[string]string
}

func (s *Store) SaveExamAttempt(attempt ExamAttempt) error {
	if attempt.ID == "" || attempt.ExamID == "" || len(attempt.QuestionAliases) == 0 {
		return fmt.Errorf("Prüfungsversuch ist unvollständig")
	}
	encoded, err := json.Marshal(attempt.QuestionAliases)
	if err != nil {
		return fmt.Errorf("Prüfungsfragen serialisieren: %w", err)
	}
	_, err = s.DB.Exec(`INSERT INTO exam_attempts(id,exam_id,created_at,questions_json) VALUES(?,?,?,?)`, attempt.ID, attempt.ExamID, attempt.CreatedAt.UTC().Format(time.RFC3339), string(encoded))
	if err != nil {
		return fmt.Errorf("Prüfungsversuch speichern: %w", err)
	}
	return nil
}

func (s *Store) ExamAttempt(id string) (ExamAttempt, bool, error) {
	var attempt ExamAttempt
	var createdAt, questions string
	err := s.DB.QueryRow(`SELECT id,exam_id,created_at,questions_json FROM exam_attempts WHERE id=?`, id).Scan(&attempt.ID, &attempt.ExamID, &createdAt, &questions)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ExamAttempt{}, false, nil
		}
		return ExamAttempt{}, false, fmt.Errorf("Prüfungsversuch lesen: %w", err)
	}
	attempt.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ExamAttempt{}, false, fmt.Errorf("Zeitstempel des Prüfungsversuchs lesen: %w", err)
	}
	if err = json.Unmarshal([]byte(questions), &attempt.QuestionAliases); err != nil {
		return ExamAttempt{}, false, fmt.Errorf("Prüfungsfragen lesen: %w", err)
	}
	return attempt, true, nil
}

func (s *Store) DeleteExamAttempt(id string) (bool, error) {
	result, err := s.DB.Exec(`DELETE FROM exam_attempts WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s *Store) DeleteExpiredExamAttempts(before time.Time) error {
	_, err := s.DB.Exec(`DELETE FROM exam_attempts WHERE created_at<?`, before.UTC().Format(time.RFC3339))
	return err
}
