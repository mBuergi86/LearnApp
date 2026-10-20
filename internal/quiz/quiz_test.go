package quiz

import (
	"math/rand"
	"testing"

	"mldl-exam-trainer/internal/content"
)

func optionQuestion(questionType content.QuestionType) content.Question {
	return content.Question{ID: "q", Type: questionType, Enabled: true, Explanation: "Warum", Options: []content.QuestionOption{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}, {ID: "d", Text: "D"}}, CorrectAnswer: "b"}
}

func TestEvaluateSingleChoice(t *testing.T) {
	result, err := Evaluate(optionQuestion(content.QuestionSingle), Answer{OptionIDs: []string{"b"}})
	if err != nil || !result.Correct {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestEvaluateNumericChoice(t *testing.T) {
	result, err := Evaluate(optionQuestion(content.QuestionNumeric), Answer{OptionIDs: []string{"a"}})
	if err != nil || result.Correct {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestEvaluateTrueFalse(t *testing.T) {
	question := optionQuestion(content.QuestionTrueFalse)
	question.Options = question.Options[:2]
	result, err := Evaluate(question, Answer{OptionIDs: []string{"b"}})
	if err != nil || !result.Correct {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestEvaluateMultipleSelectRequiresExactSet(t *testing.T) {
	question := optionQuestion(content.QuestionMultiple)
	question.CorrectAnswer = ""
	question.CorrectAnswers = []string{"a", "c"}
	for _, test := range []struct {
		selected []string
		correct  bool
	}{{[]string{"c", "a"}, true}, {[]string{"a"}, false}, {[]string{"a", "b", "c"}, false}} {
		result, err := Evaluate(question, Answer{OptionIDs: test.selected})
		if err != nil || result.Correct != test.correct {
			t.Fatalf("selected=%v result=%+v err=%v", test.selected, result, err)
		}
	}
}

func TestEvaluateOrdering(t *testing.T) {
	question := optionQuestion(content.QuestionOrder)
	question.CorrectAnswer = ""
	question.CorrectOrder = []string{"a", "b", "c", "d"}
	correct, _ := Evaluate(question, Answer{Order: []string{"a", "b", "c", "d"}})
	wrong, _ := Evaluate(question, Answer{Order: []string{"b", "a", "c", "d"}})
	if !correct.Correct || wrong.Correct {
		t.Fatalf("correct=%+v wrong=%+v", correct, wrong)
	}
}

func TestEvaluateMatching(t *testing.T) {
	question := content.Question{ID: "match", Type: content.QuestionMatching, Enabled: true, Explanation: "Warum", MatchItems: []content.MatchItem{{ID: "q", CorrectOptionID: "search"}, {ID: "v", CorrectOptionID: "content"}}}
	result, err := Evaluate(question, Answer{Matches: map[string]string{"q": "search", "v": "content"}})
	if err != nil || !result.Correct {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestShuffleChangesOrderAndKeepsStableIDs(t *testing.T) {
	question := optionQuestion(content.QuestionSingle)
	public := ToPublic(question, rand.New(rand.NewSource(7)))
	if public.Options[0].ID == question.Options[0].ID && public.Options[1].ID == question.Options[1].ID {
		t.Fatalf("options were not shuffled: %+v", public.Options)
	}
	result, err := Evaluate(question, Answer{OptionIDs: []string{"b"}})
	if err != nil || !result.Correct {
		t.Fatalf("stable correct ID lost after shuffle")
	}
}

func TestFilterExcludesDisabledQuestions(t *testing.T) {
	catalog := &content.Catalog{Topics: []content.Topic{{ID: "topic", ExamID: "exam", ModuleID: "module"}}, Questions: []content.Question{{ID: "on", TopicID: "topic", Enabled: true}, {ID: "off", TopicID: "topic", Enabled: false}}}
	questions := Filter(catalog, "exam", "")
	if len(questions) != 1 || questions[0].ID != "on" {
		t.Fatalf("questions=%+v", questions)
	}
}
