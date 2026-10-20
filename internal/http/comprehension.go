package httpserver

import (
	"net/http"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/quiz"
)

func (s *server) getComprehension(w http.ResponseWriter, r *http.Request) {
	topicID := r.URL.Query().Get("topic")
	if !s.hasTopic(topicID) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Unbekanntes Thema."})
		return
	}
	checks := content.ComprehensionQuestions(s.catalog, topicID)
	public := make([]quiz.PublicQuestion, len(checks))
	s.randomMu.Lock()
	for i, question := range checks {
		public[i] = quiz.ToPublic(question, s.random)
	}
	s.randomMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"topic_id": topicID, "questions": public})
}

func (s *server) answerComprehension(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TopicID    string      `json:"topic_id"`
		QuestionID string      `json:"question_id"`
		Response   quiz.Answer `json:"response"`
	}
	if err := decodeJSONRequest(w, r, 8*1024, &request); err != nil {
		writeRequestError(w, err, "Ungültige Verständnisantwort.")
		return
	}
	if s.hasTopic(request.TopicID) {
		for _, question := range content.ComprehensionQuestions(s.catalog, request.TopicID) {
			if question.ID != request.QuestionID {
				continue
			}
			validOption := false
			if len(request.Response.OptionIDs) == 1 && len(request.Response.Order) == 0 && len(request.Response.Matches) == 0 {
				for _, option := range question.Options {
					validOption = validOption || option.ID == request.Response.OptionIDs[0]
				}
			}
			if !validOption {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bitte Richtig oder Falsch auswählen."})
				return
			}
			result, err := quiz.Evaluate(question, request.Response)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Ungültige Verständnisantwort."})
				return
			}
			// This is explanatory practice, not an exam/review. No DB write,
			// mastery change, retry plan or scheduler advancement happens here.
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Frage gehört nicht zum gewählten Thema."})
}

func (s *server) hasTopic(id string) bool {
	for _, topic := range s.catalog.Topics {
		if topic.ID == id {
			return true
		}
	}
	return false
}
