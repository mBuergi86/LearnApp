package content

import (
	"strings"
	"testing"
)

func TestAllFormulaCardsHaveLatexAndWorkedExample(t *testing.T) {
	catalog, err := Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, topic := range catalog.Topics {
		if topic.Formula == nil {
			continue
		}
		count++
		t.Run(topic.ID, func(t *testing.T) {
			if len(topic.Formula.Latex) == 0 || topic.Formula.Example == nil {
				t.Fatal("LaTeX and worked example missing")
			}
			if err := validateFormula(*topic.Formula); err != nil {
				t.Fatal(err)
			}
		})
	}
	if count != 49 {
		t.Fatalf("formula cards %d; want 49", count)
	}
}

func TestFormulaValidationRetainsLegacyContentAndRejectsMalformedExamples(t *testing.T) {
	if err := validateFormula(Formula{Expression: "x=y"}); err != nil {
		t.Fatal(err)
	}
	base := Formula{Latex: []string{`x=y`}, Example: &FormulaExample{Title: "A", Given: []string{"x=1"}, Steps: []FormulaStep{{Text: "Choose x"}, {Text: "Apply rule"}, {Text: "Check"}}, Result: "y=1", ResultLatex: `y=1`}}
	if err := validateFormula(base); err != nil {
		t.Fatal(err)
	}
	for _, formula := range []Formula{
		{Latex: []string{""}},
		{Latex: []string{strings.Repeat("x", 4097)}},
		{Example: &FormulaExample{Title: "A", Given: []string{"x"}, Steps: []FormulaStep{{Text: "one"}}, Result: "result"}},
		{Example: &FormulaExample{Title: "A", Given: []string{""}, Steps: base.Example.Steps, Result: "result"}},
	} {
		if validateFormula(formula) == nil {
			t.Fatal("malformed formula accepted")
		}
	}
}
