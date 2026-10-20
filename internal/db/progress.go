package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/progress"
	"mldl-exam-trainer/internal/quiz"
)

type TopicProgress struct {
	TopicID        string           `json:"topic_id"`
	ExamID         string           `json:"exam_id"`
	ModuleID       string           `json:"module_id"`
	Title          string           `json:"title"`
	Priority       content.Priority `json:"priority"`
	Status         progress.Status  `json:"status"`
	Score          float64          `json:"score"`
	LastReviewedAt *time.Time       `json:"last_reviewed_at,omitempty"`
	NextReviewAt   *time.Time       `json:"next_review_at,omitempty"`
}

type ExamProgress struct {
	ExamID  string                  `json:"exam_id"`
	Percent int                     `json:"percent"`
	Counts  map[progress.Status]int `json:"counts"`
	Topics  []TopicProgress         `json:"topics"`
	Modules map[string]int          `json:"modules"`
}

func (s *Store) RecordAnswer(catalog *content.Catalog, question content.Question, correct bool, response quiz.Answer) error {
	return s.RecordAnswers(catalog, []AnswerRecord{{Question: question, Correct: correct, Response: response}})
}

type AnswerRecord struct {
	Question content.Question
	Correct  bool
	Response quiz.Answer
}

func (s *Store) RecordAnswers(catalog *content.Catalog, records []AnswerRecord) error {
	if len(records) == 0 {
		return fmt.Errorf("mindestens eine Antwort erforderlich")
	}
	now := s.Clock.Now().UTC()
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	touchedTopics := map[string]bool{}
	for _, record := range records {
		data, err := json.Marshal(record.Response)
		if err != nil {
			return err
		}
		result, err := tx.Exec(`INSERT INTO reviews(question_id,answered_at,correct,response_data) VALUES(?,?,?,?)`, record.Question.ID, now.Format(time.RFC3339), record.Correct, string(data))
		if err != nil {
			return err
		}
		reviewID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		current, err := readQuestionProgress(tx, record.Question.ID)
		if err != nil {
			return err
		}
		current = progress.ApplyResult(current, record.Correct, now, s.Location)
		if err = writeQuestionProgress(tx, current); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO progress_applied_reviews(review_id,applied_at) VALUES(?,?)`, reviewID, now.Format(time.RFC3339)); err != nil {
			return err
		}
		touchedTopics[record.Question.TopicID] = true
	}
	for topicID := range touchedTopics {
		if err = s.refreshTopic(tx, catalog, topicID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func readQuestionProgress(q interface{ QueryRow(string, ...any) *sql.Row }, id string) (progress.QuestionProgress, error) {
	p := progress.QuestionProgress{QuestionID: id}
	var last, next sql.NullString
	var lastCorrect int
	err := q.QueryRow(`SELECT correct_count,wrong_count,consecutive_correct,consecutive_wrong,last_answered_at,last_correct,next_review_at,interval_stage,successful_days,last_success_date FROM question_progress WHERE question_id=?`, id).Scan(&p.CorrectCount, &p.WrongCount, &p.ConsecutiveCorrect, &p.ConsecutiveWrong, &last, &lastCorrect, &next, &p.IntervalStage, &p.SuccessfulDays, &p.LastSuccessDate)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	p.LastCorrect = lastCorrect == 1
	if last.Valid {
		p.LastAnsweredAt, _ = time.Parse(time.RFC3339, last.String)
	}
	if next.Valid {
		p.NextReviewAt, _ = time.Parse(time.RFC3339, next.String)
	}
	return p, nil
}
func writeQuestionProgress(tx *sql.Tx, p progress.QuestionProgress) error {
	_, err := tx.Exec(`INSERT INTO question_progress(question_id,correct_count,wrong_count,consecutive_correct,consecutive_wrong,last_answered_at,last_correct,next_review_at,interval_stage,successful_days,last_success_date) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(question_id) DO UPDATE SET correct_count=excluded.correct_count,wrong_count=excluded.wrong_count,consecutive_correct=excluded.consecutive_correct,consecutive_wrong=excluded.consecutive_wrong,last_answered_at=excluded.last_answered_at,last_correct=excluded.last_correct,next_review_at=excluded.next_review_at,interval_stage=excluded.interval_stage,successful_days=excluded.successful_days,last_success_date=excluded.last_success_date`, p.QuestionID, p.CorrectCount, p.WrongCount, p.ConsecutiveCorrect, p.ConsecutiveWrong, p.LastAnsweredAt.Format(time.RFC3339), p.LastCorrect, p.NextReviewAt.Format(time.RFC3339), p.IntervalStage, p.SuccessfulDays, p.LastSuccessDate)
	return err
}

func (s *Store) MarkTopicSeen(catalog *content.Catalog, topicID string) error {
	if _, ok := content.FindTopic(catalog, topicID); !ok {
		return fmt.Errorf("unbekanntes Topic")
	}
	now := s.Clock.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`INSERT INTO topic_progress(topic_id,mastery_status,mastery_score,opened_at) VALUES(?,?,0,?) ON CONFLICT(topic_id) DO UPDATE SET opened_at=COALESCE(topic_progress.opened_at,excluded.opened_at),mastery_status=CASE WHEN topic_progress.mastery_status='unlearned' THEN 'seen' ELSE topic_progress.mastery_status END`, topicID, progress.Seen, now)
	return err
}

func (s *Store) refreshTopic(tx *sql.Tx, catalog *content.Catalog, topicID string) error {
	opened := false
	var openedAt sql.NullString
	_ = tx.QueryRow(`SELECT opened_at FROM topic_progress WHERE topic_id=?`, topicID).Scan(&openedAt)
	opened = openedAt.Valid
	var list []progress.QuestionProgress
	for _, q := range catalog.Questions {
		if q.TopicID == topicID && q.Enabled {
			p, err := readQuestionProgress(tx, q.ID)
			if err != nil {
				return err
			}
			list = append(list, p)
		}
	}
	status, score := progress.TopicStatus(opened, list)
	var last, next time.Time
	for _, p := range list {
		if p.LastAnsweredAt.After(last) {
			last = p.LastAnsweredAt
		}
		if !p.NextReviewAt.IsZero() && (next.IsZero() || p.NextReviewAt.Before(next)) {
			next = p.NextReviewAt
		}
	}
	_, err := tx.Exec(`INSERT INTO topic_progress(topic_id,mastery_status,mastery_score,last_reviewed_at,next_review_at) VALUES(?,?,?,?,?) ON CONFLICT(topic_id) DO UPDATE SET mastery_status=excluded.mastery_status,mastery_score=excluded.mastery_score,last_reviewed_at=excluded.last_reviewed_at,next_review_at=excluded.next_review_at`, topicID, status, score, nullableTime(last), nullableTime(next))
	return err
}
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *Store) BackfillReviews(catalog *content.Catalog) error {
	var progressCount, appliedCount int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM question_progress`).Scan(&progressCount); err != nil {
		return err
	}
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM progress_applied_reviews`).Scan(&appliedCount); err != nil {
		return err
	}
	if progressCount > 0 && appliedCount == 0 {
		_, err := s.DB.Exec(`INSERT OR IGNORE INTO progress_applied_reviews(review_id,applied_at) SELECT id,? FROM reviews`, s.Clock.Now().UTC().Format(time.RFC3339))
		return err
	}
	var pending int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM reviews r LEFT JOIN progress_applied_reviews p ON p.review_id=r.id WHERE p.review_id IS NULL`).Scan(&pending); err != nil {
		return err
	}
	if pending == 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT r.id,r.question_id,r.answered_at,r.correct FROM reviews r LEFT JOIN progress_applied_reviews p ON p.review_id=r.id WHERE p.review_id IS NULL ORDER BY r.id`)
	if err != nil {
		return err
	}
	type legacyReview struct {
		reviewID int64
		id, at   string
		correct  bool
	}
	var legacy []legacyReview
	for rows.Next() {
		var item legacyReview
		var correct int
		if err = rows.Scan(&item.reviewID, &item.id, &item.at, &correct); err != nil {
			rows.Close()
			return err
		}
		item.correct = correct == 1
		legacy = append(legacy, item)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	touched := map[string]bool{}
	for _, item := range legacy {
		q, ok := content.FindQuestion(catalog, item.id)
		if !ok {
			if _, e := tx.Exec(`INSERT INTO progress_applied_reviews(review_id,applied_at) VALUES(?,?)`, item.reviewID, s.Clock.Now().UTC().Format(time.RFC3339)); e != nil {
				return e
			}
			continue
		}
		p, e := readQuestionProgress(tx, item.id)
		if e != nil {
			return e
		}
		when, e := time.Parse(time.RFC3339, item.at)
		if e != nil {
			continue
		}
		p = progress.ApplyResult(p, item.correct, when, s.Location)
		if e = writeQuestionProgress(tx, p); e != nil {
			return e
		}
		if _, e = tx.Exec(`INSERT INTO progress_applied_reviews(review_id,applied_at) VALUES(?,?)`, item.reviewID, s.Clock.Now().UTC().Format(time.RFC3339)); e != nil {
			return e
		}
		touched[q.TopicID] = true
	}
	for id := range touched {
		if err = s.refreshTopic(tx, catalog, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RefreshTopics(catalog *content.Catalog) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, topic := range catalog.Topics {
		if err = s.refreshTopic(tx, catalog, topic.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Progress(catalog *content.Catalog, examFilter string) ([]ExamProgress, error) {
	var result []ExamProgress
	for _, exam := range catalog.Exams {
		if examFilter != "" && exam.ID != examFilter {
			continue
		}
		ep := ExamProgress{ExamID: exam.ID, Counts: map[progress.Status]int{}, Modules: map[string]int{}}
		total, earned := 0.0, 0.0
		moduleTotal, moduleEarned := map[string]float64{}, map[string]float64{}
		weakCore := 0
		for _, topic := range catalog.Topics {
			if topic.ExamID != exam.ID {
				continue
			}
			tp := TopicProgress{TopicID: topic.ID, ExamID: exam.ID, ModuleID: topic.ModuleID, Title: topic.Title, Priority: topic.Priority, Status: progress.Unlearned}
			var status string
			var last, next sql.NullString
			err := s.DB.QueryRow(`SELECT mastery_status,mastery_score,last_reviewed_at,next_review_at FROM topic_progress WHERE topic_id=?`, topic.ID).Scan(&status, &tp.Score, &last, &next)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
			if err == nil {
				tp.Status = progress.Status(status)
				tp.LastReviewedAt = parsePtr(last)
				tp.NextReviewAt = parsePtr(next)
			}
			if topic.Priority != content.PriorityReference {
				ep.Counts[tp.Status]++
			}
			if topic.Priority == content.PriorityCore && tp.Status == progress.Weak {
				weakCore++
			}
			w := progress.PriorityWeight(topic.Priority)
			total += w
			earned += w * progress.StatusFactor(tp.Status, tp.Score)
			moduleTotal[topic.ModuleID] += w
			moduleEarned[topic.ModuleID] += w * progress.StatusFactor(tp.Status, tp.Score)
			ep.Topics = append(ep.Topics, tp)
		}
		if total > 0 {
			ep.Percent = int(earned/total*100 + .5)
		}
		ep.Percent = progress.CapForWeakCore(ep.Percent, weakCore)
		for id, t := range moduleTotal {
			if t > 0 {
				ep.Modules[id] = int(moduleEarned[id]/t*100 + .5)
			}
		}
		result = append(result, ep)
	}
	return result, nil
}
func parsePtr(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, e := time.Parse(time.RFC3339, v.String)
	if e != nil {
		return nil
	}
	return &t
}

func (s *Store) Candidates(catalog *content.Catalog, examID string) ([]progress.Candidate, error) {
	var out []progress.Candidate
	now := s.Clock.Now().UTC()
	for _, topic := range catalog.Topics {
		if topic.ExamID != examID || topic.Priority == content.PriorityReference {
			continue
		}
		status := progress.Unlearned
		_ = s.DB.QueryRow(`SELECT mastery_status FROM topic_progress WHERE topic_id=?`, topic.ID).Scan(&status)
		moduleOrder := 0
		for _, module := range catalog.Modules {
			if module.ID == topic.ModuleID {
				moduleOrder = module.SortOrder
				break
			}
		}
		contentOrder := moduleOrder*1000 + topic.SortOrder
		for _, q := range catalog.Questions {
			if q.TopicID != topic.ID || !q.Enabled {
				continue
			}
			p, e := readQuestionProgress(s.DB, q.ID)
			if e != nil {
				return nil, e
			}
			isNew := p.CorrectCount+p.WrongCount == 0
			due := p.NextReviewAt
			if isNew {
				due = now.Add(time.Duration(contentOrder) * time.Minute)
			}
			dueState := progress.DueLater
			if !isNew {
				dueState = progress.ClassifyDue(due, now, s.Location)
			}
			answered := p.CorrectCount+p.WrongCount > 0
			out = append(out, progress.Candidate{QuestionID: q.ID, TopicID: topic.ID, Priority: topic.Priority, Status: status, Due: due, DueState: dueState, New: isNew, Weak: answered && !p.LastCorrect, SortOrder: contentOrder})
		}
	}
	return out, nil
}
