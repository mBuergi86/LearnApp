package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCatalog = `{
  "exams":[
    {"id":"machine-learning","name":"Machine Learning","description":"ML","sort_order":1},
    {"id":"deep-learning-data-science","name":"Deep Learning","description":"DL","sort_order":2}
  ],
  "modules":[{"id":"ml-foundations","exam_id":"machine-learning","name":"Grundlagen","description":"Test","sort_order":1}],
  "pdfs":[{"file":"source.pdf","title":"Testquelle","exam_id":"machine-learning"}]
}`

const validTopic = `{
  "id":"example-topic","exam_id":"machine-learning","module_id":"ml-foundations","title":"Beispiel",
  "priority":"core","summary":"Kernidee","explanation":["Erklärung"],"steps":[],"example":"Beispiel",
  "confusions":[],"sources":[{"file":"source.pdf","path":"pdfs/source.pdf","pages":[2,3]}],
  "initial_status":"unlearned","sort_order":1
}`

func writeFixture(t *testing.T, catalog, topic string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "topics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "catalog.json"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "topics", "topic.json"), []byte(topic), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestLoadValidContent(t *testing.T) {
	catalog, err := Load(writeFixture(t, validCatalog, validTopic))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(catalog.Exams) != 2 || len(catalog.Modules) != 1 || len(catalog.Topics) != 1 {
		t.Fatalf("unexpected catalog sizes: %+v", catalog)
	}
	if catalog.Topics[0].Sources[0].Pages[1] != 3 {
		t.Fatal("source pages were not loaded")
	}
}

func TestLoadNormalizesOptionalTopicCollections(t *testing.T) {
	topic := strings.Replace(validTopic, `"steps":[]`, `"steps":null`, 1)
	topic = strings.Replace(topic, `"confusions":[]`, `"confusions":null`, 1)
	catalog, err := Load(writeFixture(t, validCatalog, topic))
	if err != nil {
		t.Fatal(err)
	}
	loaded := catalog.Topics[0]
	if loaded.Steps == nil || loaded.Confusions == nil {
		t.Fatalf("optional collections were not normalized: %+v", loaded)
	}
}

func TestLoadRejectsWrongOptionalCollectionType(t *testing.T) {
	invalid := strings.Replace(validTopic, `"steps":[]`, `"steps":"hello"`, 1)
	if _, err := Load(writeFixture(t, validCatalog, invalid)); err == nil || !strings.Contains(err.Error(), "parsen") {
		t.Fatalf("Load() error = %v, want parsing error for wrong steps type", err)
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	_, err := Load(writeFixture(t, validCatalog, `{broken`))
	if err == nil || !strings.Contains(err.Error(), "parsen") {
		t.Fatalf("Load() error = %v, want parse error", err)
	}
}

func TestLoadRejectsInvalidPriority(t *testing.T) {
	invalid := strings.Replace(validTopic, `"priority":"core"`, `"priority":"urgent"`, 1)
	_, err := Load(writeFixture(t, validCatalog, invalid))
	if err == nil || !strings.Contains(err.Error(), "Priorität") {
		t.Fatalf("Load() error = %v, want priority error", err)
	}
}

func TestLoadRejectsBrokenModuleReference(t *testing.T) {
	invalid := strings.Replace(validTopic, `"module_id":"ml-foundations"`, `"module_id":"missing"`, 1)
	_, err := Load(writeFixture(t, validCatalog, invalid))
	if err == nil || !strings.Contains(err.Error(), "ungültiges Thema") {
		t.Fatalf("Load() error = %v, want reference error", err)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	invalid := strings.Replace(validTopic, `"sort_order":1`, `"sort_order":1,"surprise":true`, 1)
	_, err := Load(writeFixture(t, validCatalog, invalid))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v, want unknown field error", err)
	}
}

func TestWithSourceURLsDoesNotExposePaths(t *testing.T) {
	catalog, err := Load(writeFixture(t, validCatalog, validTopic))
	if err != nil {
		t.Fatal(err)
	}
	catalog.Topics[0].Sources[0].Available = true
	public := catalog.WithSourceURLs()
	if public.Topics[0].Sources[0].Path != "" || public.Topics[0].Sources[0].URL == "" {
		t.Fatalf("public source = %+v", public.Topics[0].Sources[0])
	}
	if catalog.Topics[0].Sources[0].Path == "" {
		t.Fatal("original catalog was mutated")
	}
}

func TestNormalizeAndFormatSourcePages(t *testing.T) {
	pages, err := NormalizePages([]int{44, 42, 43, 42, 50, 51, 47})
	if err != nil {
		t.Fatal(err)
	}
	want := []int{42, 43, 44, 47, 50, 51}
	if len(pages) != len(want) {
		t.Fatalf("pages=%v want=%v", pages, want)
	}
	for index := range want {
		if pages[index] != want[index] {
			t.Fatalf("pages=%v want=%v", pages, want)
		}
	}
	if got := FormatPages(pages); got != "42–44, 47, 50–51" {
		t.Fatalf("FormatPages()=%q", got)
	}
	if got := FormatPages([]int{42, 43, 44}); got != "42–44" {
		t.Fatalf("contiguous FormatPages()=%q", got)
	}
}

func TestSplitPageRangesForLectureExamples(t *testing.T) {
	tests := []struct {
		name  string
		pages []int
		want  []PageRange
	}{
		{
			name:  "gradient descent",
			pages: []int{44, 45, 46, 47, 55, 56, 57},
			want: []PageRange{
				{Start: 44, End: 47, Label: "44–47"},
				{Start: 55, End: 57, Label: "55–57"},
			},
		},
		{
			name:  "backpropagation",
			pages: []int{42, 48, 49, 50, 51, 52, 53, 54, 61, 63},
			want: []PageRange{
				{Start: 42, End: 42, Label: "42"},
				{Start: 48, End: 54, Label: "48–54"},
				{Start: 61, End: 61, Label: "61"},
				{Start: 63, End: 63, Label: "63"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SplitPageRanges(test.pages)
			if len(got) != len(test.want) {
				t.Fatalf("SplitPageRanges()=%+v want=%+v", got, test.want)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Fatalf("range %d=%+v want=%+v", index, got[index], test.want[index])
				}
			}
		})
	}
}

func TestWithSourceURLsCreatesOneLinkPerPageRange(t *testing.T) {
	catalog, err := Load(writeFixture(t, validCatalog, strings.Replace(validTopic, `"pages":[2,3]`, `"pages":[42,48,49,50,51,52,53,54,61,63]`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	catalog.Topics[0].Sources[0].Available = true
	ranges := catalog.WithSourceURLs().Topics[0].Sources[0].PageRanges
	wantAnchors := []string{"#page=42", "#page=48", "#page=61", "#page=63"}
	if len(ranges) != len(wantAnchors) {
		t.Fatalf("page ranges=%+v", ranges)
	}
	for index, anchor := range wantAnchors {
		if !strings.HasSuffix(ranges[index].URL, anchor) {
			t.Fatalf("range %d URL=%q want suffix %q", index, ranges[index].URL, anchor)
		}
	}
}

func TestNormalizePagesRejectsZero(t *testing.T) {
	if _, err := NormalizePages([]int{1, 0, 2}); err == nil {
		t.Fatal("NormalizePages accepted page 0")
	}
}

func TestLoadSortsAndDeduplicatesSourcePages(t *testing.T) {
	unsorted := strings.Replace(validTopic, `"pages":[2,3]`, `"pages":[3,2,3]`, 1)
	catalog, err := Load(writeFixture(t, validCatalog, unsorted))
	if err != nil {
		t.Fatal(err)
	}
	pages := catalog.Topics[0].Sources[0].Pages
	if len(pages) != 2 || pages[0] != 2 || pages[1] != 3 {
		t.Fatalf("normalized pages=%v", pages)
	}
}

func TestLoadRejectsSourcePageZero(t *testing.T) {
	invalid := strings.Replace(validTopic, `"pages":[2,3]`, `"pages":[0,2]`, 1)
	if _, err := Load(writeFixture(t, validCatalog, invalid)); err == nil {
		t.Fatal("Load accepted source page 0")
	}
}

func TestPDFURLIncludesEncodedFileAndPageFragment(t *testing.T) {
	got := PDFURL("DeepLearning_Tag3_2026_Sequence-to-Sequence Modelle(1).pdf", 42)
	if got != "/pdf/DeepLearning_Tag3_2026_Sequence-to-Sequence%20Modelle%281%29.pdf#page=42" {
		t.Fatalf("PDFURL()=%q", got)
	}
}

func TestLoadRejectsUnregisteredAndTraversingPDFSources(t *testing.T) {
	for name, file := range map[string]string{"unregistered": "unknown.pdf", "traversal": "../source.pdf"} {
		t.Run(name, func(t *testing.T) {
			invalid := strings.Replace(validTopic, `"file":"source.pdf"`, `"file":"`+file+`"`, 1)
			_, err := Load(writeFixture(t, validCatalog, invalid))
			if err == nil || !strings.Contains(err.Error(), "Quelle") {
				t.Fatalf("Load() error=%v", err)
			}
		})
	}
}

func TestMissingPhysicalPDFIsWarningNotLoadFailure(t *testing.T) {
	directory := writeFixture(t, validCatalog, validTopic)
	catalog, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	warnings := CheckSourceAvailability(catalog, t.TempDir())
	if len(warnings) != 1 || catalog.PDFs[0].Available || catalog.Topics[0].Sources[0].Available {
		t.Fatalf("warnings=%v pdf=%+v source=%+v", warnings, catalog.PDFs[0], catalog.Topics[0].Sources[0])
	}
}

func TestLoadRejectsInvalidQuestionData(t *testing.T) {
	directory := writeFixture(t, validCatalog, validTopic)
	if err := os.Mkdir(filepath.Join(directory, "questions"), 0o755); err != nil {
		t.Fatal(err)
	}
	invalid := `{"questions":[{"id":"bad-question","topic_id":"example-topic","type":"single-choice","prompt":"Frage","explanation":"Text","difficulty":1,"enabled":true,"options":[{"id":"a","text":"A"},{"id":"b","text":"B"}],"correct_answer":"missing"}]}`
	if err := os.WriteFile(filepath.Join(directory, "questions", "bad.json"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(directory)
	if err == nil || !strings.Contains(err.Error(), "korrekte Antwort") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestProjectContentProvidesThirtyQuestionsPerExam(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	catalog, err := Load(filepath.Join(projectRoot, "content"))
	if err != nil {
		t.Fatal(err)
	}
	topicExams := make(map[string]string, len(catalog.Topics))
	for _, topic := range catalog.Topics {
		topicExams[topic.ID] = topic.ExamID
	}
	counts := map[string]int{}
	types := map[QuestionType]int{}
	enabled := 0
	for _, question := range catalog.Questions {
		if question.Enabled {
			counts[topicExams[question.TopicID]]++
			types[question.Type]++
			enabled++
		}
	}
	for _, exam := range catalog.Exams {
		if counts[exam.ID] < 30 {
			t.Fatalf("exam %s has %d enabled questions, want at least 30", exam.ID, counts[exam.ID])
		}
	}
	if len(catalog.PDFs) != 13 {
		t.Fatalf("project PDF catalog has %d entries, want 13", len(catalog.PDFs))
	}
	if len(catalog.Topics) < 8 || enabled < 60 {
		t.Fatalf("project content has %d topics and %d enabled questions, want at least 8 topics and at least 60 active questions", len(catalog.Topics), enabled)
	}
	for _, questionType := range []QuestionType{QuestionSingle, QuestionMultiple, QuestionTrueFalse, QuestionOrder, QuestionMatching, QuestionNumeric} {
		if types[questionType] == 0 {
			t.Fatalf("project content has no enabled question of type %q", questionType)
		}
	}
	if warnings := CheckSourceAvailability(catalog, projectRoot); len(warnings) != 0 {
		t.Fatalf("project PDFs unavailable: %v", warnings)
	}
}

func TestProjectPilotTopicsReferenceTheirCoursePDFs(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	catalog, err := Load(filepath.Join(projectRoot, "content"))
	if err != nil {
		t.Fatal(err)
	}
	if warnings := CheckSourceAvailability(catalog, projectRoot); len(warnings) != 0 {
		t.Fatalf("project PDFs unavailable: %v", warnings)
	}
	wantFiles := map[string]string{
		"supervised-unsupervised": "MachineLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf",
		"train-validation-test":   "MachineLearning_Tag2_ModelTraining_Selection_2026_ShaoJuWoo.pdf",
		"random-forest":           "Machine_Learning_Tag3_Ensemble_Learning_2026_ShaoJuWoo.pdf",
		"k-means":                 "Machine_Learning_Tag3_Clustering_2026_ShaoJuWoo.pdf",
		"gradient-descent":        "DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf",
		"backpropagation":         "DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf",
		"cnn-convolution":         "DeepLearning_Tag2_ComputerVision_2026.pdf",
		"query-key-value":         "DeepLearning_Tag3_Transformer_2026.pdf",
	}
	public := catalog.WithSourceURLs()
	for topicID, wantFile := range wantFiles {
		topic, found := FindTopic(&public, topicID)
		if !found {
			t.Fatalf("pilot topic %q missing", topicID)
		}
		if len(topic.Sources) != 1 {
			t.Fatalf("pilot topic %q has %d sources, want exactly one", topicID, len(topic.Sources))
		}
		source := topic.Sources[0]
		if source.File != wantFile || source.Title == "" || len(source.Pages) == 0 || !source.Available || source.PageLabel == "" || source.URL == "" {
			t.Fatalf("pilot topic %q source=%+v, want available %q with pages and URL", topicID, source, wantFile)
		}
		if !strings.HasSuffix(source.URL, fmt.Sprintf("#page=%d", source.Pages[0])) {
			t.Fatalf("pilot topic %q URL=%q does not open first referenced page %d", topicID, source.URL, source.Pages[0])
		}
	}
}

func TestProjectFinalContentSeparatesAllAndCoreExamPools(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	catalog, err := Load(filepath.Join(projectRoot, "content"))
	if err != nil {
		t.Fatal(err)
	}
	topicByID := make(map[string]Topic, len(catalog.Topics))
	for _, topic := range catalog.Topics {
		topicByID[topic.ID] = topic
	}
	counts := map[string]int{}
	coreCounts := map[string]int{}
	for _, question := range catalog.Questions {
		if !question.Enabled {
			continue
		}
		topic := topicByID[question.TopicID]
		counts[topic.ExamID]++
		if topic.Priority == PriorityCore {
			coreCounts[topic.ExamID]++
		}
	}
	for examID, want := range map[string]struct{ all, core int }{
		"machine-learning":           {all: 139, core: 122},
		"deep-learning-data-science": {all: 131, core: 105},
	} {
		if counts[examID] != want.all || coreCounts[examID] != want.core {
			t.Fatalf("exam %s pools all=%d core=%d, want all=%d core=%d", examID, counts[examID], coreCounts[examID], want.all, want.core)
		}
		if counts[examID] == coreCounts[examID] {
			t.Fatalf("exam %s all and core pools unexpectedly identical", examID)
		}
	}
}

func TestProjectFinalContentPopulatesEveryModule(t *testing.T) {
	projectRoot := filepath.Join("..", "..")
	catalog, err := Load(filepath.Join(projectRoot, "content"))
	if err != nil {
		t.Fatal(err)
	}
	byExam := map[string]int{}
	byModule := map[string]int{}
	for _, topic := range catalog.Topics {
		byExam[topic.ExamID]++
		byModule[topic.ModuleID]++
	}
	if byExam["machine-learning"] != 28 || byExam["deep-learning-data-science"] != 30 {
		t.Fatalf("final topic counts ML=%d DL/DS=%d, want 28 and 30", byExam["machine-learning"], byExam["deep-learning-data-science"])
	}
	if len(catalog.Modules) != 17 {
		t.Fatalf("module count=%d, want 17", len(catalog.Modules))
	}
	wantModuleTopics := map[string]int{
		"ml-foundations": 3, "ml-eda": 3, "ml-preprocessing": 5, "ml-regression": 3,
		"ml-logistic-regression": 2, "ml-model-selection": 8, "ml-decision-trees": 1,
		"ml-ensemble": 2, "ml-clustering": 1,
		"dl-foundations": 2, "dl-neural-networks": 3, "dl-training": 8, "dl-cnn": 5,
		"dl-sequence-models": 4, "dl-seq2seq": 2, "dl-attention": 1, "dl-transformer": 5,
	}
	for _, module := range catalog.Modules {
		if byModule[module.ID] == 0 {
			t.Fatalf("module %q has no final topics", module.ID)
		}
		if byModule[module.ID] != wantModuleTopics[module.ID] {
			t.Fatalf("module %q has %d topics, want %d", module.ID, byModule[module.ID], wantModuleTopics[module.ID])
		}
	}
	for _, topic := range catalog.Topics {
		if topic.Steps == nil || topic.Confusions == nil || (topic.Formula != nil && topic.Formula.Variables == nil) {
			t.Fatalf("topic %q exposes a nil optional collection", topic.ID)
		}
	}
}
