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
		"dl-gradient-numeric":                     decimal(4 - 0.1*8),
		"dl-gradient-negative-value":              decimal(4 - 0.5*(-2)),
		"dl-cnn-dot-product":                      decimal(5 * 5 * 3),
		"dl-backprop-cost-numeric":                fmt.Sprintf("%.3f", 0.5*(math.Pow(1-0.8, 2)+math.Pow(0-0.3, 2))),
		"dl-cnn-rgb-depth":                        "3",
		"dl-cnn-output-width":                     decimal(math.Floor((32-5)/1) + 1),
		"dl-cnn-output-depth":                     "6",
		"dl-qkv-embedding-size":                   decimal(8 * 2),
		"dl-activation-functions-core":            "(0,0,2)",
		"dl-activation-functions-detail":          decimal(3.0 / (1 + 2 + 3)),
		"dl-artificial-neuron-core":               decimal(math.Max(0, 0.5*2+(-2)*(-1)-1)),
		"dl-batch-iteration-epoch-core":           decimal(1000 / 250),
		"dl-batch-iteration-epoch-detail":         decimal(math.Ceil(1050.0 / 250)),
		"dl-batch-normalization-misconception":    "(-1,1)",
		"dl-classic-attention-detail":             fmt.Sprintf("(%.1f,%.1f)", 0.2+0.3, 0.5*2+0.3),
		"dl-cnn-architecture-detail":              decimal(10 * (5*5*6 + 1)),
		"dl-cnn-spatial-dimensions-core":          "3 × 3",
		"dl-cnn-spatial-dimensions-detail":        decimal(10 * (5*5*3 + 1)),
		"dl-computer-vision-images-core":          decimal(32 * 32 * 3),
		"dl-dl-loss-functions-detail":             decimal((1 + 4) / 2.0),
		"dl-dl-regularization-detail":             decimal(0.4 + 0.1*(4+1)),
		"dl-forward-backward-pass-detail":         decimal((1*2 - 4) * 2),
		"dl-lstm-detail":                          decimal(0.8*2 + 0.5*(-0.4)),
		"dl-multi-head-attention-detail":          "5 × 12",
		"dl-neural-network-architecture-detail":   decimal(3*2 + 2),
		"dl-pooling-resnet-core":                  "6 und 3.75",
		"dl-pooling-resnet-misconception":         "(2.5,-0.8)",
		"dl-positional-information-misconception": "(0,1,0,1)",
		"dl-rnn-basics-detail":                    fmt.Sprintf("%.3f", math.Tanh(0.5*math.Tanh(1))),
		"dl-rnn-bptt-detail":                      decimal(0.2 - 0.1 + 0.4),
		"dl-self-attention-detail":                fmt.Sprintf("(%.1f,%.1f)", 0.1*2+0.2, 0.7*4+0.2),
		"dl-text-representations-misconception":   decimal(100 * 8),
		"dl-transformer-architecture-detail":      "3 × 4",
		"dl-vanishing-gradients-detail":           fmt.Sprintf("%.6f", math.Pow(0.5, 10)),
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
	if checked != 134 {
		t.Fatalf("checked %d questions, want 134", checked)
	}
}
