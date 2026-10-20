package content

import (
	"fmt"
	"strings"
)

func validateFormula(formula Formula) error {
	validText := func(value string, limit int) bool {
		return strings.TrimSpace(value) != "" && len([]rune(value)) <= limit && !strings.ContainsRune(value, 0)
	}
	if len(formula.Latex) > 8 {
		return fmt.Errorf("höchstens acht Hauptformeln erlaubt")
	}
	for _, latex := range formula.Latex {
		if !validText(latex, 4096) {
			return fmt.Errorf("ungültiger LaTeX-Ausdruck")
		}
	}
	if formula.Example == nil {
		return nil
	} // existing text-only content remains valid
	example := formula.Example
	if !validText(example.Title, 160) || !validText(example.Result, 1024) || len(example.Given) < 1 || len(example.Given) > 12 || len(example.Steps) < 3 || len(example.Steps) > 12 {
		return fmt.Errorf("Beispiel benötigt Titel, Angaben, 3 bis 12 Schritte und Ergebnis")
	}
	for _, given := range example.Given {
		if !validText(given, 1024) {
			return fmt.Errorf("ungültige Beispielangabe")
		}
	}
	for _, step := range example.Steps {
		if !validText(step.Text, 1024) || (step.Latex != "" && !validText(step.Latex, 4096)) {
			return fmt.Errorf("ungültiger Beispielschritt")
		}
	}
	if example.ResultLatex != "" && !validText(example.ResultLatex, 4096) {
		return fmt.Errorf("ungültige Ergebnisformel")
	}
	return nil
}
