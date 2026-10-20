package progress

import (
	"sort"
	"time"

	"mldl-exam-trainer/internal/content"
)

type Clock interface{ Now() time.Time }
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type Status string

type DueState string

const (
	Unlearned  Status   = "unlearned"
	Seen       Status   = "seen"
	Practicing Status   = "practicing"
	Mastered   Status   = "mastered"
	Weak       Status   = "weak"
	DueOverdue DueState = "overdue"
	DueToday   DueState = "today"
	DueLater   DueState = "later"
)

var Intervals = []time.Duration{48 * time.Hour, 5 * 24 * time.Hour, 10 * 24 * time.Hour, 18 * 24 * time.Hour, 30 * 24 * time.Hour}

type QuestionProgress struct {
	QuestionID         string    `json:"question_id"`
	CorrectCount       int       `json:"correct_count"`
	WrongCount         int       `json:"wrong_count"`
	ConsecutiveCorrect int       `json:"consecutive_correct"`
	ConsecutiveWrong   int       `json:"consecutive_wrong"`
	LastAnsweredAt     time.Time `json:"last_answered_at"`
	LastCorrect        bool      `json:"last_correct"`
	NextReviewAt       time.Time `json:"next_review_at"`
	IntervalStage      int       `json:"interval_stage"`
	SuccessfulDays     int       `json:"successful_days"`
	LastSuccessDate    string    `json:"-"`
}

func ApplyResult(current QuestionProgress, correct bool, now time.Time, location *time.Location) QuestionProgress {
	current.LastAnsweredAt = now.UTC()
	current.LastCorrect = correct
	if correct {
		current.CorrectCount++
		current.ConsecutiveCorrect++
		current.ConsecutiveWrong = 0
		date := now.In(location).Format("2006-01-02")
		if date != current.LastSuccessDate {
			current.SuccessfulDays++
			current.LastSuccessDate = date
		}
		if current.IntervalStage < len(Intervals) {
			current.IntervalStage++
		}
		interval := Intervals[min(current.IntervalStage-1, len(Intervals)-1)]
		if current.WrongCount > 0 && current.ConsecutiveCorrect == 1 {
			interval = 3 * 24 * time.Hour
			current.IntervalStage = 1
		}
		current.NextReviewAt = now.Add(interval).UTC()
	} else {
		current.WrongCount++
		current.ConsecutiveWrong++
		current.ConsecutiveCorrect = 0
		current.IntervalStage = 0
		current.NextReviewAt = now.Add(24 * time.Hour).UTC()
	}
	return current
}

func TopicStatus(opened bool, questions []QuestionProgress) (Status, float64) {
	if len(questions) == 0 {
		if opened {
			return Seen, .1
		}
		return Unlearned, 0
	}
	score := 0.0
	weak := false
	mastered := 0
	answered := 0
	for _, q := range questions {
		total := q.CorrectCount + q.WrongCount
		if total == 0 {
			continue
		}
		answered++
		recent := float64(q.ConsecutiveCorrect) / 3
		if recent > 1 {
			recent = 1
		}
		accuracy := float64(q.CorrectCount) / float64(total)
		score += .65*recent + .35*accuracy
		if !q.LastCorrect || q.ConsecutiveWrong > 0 {
			weak = true
		}
		if q.SuccessfulDays >= 3 && q.ConsecutiveCorrect >= 3 && q.LastCorrect {
			mastered++
		}
	}
	if answered == 0 {
		if opened {
			return Seen, .1
		}
		return Unlearned, 0
	}
	score /= float64(len(questions))
	if weak {
		return Weak, score
	}
	if mastered == len(questions) {
		return Mastered, score
	}
	return Practicing, score
}

func PriorityWeight(p content.Priority) float64 {
	if p == content.PriorityCore {
		return 2
	}
	if p == content.PrioritySupport {
		return 1
	}
	return 0
}
func StatusFactor(s Status, score float64) float64 {
	switch s {
	case Mastered:
		return 1
	case Practicing:
		return .25 + .45*score
	case Seen:
		return .1
	case Weak:
		return .1 * score
	default:
		return 0
	}
}

func CapForWeakCore(percent, weakCore int) int {
	if weakCore >= 2 && percent > 89 {
		return 89
	}
	if weakCore == 1 && percent > 94 {
		return 94
	}
	return percent
}

type Candidate struct {
	QuestionID string           `json:"question_id"`
	TopicID    string           `json:"topic_id"`
	Priority   content.Priority `json:"priority"`
	Status     Status           `json:"status"`
	Due        time.Time        `json:"due_at"`
	DueState   DueState         `json:"due_state"`
	New        bool             `json:"new"`
	Weak       bool             `json:"weak"`
	SortOrder  int              `json:"-"`
}

func ClassifyDue(due, now time.Time, location *time.Location) DueState {
	localNow := now.In(location)
	startToday := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	localDue := due.In(location)
	if localDue.Before(startToday) {
		return DueOverdue
	}
	if localDue.Before(startToday.AddDate(0, 0, 1)) {
		return DueToday
	}
	return DueLater
}

func SelectSession(items []Candidate, minutes int, now time.Time) []Candidate {
	filtered := make([]Candidate, 0, len(items))
	for _, item := range items {
		if (!item.New && item.DueState == DueLater && !item.Weak) || ((minutes == 10 || minutes == 20) && item.New) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		rank := func(c Candidate) int {
			if !c.New && c.Priority == content.PriorityCore && c.DueState == DueOverdue {
				return 0
			}
			if !c.New && c.Priority == content.PriorityCore && c.DueState == DueToday {
				return 1
			}
			if c.Weak {
				return 2
			}
			if !c.New && c.Priority == content.PrioritySupport && c.DueState != DueLater {
				return 3
			}
			if c.New && c.Priority == content.PriorityCore {
				return 4
			}
			return 5
		}
		ri, rj := rank(filtered[i]), rank(filtered[j])
		if ri != rj {
			return ri < rj
		}
		if filtered[i].New && filtered[j].New && filtered[i].SortOrder != filtered[j].SortOrder {
			return filtered[i].SortOrder < filtered[j].SortOrder
		}
		return filtered[i].Due.Before(filtered[j].Due)
	})
	limit := map[int]int{10: 5, 20: 10, 30: 15, 45: 22}[minutes]
	if limit == 0 {
		limit = 15
	}
	maxNewTopics := map[int]int{10: 0, 20: 0, 30: 1, 45: 2}[minutes]
	selected := make([]Candidate, 0, min(limit, len(filtered)))
	newTopics := map[string]bool{}
	for _, item := range filtered {
		if len(selected) == limit {
			break
		}
		if item.New && !newTopics[item.TopicID] && len(newTopics) == maxNewTopics {
			continue
		}
		if item.New {
			newTopics[item.TopicID] = true
		}
		selected = append(selected, item)
	}
	return selected
}

func PlanRetry(questionID string, sessionQuestionIDs []string, currentIndex, minimumGap int) []string {
	if questionID == "" || currentIndex < 0 || currentIndex >= len(sessionQuestionIDs) || sessionQuestionIDs[currentIndex] != questionID {
		return nil
	}
	if minimumGap < 1 {
		minimumGap = 1
	}
	remaining := map[string]bool{}
	gapCount := 0
	for _, id := range sessionQuestionIDs[currentIndex+1:] {
		if id != questionID && !remaining[id] {
			remaining[id] = true
			gapCount++
		}
	}
	availableOthers := map[string]bool{}
	for _, id := range sessionQuestionIDs {
		if id != questionID {
			availableOthers[id] = true
		}
	}
	if len(availableOthers) == 0 {
		return nil
	}
	tail := make([]string, 0, minimumGap+1)
	for index := currentIndex - 1; index >= 0 && gapCount < minimumGap; index-- {
		id := sessionQuestionIDs[index]
		if id == questionID || remaining[id] {
			continue
		}
		remaining[id] = true
		gapCount++
		tail = append(tail, id)
	}
	return append(tail, questionID)
}
