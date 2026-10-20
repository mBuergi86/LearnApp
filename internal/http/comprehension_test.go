package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"mldl-exam-trainer/internal/auth"
	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/quiz"
)

func TestComprehensionScopesGradesAndDoesNotWriteLearningState(t *testing.T) {
	_, store, catalog := testEnvironment(t)
	check := content.Question{ID: "check-example", TopicID: "example", Type: content.QuestionTrueFalse, Prompt: "A statement", Explanation: "Why true, and why not false.", Difficulty: 1, Enabled: true, Options: []content.QuestionOption{{ID: "true", Text: "Richtig"}, {ID: "false", Text: "Falsch"}}, CorrectAnswer: "true"}
	other, disabled := check, check
	other.ID, other.TopicID = "check-other", "dl-example"
	disabled.ID, disabled.Enabled = "check-disabled", false
	catalog.Comprehension = []content.Question{check, other}
	catalog.Questions = append(catalog.Questions, disabled)
	handler := NewHandler(store, catalog, t.TempDir(), fstest.MapFS{"index.html": {Data: []byte("test")}})
	before, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/comprehension?topic=example", nil))
	var response struct {
		TopicID   string                `json:"topic_id"`
		Questions []quiz.PublicQuestion `json:"questions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || response.TopicID != "example" || len(response.Questions) != 1 || response.Questions[0].ID != check.ID {
		t.Fatalf("scope: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "correct_answer") || strings.Contains(w.Body.String(), "explanation") {
		t.Fatal("answer leaked before submission")
	}
	for _, tc := range []struct {
		name, body string
		status     int
		correct    bool
	}{
		{"correct", `{"topic_id":"example","question_id":"check-example","response":{"option_ids":["true"]}}`, 200, true},
		{"incorrect", `{"topic_id":"example","question_id":"check-example","response":{"option_ids":["false"]}}`, 200, false},
		{"other topic", `{"topic_id":"example","question_id":"check-other","response":{"option_ids":["true"]}}`, 404, false},
		{"disabled", `{"topic_id":"example","question_id":"check-disabled","response":{"option_ids":["true"]}}`, 404, false},
		{"ordinary quiz", `{"topic_id":"example","question_id":"enabled-question","response":{"option_ids":["a"]}}`, 404, false},
		{"injection", `{"topic_id":"' OR 1=1 --","question_id":"check-example","response":{"option_ids":["true"]}}`, 404, false},
		{"invalid option", `{"topic_id":"example","question_id":"check-example","response":{"option_ids":["anything"]}}`, 400, false},
		{"missing response", `{"topic_id":"example","question_id":"check-example","response":{}}`, 400, false},
		{"multiple choices", `{"topic_id":"example","question_id":"check-example","response":{"option_ids":["true","false"]}}`, 400, false},
		{"client answer key", `{"topic_id":"example","question_id":"check-example","correct":true}`, 400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, jsonRequest(http.MethodPost, "/api/comprehension/answers", tc.body))
			if w.Code != tc.status {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if tc.status == 200 {
				var result quiz.Result
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Correct != tc.correct || result.Explanation != check.Explanation || !reflect.DeepEqual(result.CorrectOptionIDs, []string{"true"}) {
					t.Fatalf("result: %+v", result)
				}
			}
		})
	}
	after, err := store.ExportBackup()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Data, after.Data) {
		t.Fatal("comprehension changed reviews, mastery, progress, scheduler or exam state")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/catalog", nil))
	if strings.Contains(w.Body.String(), "check-example") {
		t.Fatal("catalog exposes supplemental answers")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/comprehension?topic=unknown", nil))
	if w.Code != 404 {
		t.Fatal("unknown topic accepted")
	}
}

func TestComprehensionRetainsSessionAndCSRFProtection(t *testing.T) {
	handler, _, _, session, csrf := authenticatedHandler(t, auth.RoleUser)
	for _, path := range []string{"/api/comprehension?topic=example", "/api/comprehension/answers"} {
		w := httptest.NewRecorder()
		method := http.MethodGet
		if strings.HasSuffix(path, "answers") {
			method = http.MethodPost
		}
		handler.ServeHTTP(w, jsonRequest(method, path, `{}`))
		if w.Code != 401 {
			t.Fatalf("unprotected endpoint %s: %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, requestWithSession(http.MethodPost, "/api/comprehension/answers", session, csrf, `{}`, false))
	if w.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d", w.Code)
	}
}
