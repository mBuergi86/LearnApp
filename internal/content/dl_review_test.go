package content

import (
	"strings"
	"testing"
)

func TestDuplicateOptionTextIsRejected(t *testing.T) {
	question := Question{ID: "duplicate", Type: QuestionSingle, Prompt: "Choose", Explanation: "Because", Difficulty: 1,
		Options: []QuestionOption{{ID: "a", Text: "Eine Antwort"}, {ID: "b", Text: "  eine   ANTWORT "}}, CorrectAnswer: "a"}
	if err := validateQuestion(question); err == nil || !strings.Contains(err.Error(), "identische") {
		t.Fatalf("duplicate text accepted: %v", err)
	}
	question = Question{ID: "duplicate-match", Type: QuestionMatching, Prompt: "Match", Explanation: "Because", Difficulty: 1,
		MatchOptions: []QuestionOption{{ID: "a", Text: "Ziel"}, {ID: "b", Text: " ziel "}},
		MatchItems:   []MatchItem{{ID: "x", Text: "X", CorrectOptionID: "a"}, {ID: "y", Text: "Y", CorrectOptionID: "b"}}}
	if err := validateQuestion(question); err == nil {
		t.Fatal("ambiguous identical match targets accepted")
	}
}

func TestIllustrationRejectsMalformedNumericalData(t *testing.T) {
	base := Illustration{Kind: "matrix", Title: "Example", Caption: "A matrix", Labels: []string{"Values"}, Values: [][]float64{{1, 2}, {3}}}
	if err := validateIllustration(base); err == nil {
		t.Fatal("ragged matrix accepted")
	}
	base.Kind = "plot"
	base.Labels = []string{"x", "y"}
	base.Values = [][]float64{{1, 2}, {1, 3}}
	if err := validateIllustration(base); err == nil {
		t.Fatal("non-increasing curve coordinates accepted")
	}
}

func TestDLTopicsProvideExplanationsExamplesAndAccessibleFigures(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, topic := range catalog.Topics {
		if topic.ExamID != "deep-learning-data-science" {
			continue
		}
		count++
		if len(topic.Explanation) < 5 || len(topic.Steps) < 3 || strings.TrimSpace(topic.Example) == "" || len(topic.Confusions) < 2 {
			t.Fatalf("%s has incomplete learning material", topic.ID)
		}
		if len(topic.Illustrations) != 2 {
			t.Fatalf("%s needs a concept figure and a worked-example figure", topic.ID)
		}
		for _, figure := range topic.Illustrations {
			if err := validateIllustration(figure); err != nil {
				t.Fatalf("%s: %v", topic.ID, err)
			}
		}
	}
	if count != 30 {
		t.Fatalf("topics=%d, want 30", count)
	}
	seen := map[string]bool{}
	for _, question := range catalog.Questions {
		if !question.Enabled {
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(question.Prompt), " "))
		if seen[key] {
			t.Fatalf("repeated active prompt: %s", question.ID)
		}
		seen[key] = true
	}
}
