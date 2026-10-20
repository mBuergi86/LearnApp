package quiz

import (
	"fmt"
	"math/rand"
	"slices"

	"mldl-exam-trainer/internal/content"
)

type Answer struct {
	OptionIDs []string          `json:"option_ids,omitempty"`
	Order     []string          `json:"order,omitempty"`
	Matches   map[string]string `json:"matches,omitempty"`
}

type Result struct {
	Correct          bool              `json:"correct"`
	Explanation      string            `json:"explanation"`
	CorrectOptionIDs []string          `json:"correct_option_ids,omitempty"`
	CorrectOrder     []string          `json:"correct_order,omitempty"`
	CorrectMatches   map[string]string `json:"correct_matches,omitempty"`
}

type PublicQuestion struct {
	ID           string                   `json:"id"`
	TopicID      string                   `json:"topic_id"`
	Type         content.QuestionType     `json:"type"`
	Prompt       string                   `json:"prompt"`
	Context      string                   `json:"context,omitempty"`
	Difficulty   int                      `json:"difficulty"`
	Options      []content.QuestionOption `json:"options,omitempty"`
	MatchItems   []PublicMatchItem        `json:"match_items,omitempty"`
	MatchOptions []content.QuestionOption `json:"match_options,omitempty"`
}

type PublicMatchItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func Evaluate(question content.Question, answer Answer) (Result, error) {
	if !question.Enabled {
		return Result{}, fmt.Errorf("deaktivierte Frage kann nicht bewertet werden")
	}
	result := Result{Explanation: question.Explanation}
	switch question.Type {
	case content.QuestionSingle, content.QuestionTrueFalse, content.QuestionNumeric:
		if len(answer.OptionIDs) != 1 {
			return Result{}, fmt.Errorf("genau eine Antwort erforderlich")
		}
		result.CorrectOptionIDs = []string{question.CorrectAnswer}
		result.Correct = answer.OptionIDs[0] == question.CorrectAnswer
	case content.QuestionMultiple:
		if len(answer.OptionIDs) == 0 {
			return Result{}, fmt.Errorf("mindestens eine Antwort erforderlich")
		}
		result.CorrectOptionIDs = append([]string(nil), question.CorrectAnswers...)
		result.Correct = equalSet(answer.OptionIDs, question.CorrectAnswers)
	case content.QuestionOrder:
		if len(answer.Order) != len(question.CorrectOrder) {
			return Result{}, fmt.Errorf("vollständige Reihenfolge erforderlich")
		}
		result.CorrectOrder = append([]string(nil), question.CorrectOrder...)
		result.Correct = slices.Equal(answer.Order, question.CorrectOrder)
	case content.QuestionMatching:
		if len(answer.Matches) != len(question.MatchItems) {
			return Result{}, fmt.Errorf("vollständige Zuordnung erforderlich")
		}
		result.CorrectMatches = make(map[string]string, len(question.MatchItems))
		result.Correct = true
		for _, item := range question.MatchItems {
			result.CorrectMatches[item.ID] = item.CorrectOptionID
			if answer.Matches[item.ID] != item.CorrectOptionID {
				result.Correct = false
			}
		}
	default:
		return Result{}, fmt.Errorf("unbekannter Fragetyp %q", question.Type)
	}
	return result, nil
}

func ToPublic(question content.Question, random *rand.Rand) PublicQuestion {
	public := PublicQuestion{ID: question.ID, TopicID: question.TopicID, Type: question.Type, Prompt: question.Prompt, Context: question.Context, Difficulty: question.Difficulty}
	public.Options = append([]content.QuestionOption(nil), question.Options...)
	random.Shuffle(len(public.Options), func(i, j int) { public.Options[i], public.Options[j] = public.Options[j], public.Options[i] })
	public.MatchOptions = append([]content.QuestionOption(nil), question.MatchOptions...)
	random.Shuffle(len(public.MatchOptions), func(i, j int) {
		public.MatchOptions[i], public.MatchOptions[j] = public.MatchOptions[j], public.MatchOptions[i]
	})
	for _, item := range question.MatchItems {
		public.MatchItems = append(public.MatchItems, PublicMatchItem{ID: item.ID, Text: item.Text})
	}
	random.Shuffle(len(public.MatchItems), func(i, j int) {
		public.MatchItems[i], public.MatchItems[j] = public.MatchItems[j], public.MatchItems[i]
	})
	return public
}

func Filter(catalog *content.Catalog, examID, moduleID string) []content.Question {
	topicIDs := make(map[string]bool)
	for _, topic := range catalog.Topics {
		if topic.ExamID == examID && (moduleID == "" || topic.ModuleID == moduleID) {
			topicIDs[topic.ID] = true
		}
	}
	questions := make([]content.Question, 0)
	for _, question := range catalog.Questions {
		if question.Enabled && topicIDs[question.TopicID] {
			questions = append(questions, question)
		}
	}
	return questions
}

func equalSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}
