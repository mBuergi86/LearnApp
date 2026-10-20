package content

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var validID = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type catalogFile struct {
	Exams   []Exam      `json:"exams"`
	Modules []Module    `json:"modules"`
	PDFs    []PDFSource `json:"pdfs"`
}

func Load(directory string) (*Catalog, error) {
	catalogData, err := os.ReadFile(filepath.Join(directory, "catalog.json"))
	if err != nil {
		return nil, fmt.Errorf("Katalog lesen: %w", err)
	}
	var base catalogFile
	if err := decodeJSON(catalogData, &base); err != nil {
		return nil, fmt.Errorf("Katalog parsen: %w", err)
	}

	topicsDirectory := filepath.Join(directory, "topics")
	entries, err := os.ReadDir(topicsDirectory)
	if err != nil {
		return nil, fmt.Errorf("Themenverzeichnis lesen: %w", err)
	}
	topics := make([]Topic, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(topicsDirectory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("Thema %s lesen: %w", entry.Name(), err)
		}
		var topic Topic
		if err := decodeJSON(data, &topic); err != nil {
			return nil, fmt.Errorf("Thema %s parsen: %w", entry.Name(), err)
		}
		normalizeTopicOptionalFields(&topic)
		topics = append(topics, topic)
	}
	questions, err := loadQuestions(filepath.Join(directory, "questions"))
	if err != nil {
		return nil, err
	}

	catalog := &Catalog{Exams: base.Exams, Modules: base.Modules, Topics: topics, PDFs: base.PDFs, Questions: questions}
	checksData, err := os.ReadFile(filepath.Join(directory, "comprehension.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("Verständnisfragen lesen: %w", err)
	}
	if err == nil {
		var checks struct {
			Questions []Question `json:"questions"`
		}
		if err := decodeJSON(checksData, &checks); err != nil {
			return nil, fmt.Errorf("Verständnisfragen parsen: %w", err)
		}
		catalog.Comprehension = checks.Questions
	}
	if err := catalog.normalizeSources(); err != nil {
		return nil, err
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	sort.Slice(catalog.Exams, func(i, j int) bool { return catalog.Exams[i].SortOrder < catalog.Exams[j].SortOrder })
	sort.Slice(catalog.Modules, func(i, j int) bool { return catalog.Modules[i].SortOrder < catalog.Modules[j].SortOrder })
	sort.Slice(catalog.Topics, func(i, j int) bool { return catalog.Topics[i].SortOrder < catalog.Topics[j].SortOrder })
	sort.Slice(catalog.Questions, func(i, j int) bool { return catalog.Questions[i].ID < catalog.Questions[j].ID })
	return catalog, nil
}

// normalizeTopicOptionalFields keeps optional JSON collections safe for API
// consumers: null and omitted collections are represented as empty arrays.
func normalizeTopicOptionalFields(topic *Topic) {
	if topic.Steps == nil {
		topic.Steps = []string{}
	}
	if topic.Confusions == nil {
		topic.Confusions = []Confusion{}
	}
	if topic.Formula != nil && topic.Formula.Variables == nil {
		topic.Formula.Variables = []Variable{}
	}
}

func loadQuestions(directory string) ([]Question, error) {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Fragenverzeichnis lesen: %w", err)
	}
	questions := make([]Question, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("Fragen %s lesen: %w", entry.Name(), err)
		}
		var document struct {
			Questions []Question `json:"questions"`
		}
		if err := decodeJSON(data, &document); err != nil {
			return nil, fmt.Errorf("Fragen %s parsen: %w", entry.Name(), err)
		}
		questions = append(questions, document.Questions...)
	}
	return questions, nil
}

func decodeJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("unerwartete weitere JSON-Daten")
	}
	return nil
}

func (c *Catalog) Validate() error {
	if len(c.Exams) != 2 {
		return fmt.Errorf("Content validieren: genau zwei Prüfungsbereiche erforderlich")
	}
	exams := make(map[string]bool, len(c.Exams))
	modules := make(map[string]Module, len(c.Modules))
	topics := make(map[string]bool, len(c.Topics))
	questions := make(map[string]bool, len(c.Questions))
	pdfs := make(map[string]PDFSource, len(c.PDFs))
	for _, exam := range c.Exams {
		if !validID.MatchString(exam.ID) || strings.TrimSpace(exam.Name) == "" || exams[exam.ID] {
			return fmt.Errorf("Content validieren: ungültiger oder doppelter Prüfungsbereich %q", exam.ID)
		}
		exams[exam.ID] = true
	}
	for _, pdf := range c.PDFs {
		if !validPDFFileName(pdf.File) || strings.TrimSpace(pdf.Title) == "" || !exams[pdf.ExamID] || pdfs[pdf.File].File != "" {
			return fmt.Errorf("Content validieren: ungültige oder doppelte PDF-Quelle %q", pdf.File)
		}
		pdfs[pdf.File] = pdf
	}
	for _, module := range c.Modules {
		if !validID.MatchString(module.ID) || strings.TrimSpace(module.Name) == "" || !exams[module.ExamID] {
			return fmt.Errorf("Content validieren: ungültiges Modul %q", module.ID)
		}
		if _, exists := modules[module.ID]; exists {
			return fmt.Errorf("Content validieren: doppeltes Modul %q", module.ID)
		}
		modules[module.ID] = module
	}
	for _, topic := range c.Topics {
		module, ok := modules[topic.ModuleID]
		if !validID.MatchString(topic.ID) || topics[topic.ID] || !ok || module.ExamID != topic.ExamID {
			return fmt.Errorf("Content validieren: ungültiges Thema %q", topic.ID)
		}
		if topic.Priority != PriorityCore && topic.Priority != PrioritySupport && topic.Priority != PriorityReference {
			return fmt.Errorf("Content validieren: ungültige Priorität für %q", topic.ID)
		}
		if strings.TrimSpace(topic.Title) == "" || strings.TrimSpace(topic.Summary) == "" || len(topic.Explanation) == 0 || len(topic.Sources) == 0 {
			return fmt.Errorf("Content validieren: Thema %q ist unvollständig", topic.ID)
		}
		if topic.InitialStatus != "unlearned" {
			return fmt.Errorf("Content validieren: initial_status für %q muss unlearned sein", topic.ID)
		}
		for _, illustration := range topic.Illustrations {
			if err := validateIllustration(illustration); err != nil {
				return fmt.Errorf("Content validieren: Abbildung für %q: %w", topic.ID, err)
			}
		}
		if topic.Formula != nil {
			if err := validateFormula(*topic.Formula); err != nil {
				return fmt.Errorf("Content validieren: Formel für %q: %w", topic.ID, err)
			}
		}
		for _, source := range topic.Sources {
			pdf, registered := pdfs[source.File]
			if !validPDFFileName(source.File) || !registered || pdf.ExamID != topic.ExamID || source.Path != filepath.ToSlash(filepath.Join("content", "pdfs", source.File)) || len(source.Pages) == 0 {
				return fmt.Errorf("Content validieren: ungültige Quelle für %q", topic.ID)
			}
			for _, page := range source.Pages {
				if page < 1 {
					return fmt.Errorf("Content validieren: ungültige Seite für %q", topic.ID)
				}
			}
		}
		topics[topic.ID] = true
	}
	for _, question := range c.Questions {
		if !validID.MatchString(question.ID) || questions[question.ID] || !topics[question.TopicID] {
			return fmt.Errorf("Content validieren: ungültige Frage %q", question.ID)
		}
		if err := validateQuestion(question); err != nil {
			return fmt.Errorf("Content validieren: Frage %q: %w", question.ID, err)
		}
		questions[question.ID] = true
	}
	for _, check := range c.Comprehension {
		if !validID.MatchString(check.ID) || questions[check.ID] || !topics[check.TopicID] || check.Type != QuestionTrueFalse || !check.Enabled {
			return fmt.Errorf("Content validieren: ungültige Verständnisfrage %q", check.ID)
		}
		if err := validateQuestion(check); err != nil {
			return fmt.Errorf("Content validieren: Verständnisfrage %q: %w", check.ID, err)
		}
		questions[check.ID] = true
	}
	return nil
}

func validateQuestion(question Question) error {
	if strings.TrimSpace(question.Prompt) == "" || strings.TrimSpace(question.Explanation) == "" || question.Difficulty < 1 || question.Difficulty > 3 {
		return fmt.Errorf("Pflichtfelder oder Schwierigkeit ungültig")
	}
	validTypes := map[QuestionType]bool{QuestionSingle: true, QuestionMultiple: true, QuestionTrueFalse: true, QuestionOrder: true, QuestionMatching: true, QuestionNumeric: true}
	if !validTypes[question.Type] {
		return fmt.Errorf("unbekannter Fragetyp %q", question.Type)
	}
	optionIDs := map[string]bool{}
	optionTexts := map[string]bool{}
	for _, option := range question.Options {
		if !validID.MatchString(option.ID) || strings.TrimSpace(option.Text) == "" || optionIDs[option.ID] {
			return fmt.Errorf("ungültige oder doppelte Option %q", option.ID)
		}
		optionIDs[option.ID] = true
		key := strings.ToLower(strings.Join(strings.Fields(option.Text), " "))
		if optionTexts[key] {
			return fmt.Errorf("inhaltlich identische Antwortoption %q", option.ID)
		}
		optionTexts[key] = true
	}
	contains := func(ids []string, id string) bool {
		for _, candidate := range ids {
			if candidate == id {
				return true
			}
		}
		return false
	}
	switch question.Type {
	case QuestionSingle, QuestionTrueFalse, QuestionNumeric:
		if len(question.Options) < 2 || !optionIDs[question.CorrectAnswer] || len(question.CorrectAnswers)+len(question.CorrectOrder)+len(question.MatchItems) > 0 {
			return fmt.Errorf("genau eine gültige korrekte Antwort erforderlich")
		}
		if question.Type == QuestionTrueFalse && len(question.Options) != 2 {
			return fmt.Errorf("Richtig/Falsch benötigt zwei Optionen")
		}
	case QuestionMultiple:
		if len(question.Options) < 3 || len(question.CorrectAnswers) < 2 {
			return fmt.Errorf("Multiple Select benötigt mehrere korrekte Antworten")
		}
		seen := map[string]bool{}
		for _, id := range question.CorrectAnswers {
			if !optionIDs[id] || seen[id] {
				return fmt.Errorf("ungültige korrekte Option %q", id)
			}
			seen[id] = true
		}
	case QuestionOrder:
		if len(question.Options) < 3 || len(question.CorrectOrder) != len(question.Options) {
			return fmt.Errorf("Reihenfolge muss alle Optionen enthalten")
		}
		for id := range optionIDs {
			if !contains(question.CorrectOrder, id) {
				return fmt.Errorf("Option %q fehlt in Reihenfolge", id)
			}
		}
	case QuestionMatching:
		if len(question.MatchItems) < 2 || len(question.MatchOptions) < 2 {
			return fmt.Errorf("Zuordnung benötigt Elemente und Ziele")
		}
		matchOptions := map[string]bool{}
		matchTexts := map[string]bool{}
		for _, option := range question.MatchOptions {
			if !validID.MatchString(option.ID) || strings.TrimSpace(option.Text) == "" || matchOptions[option.ID] {
				return fmt.Errorf("ungültiges Zuordnungsziel %q", option.ID)
			}
			matchOptions[option.ID] = true
			key := strings.ToLower(strings.Join(strings.Fields(option.Text), " "))
			if matchTexts[key] {
				return fmt.Errorf("inhaltlich identisches Zuordnungsziel %q", option.ID)
			}
			matchTexts[key] = true
		}
		items := map[string]bool{}
		for _, item := range question.MatchItems {
			if !validID.MatchString(item.ID) || strings.TrimSpace(item.Text) == "" || items[item.ID] || !matchOptions[item.CorrectOptionID] {
				return fmt.Errorf("ungültiges Zuordnungselement %q", item.ID)
			}
			items[item.ID] = true
		}
	}
	return nil
}

func (c *Catalog) WithSourceURLs() Catalog {
	copyCatalog := *c
	copyCatalog.Topics = append([]Topic(nil), c.Topics...)
	for topicIndex := range copyCatalog.Topics {
		copyCatalog.Topics[topicIndex].Sources = append([]Source(nil), copyCatalog.Topics[topicIndex].Sources...)
		for sourceIndex := range copyCatalog.Topics[topicIndex].Sources {
			source := &copyCatalog.Topics[topicIndex].Sources[sourceIndex]
			source.Path = ""
			source.PageLabel = FormatPages(source.Pages)
			source.PageRanges = SplitPageRanges(source.Pages)
			if source.Available {
				source.URL = PDFURL(source.File, source.Pages[0])
				for rangeIndex := range source.PageRanges {
					source.PageRanges[rangeIndex].URL = PDFURL(source.File, source.PageRanges[rangeIndex].Start)
				}
			}
		}
	}
	return copyCatalog
}

func (c *Catalog) normalizeSources() error {
	pdfs := make(map[string]PDFSource, len(c.PDFs))
	for _, pdf := range c.PDFs {
		pdfs[pdf.File] = pdf
	}
	for topicIndex := range c.Topics {
		for sourceIndex := range c.Topics[topicIndex].Sources {
			source := &c.Topics[topicIndex].Sources[sourceIndex]
			pages, err := NormalizePages(source.Pages)
			if err != nil {
				return fmt.Errorf("Content validieren: Quelle für %q: %w", c.Topics[topicIndex].ID, err)
			}
			source.Pages = pages
			if pdf, found := pdfs[source.File]; found && source.Title == "" {
				source.Title = pdf.Title
			}
			source.Path = filepath.ToSlash(filepath.Join("content", "pdfs", source.File))
		}
	}
	return nil
}

func NormalizePages(pages []int) ([]int, error) {
	unique := make(map[int]bool, len(pages))
	for _, page := range pages {
		if page <= 0 {
			return nil, fmt.Errorf("Seiten müssen grösser als 0 sein")
		}
		unique[page] = true
	}
	result := make([]int, 0, len(unique))
	for page := range unique {
		result = append(result, page)
	}
	sort.Ints(result)
	return result, nil
}

func FormatPages(pages []int) string {
	ranges := SplitPageRanges(pages)
	parts := make([]string, len(ranges))
	for index, pageRange := range ranges {
		parts[index] = pageRange.Label
	}
	return strings.Join(parts, ", ")
}

func SplitPageRanges(pages []int) []PageRange {
	normalized, err := NormalizePages(pages)
	if err != nil || len(normalized) == 0 {
		return nil
	}
	ranges := make([]PageRange, 0)
	for start := 0; start < len(normalized); {
		end := start
		for end+1 < len(normalized) && normalized[end+1] == normalized[end]+1 {
			end++
		}
		pageRange := PageRange{Start: normalized[start], End: normalized[end], Label: strconv.Itoa(normalized[start])}
		if start != end {
			pageRange.Label = fmt.Sprintf("%d–%d", normalized[start], normalized[end])
		}
		ranges = append(ranges, pageRange)
		start = end + 1
	}
	return ranges
}

func PDFURL(file string, page int) string {
	path := "/pdf/" + url.PathEscape(file)
	if page > 0 {
		path += "#page=" + strconv.Itoa(page)
	}
	return path
}

func validPDFFileName(file string) bool {
	return strings.TrimSpace(file) != "" && !strings.ContainsAny(file, `/\\`) && file != "." && file != ".." && strings.EqualFold(filepath.Ext(file), ".pdf")
}

func FindTopic(catalog *Catalog, id string) (*Topic, bool) {
	for index := range catalog.Topics {
		if catalog.Topics[index].ID == id {
			return &catalog.Topics[index], true
		}
	}
	return nil, false
}

func FindQuestion(catalog *Catalog, id string) (*Question, bool) {
	for index := range catalog.Questions {
		if catalog.Questions[index].ID == id {
			return &catalog.Questions[index], true
		}
	}
	return nil, false
}

func CheckSourceAvailability(catalog *Catalog, projectRoot string) []string {
	warnings := make([]string, 0)
	availability := make(map[string]bool, len(catalog.PDFs))
	for index := range catalog.PDFs {
		pdf := &catalog.PDFs[index]
		path := filepath.Join(projectRoot, "content", "pdfs", pdf.File)
		info, err := os.Lstat(path)
		pdf.Available = err == nil && info.Mode().IsRegular()
		availability[pdf.File] = pdf.Available
		if !pdf.Available {
			warnings = append(warnings, fmt.Sprintf("PDF momentan nicht verfügbar: %s", pdf.File))
		}
	}
	for topicIndex := range catalog.Topics {
		for sourceIndex := range catalog.Topics[topicIndex].Sources {
			source := &catalog.Topics[topicIndex].Sources[sourceIndex]
			source.Available = availability[source.File]
		}
	}
	return warnings
}
