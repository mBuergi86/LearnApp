package progress

import (
	"mldl-exam-trainer/internal/content"
	"slices"
	"testing"
	"time"
)

func TestIntervalsAndRecovery(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Zurich")
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	p := QuestionProgress{}
	for i, want := range []time.Duration{2 * 24 * time.Hour, 5 * 24 * time.Hour, 10 * 24 * time.Hour, 18 * 24 * time.Hour, 30 * 24 * time.Hour, 30 * 24 * time.Hour} {
		p = ApplyResult(p, true, now.Add(time.Duration(i)*24*time.Hour), loc)
		got := p.NextReviewAt.Sub(now.Add(time.Duration(i) * 24 * time.Hour))
		if got != want {
			t.Fatalf("stage %d: got %v want %v", i+1, got, want)
		}
	}
	p = ApplyResult(p, false, now.Add(7*24*time.Hour), loc)
	if got := p.NextReviewAt.Sub(now.Add(7 * 24 * time.Hour)); got != 24*time.Hour {
		t.Fatalf("wrong: %v", got)
	}
	p = ApplyResult(p, true, now.Add(8*24*time.Hour), loc)
	if got := p.NextReviewAt.Sub(now.Add(8 * 24 * time.Hour)); got != 3*24*time.Hour {
		t.Fatalf("recovery: %v", got)
	}
}
func TestMasteryStates(t *testing.T) {
	status, _ := TopicStatus(false, nil)
	if status != Unlearned {
		t.Fatal(status)
	}
	status, _ = TopicStatus(true, nil)
	if status != Seen {
		t.Fatal(status)
	}
	p := QuestionProgress{CorrectCount: 1, ConsecutiveCorrect: 1, LastCorrect: true, SuccessfulDays: 1}
	status, _ = TopicStatus(true, []QuestionProgress{p})
	if status != Practicing {
		t.Fatal(status)
	}
	p.CorrectCount = 3
	p.ConsecutiveCorrect = 3
	p.SuccessfulDays = 3
	status, _ = TopicStatus(true, []QuestionProgress{p})
	if status != Mastered {
		t.Fatal(status)
	}
	p.LastCorrect = false
	p.WrongCount = 1
	p.ConsecutiveWrong = 1
	status, _ = TopicStatus(true, []QuestionProgress{p})
	if status != Weak {
		t.Fatal(status)
	}
	p.LastCorrect = true
	p.ConsecutiveWrong = 0
	p.ConsecutiveCorrect = 1
	status, _ = TopicStatus(true, []QuestionProgress{p})
	if status == Weak {
		t.Fatal("weak must recover")
	}
}

func TestMasteryRequiresSuccessfulReviewsOnDifferentDays(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Zurich")
	start := time.Date(2026, 9, 1, 8, 0, 0, 0, location)
	item := QuestionProgress{}
	for day := 0; day < 3; day++ {
		item = ApplyResult(item, true, start.AddDate(0, 0, day), location)
	}
	status, _ := TopicStatus(true, []QuestionProgress{item})
	if status != Mastered {
		t.Fatalf("status=%s, want mastered", status)
	}
	item = ApplyResult(item, false, start.AddDate(0, 0, 3), location)
	status, _ = TopicStatus(true, []QuestionProgress{item})
	if status != Weak {
		t.Fatalf("status=%s, want weak", status)
	}
	item = ApplyResult(item, true, start.AddDate(0, 0, 4), location)
	status, _ = TopicStatus(true, []QuestionProgress{item})
	if status != Practicing {
		t.Fatalf("status=%s, want practicing", status)
	}
}
func TestWeightsAndSessionPriority(t *testing.T) {
	if PriorityWeight(content.PriorityCore) <= PriorityWeight(content.PrioritySupport) {
		t.Fatal("CORE must weigh more")
	}
	if PriorityWeight(content.PriorityReference) != 0 {
		t.Fatal("REFERENCE must be zero")
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	items := []Candidate{{QuestionID: "new", TopicID: "new-topic", New: true, Priority: content.PriorityCore, Due: now, SortOrder: 1}, {QuestionID: "weak", Weak: true, Status: Weak, Priority: content.PrioritySupport, Due: now.Add(24 * time.Hour), DueState: DueLater}, {QuestionID: "overdue", Priority: content.PriorityCore, Due: now.Add(-48 * time.Hour), DueState: DueOverdue}}
	got := SelectSession(items, 20, now)
	if got[0].QuestionID != "overdue" || got[1].QuestionID != "weak" {
		t.Fatalf("wrong order: %#v", got)
	}
	short := SelectSession(items, 10, now)
	for _, x := range short {
		if x.New {
			t.Fatal("10 minute session contains new topic")
		}
	}
	medium := SelectSession(items, 20, now)
	for _, x := range medium {
		if x.New {
			t.Fatal("20 minute session contains new topic")
		}
	}
	if CapForWeakCore(98, 2) != 89 || CapForWeakCore(98, 1) != 94 || CapForWeakCore(80, 2) != 80 {
		t.Fatal("weak CORE cap is incorrect")
	}
	long := SelectSession(items, 30, now)
	if len(long) < 3 || long[0].QuestionID != "overdue" || long[1].QuestionID != "weak" || long[2].QuestionID != "new" {
		t.Fatalf("30 minute priority is incorrect: %#v", long)
	}
}

func TestClassifyDueUsesZurichCalendarDay(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Zurich")
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, location)
	if got := ClassifyDue(time.Date(2026, 9, 2, 23, 59, 0, 0, location), now, location); got != DueOverdue {
		t.Fatal(got)
	}
	if got := ClassifyDue(time.Date(2026, 9, 3, 23, 59, 0, 0, location), now, location); got != DueToday {
		t.Fatal(got)
	}
	if got := ClassifyDue(time.Date(2026, 9, 4, 0, 0, 0, 0, location), now, location); got != DueLater {
		t.Fatal(got)
	}
}

func TestNewTopicLimits(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	items := []Candidate{
		{QuestionID: "a1", TopicID: "a", New: true, Priority: content.PriorityCore, SortOrder: 1},
		{QuestionID: "a2", TopicID: "a", New: true, Priority: content.PriorityCore, SortOrder: 1},
		{QuestionID: "b1", TopicID: "b", New: true, Priority: content.PriorityCore, SortOrder: 2},
		{QuestionID: "c1", TopicID: "c", New: true, Priority: content.PrioritySupport, SortOrder: 3},
	}
	thirty := SelectSession(items, 30, now)
	for _, item := range thirty {
		if item.TopicID != "a" {
			t.Fatalf("30 minutes included second new topic: %#v", thirty)
		}
	}
	fortyFive := SelectSession(items, 45, now)
	topics := map[string]bool{}
	for _, item := range fortyFive {
		topics[item.TopicID] = true
	}
	if len(topics) != 2 || !topics["a"] || !topics["b"] {
		t.Fatalf("45 minute topics=%v", topics)
	}
}

func TestPlanRetryKeepsThreeOtherQuestionsBetweenAttempts(t *testing.T) {
	questions := []string{"a", "b", "c", "wrong"}
	if got := PlanRetry("a", questions, 0, 3); len(got) != 1 || got[0] != "a" {
		t.Fatalf("early retry tail=%v", got)
	}
	got := PlanRetry("wrong", questions, 3, 3)
	want := []string{"c", "b", "a", "wrong"}
	if !slices.Equal(got, want) {
		t.Fatalf("late retry tail=%v want=%v", got, want)
	}
	if got := PlanRetry("wrong", []string{"other", "wrong"}, 1, 3); !slices.Equal(got, []string{"other", "wrong"}) {
		t.Fatalf("small session retry tail=%v", got)
	}
	if got := PlanRetry("only", []string{"only"}, 0, 3); got != nil {
		t.Fatalf("single-question retry=%v", got)
	}
}
