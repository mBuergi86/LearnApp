package content

import (
	"math"
	"strings"
	"testing"
)

func TestBothExamAreasProvideWorkedLearningMaterial(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, topic := range catalog.Topics {
		counts[topic.ExamID]++
		t.Run(topic.ID, func(t *testing.T) {
			if len(topic.Explanation) < 5 || len(topic.Steps) < 3 || len(topic.Confusions) < 2 || len([]rune(topic.Example)) < 170 {
				t.Fatal("missing explanation, procedure, worked example or misconceptions")
			}
			if len(topic.Illustrations) != 2 {
				t.Fatal("concept and example figures required")
			}
			if topic.Formula != nil {
				if topic.Formula.Expression == "" || topic.Formula.Meaning == "" || len(topic.Formula.Variables) == 0 {
					t.Fatal("formula must explain its notation")
				}
				for _, variable := range topic.Formula.Variables {
					if variable.Symbol == "" || variable.Meaning == "" {
						t.Fatal("empty formula symbol explanation")
					}
				}
			}
		})
	}
	if counts["machine-learning"] != 28 || counts["deep-learning-data-science"] != 30 {
		t.Fatalf("unexpected topic counts: %v", counts)
	}
	if len(catalog.PDFs) != 13 {
		t.Fatal("PDF inventory changed")
	}
}

func TestActivationMaterialDistinguishesArgumentsFromIntervals(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	topic, ok := FindTopic(catalog, "activation-functions")
	if !ok {
		t.Fatal("missing activation topic")
	}
	text := strings.Join(topic.Explanation, " ") + topic.Example + topic.Formula.Expression
	for _, marker := range []string{"max(0,−1)=0", "ReLU(−3)=0", "ReLU(−4)=0", "ReLU(4)=4", "kein", "Intervall", "(0,1)", "(−1,1)", "z=0"} {
		if !strings.Contains(text, marker) {
			t.Errorf("missing teaching point %q", marker)
		}
	}
	for _, z := range []float64{-4, -3, -1, 0, 4} {
		want := 0.0
		if z > 0 {
			want = z
		}
		if math.Max(0, z) != want {
			t.Fatal("ReLU boundary calculation failed")
		}
	}
}

func TestThresholdMaterialDoesNotPromiseMonotonicPrecision(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	topic, _ := FindTopic(catalog, "classification-threshold")
	text := strings.Join(topic.Explanation, " ") + topic.Example
	for _, marker := range []string{"nicht garantiert", "Validation", "≥", "TP=2,FP=1,FN=0,TN=1"} {
		if !strings.Contains(text, marker) {
			t.Errorf("missing threshold explanation %q", marker)
		}
	}
	precision := func(threshold float64) float64 {
		tp, fp := 0.0, 0.0
		for i, p := range []float64{0.9, 0.6, 0.4, 0.1} {
			if p >= threshold {
				if []int{1, 0, 1, 0}[i] == 1 {
					tp++
				} else {
					fp++
				}
			}
		}
		return tp / (tp + fp)
	}
	if precision(0.3) <= precision(0.5) {
		t.Fatal("worked precision counterexample is invalid")
	}
}

func TestNeuronWorkedChainAndBiasGradient(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	topic, _ := FindTopic(catalog, "backpropagation")
	x, w, b, y, eta := 3.0, 2.0, 1.0, 5.0, 0.1
	z := w*x + b
	a := math.Max(0, z)
	delta := a - y // positive ReLU has local derivative one
	gw, gb := x*delta, delta
	wn, bn := w-eta*gw, b-eta*gb
	if gw != 6 || gb != 2 || math.Abs(math.Max(0, wn*x+bn)-y) > 1e-12 {
		t.Fatal("worked chain is inconsistent")
	}
	for _, marker := range []string{"Gewichtsgradient=3·2=6", "Biasgradient=2", "w=1.4", "b=0.8"} {
		if !strings.Contains(topic.Example, marker) {
			t.Errorf("missing worked step %q", marker)
		}
	}
}
