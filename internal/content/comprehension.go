package content

// ComprehensionQuestions deliberately stays inside one topic. Existing
// true/false questions are reused; supplemental checks never enter exam pools.
func ComprehensionQuestions(catalog *Catalog, topicID string) []Question {
	checks := []Question{}
	for _, pool := range [][]Question{catalog.Questions, catalog.Comprehension} {
		for _, question := range pool {
			if question.Enabled && question.TopicID == topicID && question.Type == QuestionTrueFalse {
				checks = append(checks, question)
			}
		}
	}
	return checks
}
