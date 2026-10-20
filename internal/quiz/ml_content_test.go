package quiz

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	"mldl-exam-trainer/internal/content"
)

// Each expected option is calculated independently of the JSON answer key.
// The option text is checked as well, so changing the key cannot make a wrong
// numerical result silently pass.
func TestMLNumericAnswerKeysAgainstWorkedCalculations(t *testing.T) {
	catalog, err := content.Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	decimal := func(x float64) string { return fmt.Sprintf("%g", x) }
	want := map[string]string{
		"ml-scaling-standard":     decimal((16 - 10) / 2),
		"ml-logistic-probability": decimal(1 / (1 + 1.0/4)),
		"ml-loss-optimize":        decimal((math.Pow(3-1, 2) + math.Pow(3-4, 2)) / 2),
		"ml-confusion-metrics":    decimal(12.0 / (12 + 4)),
		"ml-performance-f1":       fmt.Sprintf("%.3f", 2.0*6/(2*6+2+4)),
		"ml-tree-impurity":        decimal(1 - math.Pow(3.0/4, 2) - math.Pow(1.0/4, 2)),
	}
	checked := 0
	for _, q := range catalog.Questions {
		if !q.Enabled || q.Type != content.QuestionNumeric || !strings.HasPrefix(q.ID, "ml-") {
			continue
		}
		t.Run(q.ID, func(t *testing.T) {
			expected, ok := want[q.ID]
			if !ok {
				t.Fatalf("no independent calculation for %s", q.ID)
			}
			matches := 0
			for _, option := range q.Options {
				expectedCorrect := strings.ReplaceAll(option.Text, "−", "-") == expected
				if expectedCorrect {
					matches++
				}
				result, err := Evaluate(q, Answer{OptionIDs: []string{option.ID}})
				if err != nil || result.Correct != expectedCorrect {
					t.Fatalf("option %q: expected answer %q; grading %+v, error %v", option.Text, expected, result, err)
				}
			}
			if matches != 1 {
				t.Fatalf("calculated answer %q occurs %d times", expected, matches)
			}
			if q.Context == "" || len([]rune(q.Explanation)) < 140 {
				t.Fatal("numeric task needs assumptions, calculation and misconception feedback")
			}
		})
		checked++
	}
	if checked != len(want) {
		t.Fatalf("checked %d; expected %d", checked, len(want))
	}
}

func TestBothExamPoolsGradeAllSixTypesAfterShuffle(t *testing.T) {
	catalog, err := content.Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	examByTopic := map[string]string{}
	for _, topic := range catalog.Topics {
		examByTopic[topic.ID] = topic.ExamID
	}
	counts := map[string]int{}
	types := map[string]map[content.QuestionType]bool{}
	for _, q := range catalog.Questions {
		if !q.Enabled {
			continue
		}
		exam := examByTopic[q.TopicID]
		counts[exam]++
		if types[exam] == nil {
			types[exam] = map[content.QuestionType]bool{}
		}
		types[exam][q.Type] = true
		for seed := int64(0); seed < 4; seed++ {
			public := ToPublic(q, rand.New(rand.NewSource(seed)))
			answer := Answer{Matches: map[string]string{}}
			switch q.Type {
			case content.QuestionSingle, content.QuestionTrueFalse, content.QuestionNumeric:
				answer.OptionIDs = []string{q.CorrectAnswer}
				found := false
				for _, option := range public.Options {
					if option.ID == q.CorrectAnswer {
						found = true
					}
				}
				if !found {
					t.Fatalf("%s lost correct option after shuffle", q.ID)
				}
			case content.QuestionMultiple:
				answer.OptionIDs = q.CorrectAnswers
			case content.QuestionOrder:
				answer.Order = q.CorrectOrder
			case content.QuestionMatching:
				for _, item := range q.MatchItems {
					answer.Matches[item.ID] = item.CorrectOptionID
				}
			}
			result, err := Evaluate(q, answer)
			if err != nil || !result.Correct {
				t.Fatalf("%s seed %d: %+v %v", q.ID, seed, result, err)
			}
		}
	}
	for exam, want := range map[string]int{"machine-learning": 139, "deep-learning-data-science": 131} {
		if counts[exam] != want || len(types[exam]) != 6 {
			t.Fatalf("%s: count %d, types %d", exam, counts[exam], len(types[exam]))
		}
	}
}
