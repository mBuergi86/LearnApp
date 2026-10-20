package content

import (
	"regexp"
	"testing"
)

func TestEveryTopicHasComprehensionWithoutChangingExamQuestions(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Comprehension) != 8 {
		t.Fatalf("supplemental checks=%d", len(catalog.Comprehension))
	}
	active, totalChecks := 0, 0
	lecture := regexp.MustCompile(`(?i)Vorlesung|Folien|laut Lernseite|gemäß`)
	for _, question := range catalog.Questions {
		if question.Enabled {
			active++
			if lecture.MatchString(question.Prompt) {
				t.Fatalf("lecture-recall prompt: %s", question.ID)
			}
		}
	}
	if active != 270 {
		t.Fatalf("exam questions=%d", active)
	}
	for _, topic := range catalog.Topics {
		t.Run(topic.ID, func(t *testing.T) {
			checks := ComprehensionQuestions(catalog, topic.ID)
			if len(checks) == 0 {
				t.Fatal("no comprehension check")
			}
			totalChecks += len(checks)
			for _, check := range checks {
				if check.TopicID != topic.ID || check.Type != QuestionTrueFalse || !check.Enabled || check.Explanation == "" {
					t.Fatal("unrelated or incomplete comprehension check")
				}
			}
		})
	}
	if totalChecks != 84 {
		t.Fatalf("checks=%d", totalChecks)
	}
	if len(ComprehensionQuestions(catalog, "does-not-exist")) != 0 {
		t.Fatal("unknown topic got questions")
	}
	for _, extra := range catalog.Comprehension {
		if _, found := FindQuestion(catalog, extra.ID); found {
			t.Fatal("supplement entered exam question pool")
		}
	}
}

func TestSupplementalChecksRejectWrongTopicTypeAndDuplicateID(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Question){
		func(q *Question) { q.TopicID = "unknown" },
		func(q *Question) { q.ID = catalog.Questions[0].ID },
		func(q *Question) { q.Type = QuestionSingle },
		func(q *Question) { q.CorrectAnswer = "unknown" },
		func(q *Question) { q.Enabled = false },
	} {
		copyCatalog := *catalog
		copyCatalog.Comprehension = append([]Question(nil), catalog.Comprehension...)
		mutate(&copyCatalog.Comprehension[0])
		if copyCatalog.Validate() == nil {
			t.Fatal("invalid supplemental check accepted")
		}
	}
}
