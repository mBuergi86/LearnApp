package quiz

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
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
	median := func(values ...float64) float64 {
		sort.Float64s(values)
		return values[len(values)/2]
	}
	positiveCount := func(values []float64, threshold float64) float64 {
		count := 0.0
		for _, value := range values {
			if value >= threshold {
				count++
			}
		}
		return count
	}
	want := map[string]string{
		"ml-holdout-validation-share":  decimal(1000 * 0.2),
		"ml-lifecycle-evaluate":        fmt.Sprintf("%d CHF", 2000*90+50000),
		"ml-features-matrix":           decimal(8 * 3),
		"ml-regression-label-type":     fmt.Sprintf("%d CHF", 10500-12000),
		"ml-eda-before-model":          fmt.Sprintf("%g %%", 18.0/120*100),
		"ml-eda-outlier":               fmt.Sprintf("%.3f", (math.Pow(1-3, 2)+math.Pow(3-3, 2)+math.Pow(5-3, 2))/3),
		"ml-correlation-coefficient":   decimal(6 / math.Sqrt(2*18)),
		"ml-missing-values-problem":    decimal((4 + 8 + 12) / 3),
		"ml-noise-binning":             decimal(median(3, 99, 5)),
		"ml-scaling-standard":          decimal((16 - 10) / 2),
		"ml-encoding-periodic":         fmt.Sprintf("(%g,%g)", math.Sin(0), math.Cos(0)),
		"ml-collinearity-detect":       decimal(1*3 + 2*(2*3)),
		"ml-regression-prediction":     decimal(3*4 + 2),
		"ml-least-squares-parameters":  decimal((1.0*3 + 2*6) / (1 + 4)),
		"ml-regression-bias-parameter": decimal(4*0 + 7),
		"ml-logistic-probability":      decimal(1 / (1 + 1.0/4)),
		"ml-threshold-role":            decimal(positiveCount([]float64{0.8, 0.5, 0.3}, 0.5)),
		"ml-loss-optimize":             decimal((math.Pow(3-1, 2) + math.Pow(3-4, 2)) / 2),
		"ml-gradient-convergence":      decimal(5 - 0.1*(2*(5-2))),
		"ml-overfit-complexity":        decimal(17 - 2),
		"ml-confusion-metrics":         decimal(12.0 / (12 + 4)),
		"ml-performance-f1":            fmt.Sprintf("%.3f", 2.0*6/(2*6+2+4)),
		"ml-hyperparameter-selection":  decimal((math.Pow(3-5, 2) + math.Pow(7-5, 2)) / 2),
		"ml-crossval-average":          decimal(3 * 2 * 5),
		"ml-tree-impurity":             decimal(1 - math.Pow(3.0/4, 2) - math.Pow(1.0/4, 2)),
		"ml-ensemble-purpose":          decimal((6 + 9 + 12) / 3),
		"ml-random-forest-tree-count":  decimal(positiveCount([]float64{1, 0, 1, 1, 0}, 1)),
		"ml-random-forest-leaf-count":  decimal((8 + 12 + 10) / 3),
		"ml-kmeans-k-value":            decimal((2 + 4 + 6) / 3),
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
	for exam, want := range map[string]int{"machine-learning": 122, "deep-learning-data-science": 134} {
		if counts[exam] != want || len(types[exam]) != 6 {
			t.Fatalf("%s: count %d, types %d", exam, counts[exam], len(types[exam]))
		}
	}
}
