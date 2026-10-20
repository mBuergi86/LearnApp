package content

import (
	"fmt"
	"math"
	"strings"
)

func validateIllustration(figure Illustration) error {
	if strings.TrimSpace(figure.Title) == "" || strings.TrimSpace(figure.Caption) == "" || len(figure.Labels) == 0 || len(figure.Labels) > 8 {
		return fmt.Errorf("Titel, Bildbeschreibung und 1 bis 8 Beschriftungen erforderlich")
	}
	for _, label := range figure.Labels {
		if strings.TrimSpace(label) == "" || len([]rune(label)) > 90 {
			return fmt.Errorf("ungültige Bildbeschriftung")
		}
	}
	if len(figure.Values) > 64 {
		return fmt.Errorf("zu viele Datenzeilen")
	}
	for _, row := range figure.Values {
		if len(row) == 0 || len(row) > 8 {
			return fmt.Errorf("ungültige Datenzeile")
		}
		for _, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e9 {
				return fmt.Errorf("ungültiger Abbildungswert")
			}
		}
	}
	switch figure.Kind {
	case "flow", "comparison":
		if len(figure.Labels) < 2 || len(figure.Labels) > 6 || len(figure.Values) != 0 {
			return fmt.Errorf("Ablauf benötigt 2 bis 6 Schritte ohne Zahlentabelle")
		}
	case "nested", "residual":
		if len(figure.Labels) != 3 || len(figure.Values) != 0 {
			return fmt.Errorf("Abbildung benötigt genau drei Beschriftungen")
		}
	case "matrix":
		if len(figure.Values) == 0 || len(figure.Values) > 8 {
			return fmt.Errorf("Matrix benötigt 1 bis 8 Zeilen")
		}
		for _, row := range figure.Values {
			if len(row) != len(figure.Values[0]) {
				return fmt.Errorf("Matrixzeilen müssen gleich lang sein")
			}
		}
	case "bars", "network":
		if len(figure.Values) != 1 || len(figure.Values[0]) != len(figure.Labels) {
			return fmt.Errorf("genau ein Wert je Beschriftung erforderlich")
		}
		if figure.Kind == "network" {
			if len(figure.Labels) > 4 {
				return fmt.Errorf("Netzdiagramm unterstützt höchstens vier Schichten")
			}
			for _, count := range figure.Values[0] {
				if count < 1 || math.Trunc(count) != count {
					return fmt.Errorf("Neuronenanzahl muss eine positive ganze Zahl sein")
				}
			}
		}
	case "plot":
		if len(figure.Labels) != 2 || len(figure.Values) < 2 {
			return fmt.Errorf("Kurve benötigt zwei Achsen und mehrere Punkte")
		}
		for i, point := range figure.Values {
			if len(point) != 2 || (i > 0 && point[0] <= figure.Values[i-1][0]) {
				return fmt.Errorf("Kurvenpunkte müssen x/y-Paare mit wachsendem x sein")
			}
		}
	default:
		return fmt.Errorf("unbekannter Abbildungstyp %q", figure.Kind)
	}
	return nil
}
