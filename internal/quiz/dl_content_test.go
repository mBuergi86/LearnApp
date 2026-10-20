package quiz

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"mldl-exam-trainer/internal/content"
)

// These expectations are computed independently of the stored answer IDs.
func TestDLNumericAnswerKeysAgainstWorkedCalculations(t *testing.T) {
	catalog, err := content.Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	decimal := func(x float64) string { return fmt.Sprintf("%g", x) }
	want := map[string]string{
		"dl-cnn-output-width":              decimal(math.Floor((32-5)/1) + 1),
		"dl-batch-iteration-epoch-core":    decimal(1000 / 250),
		"dl-cnn-architecture-detail":       decimal(10 * (5*5*6 + 1)),
		"dl-cnn-spatial-dimensions-detail": decimal(10 * (5*5*3 + 1)),
	}
	checked := 0
	for _, question := range catalog.Questions {
		if !question.Enabled || question.Type != content.QuestionNumeric || !strings.HasPrefix(question.ID, "dl-") {
			continue
		}
		expected, ok := want[question.ID]
		if !ok {
			t.Fatalf("numeric question lacks an independent calculation: %s", question.ID)
		}
		for _, option := range question.Options {
			result, err := Evaluate(question, Answer{OptionIDs: []string{option.ID}})
			if err != nil {
				t.Fatal(err)
			}
			expectCorrect := strings.ReplaceAll(option.Text, "−", "-") == expected
			if result.Correct != expectCorrect {
				t.Fatalf("%s: option %q graded %v; calculated answer %q", question.ID, option.Text, result.Correct, expected)
			}
		}
		checked++
	}
	if checked != len(want) {
		t.Fatalf("checked %d numeric questions, want %d", checked, len(want))
	}
}

func TestDLAnswerKeysRemainValidAfterPublicOptionShuffle(t *testing.T) {
	catalog, err := content.Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	topicExam := map[string]string{}
	for _, topic := range catalog.Topics {
		topicExam[topic.ID] = topic.ExamID
	}
	checked := 0
	for _, question := range catalog.Questions {
		if !question.Enabled || topicExam[question.TopicID] != "deep-learning-data-science" {
			continue
		}
		for seed := int64(0); seed < 4; seed++ {
			public := ToPublic(question, rand.New(rand.NewSource(seed)))
			answer := Answer{Matches: map[string]string{}}
			switch question.Type {
			case content.QuestionSingle, content.QuestionTrueFalse, content.QuestionNumeric:
				answer.OptionIDs = []string{question.CorrectAnswer}
				found := false
				for _, option := range public.Options {
					if option.ID == question.CorrectAnswer {
						found = true
					}
				}
				if !found {
					t.Fatalf("%s: answer ID lost after shuffle", question.ID)
				}
			case content.QuestionMultiple:
				answer.OptionIDs = question.CorrectAnswers
			case content.QuestionOrder:
				answer.Order = question.CorrectOrder
			case content.QuestionMatching:
				for _, item := range question.MatchItems {
					answer.Matches[item.ID] = item.CorrectOptionID
				}
			}
			result, err := Evaluate(question, answer)
			if err != nil || !result.Correct {
				t.Fatalf("%s seed %d: %+v %v", question.ID, seed, result, err)
			}
		}
		checked++
	}
	if checked != 131 {
		t.Fatalf("checked %d questions, want 131", checked)
	}
}
