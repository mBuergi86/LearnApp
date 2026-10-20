package content

type Priority string

const (
	PriorityCore      Priority = "core"
	PrioritySupport   Priority = "support"
	PriorityReference Priority = "reference"
)

type Catalog struct {
	Exams         []Exam      `json:"exams"`
	Modules       []Module    `json:"modules"`
	Topics        []Topic     `json:"topics"`
	PDFs          []PDFSource `json:"pdfs"`
	Questions     []Question  `json:"-"`
	Comprehension []Question  `json:"-"`
}

type PDFSource struct {
	File      string `json:"file"`
	Title     string `json:"title"`
	ExamID    string `json:"exam_id"`
	Available bool   `json:"available"`
}

type QuestionType string

const (
	QuestionSingle    QuestionType = "single-choice"
	QuestionMultiple  QuestionType = "multiple-select"
	QuestionTrueFalse QuestionType = "true-false"
	QuestionOrder     QuestionType = "ordering"
	QuestionMatching  QuestionType = "matching"
	QuestionNumeric   QuestionType = "numeric-choice"
)

type Question struct {
	ID             string           `json:"id"`
	TopicID        string           `json:"topic_id"`
	Type           QuestionType     `json:"type"`
	Prompt         string           `json:"prompt"`
	Context        string           `json:"context,omitempty"`
	Explanation    string           `json:"explanation"`
	Difficulty     int              `json:"difficulty"`
	Enabled        bool             `json:"enabled"`
	Options        []QuestionOption `json:"options,omitempty"`
	CorrectAnswer  string           `json:"correct_answer,omitempty"`
	CorrectAnswers []string         `json:"correct_answers,omitempty"`
	CorrectOrder   []string         `json:"correct_order,omitempty"`
	MatchItems     []MatchItem      `json:"match_items,omitempty"`
	MatchOptions   []QuestionOption `json:"match_options,omitempty"`
}

type QuestionOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type MatchItem struct {
	ID              string `json:"id"`
	Text            string `json:"text"`
	CorrectOptionID string `json:"correct_option_id"`
}

type Exam struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type Module struct {
	ID          string `json:"id"`
	ExamID      string `json:"exam_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type Topic struct {
	ID            string         `json:"id"`
	ExamID        string         `json:"exam_id"`
	ModuleID      string         `json:"module_id"`
	Title         string         `json:"title"`
	Priority      Priority       `json:"priority"`
	Summary       string         `json:"summary"`
	Explanation   []string       `json:"explanation"`
	Steps         []string       `json:"steps"`
	Example       string         `json:"example"`
	Illustrations []Illustration `json:"illustrations,omitempty"`
	Formula       *Formula       `json:"formula,omitempty"`
	Confusions    []Confusion    `json:"confusions"`
	Sources       []Source       `json:"sources"`
	InitialStatus string         `json:"initial_status"`
	SortOrder     int            `json:"sort_order"`
}

// Illustration is declarative data, never executable markup from content.
type Illustration struct {
	Kind    string      `json:"kind"`
	Title   string      `json:"title"`
	Caption string      `json:"caption"`
	Labels  []string    `json:"labels"`
	Values  [][]float64 `json:"values,omitempty"`
}

type Formula struct {
	Expression string          `json:"expression"`
	Meaning    string          `json:"meaning"`
	Variables  []Variable      `json:"variables"`
	Latex      []string        `json:"latex,omitempty"`
	Example    *FormulaExample `json:"worked_example,omitempty"`
}

// FormulaExample keeps the pedagogical calculation separate from the general
// definition. LaTeX is data, never trusted executable HTML.
type FormulaExample struct {
	Title       string        `json:"title"`
	Given       []string      `json:"given"`
	Steps       []FormulaStep `json:"steps"`
	Result      string        `json:"result"`
	ResultLatex string        `json:"result_latex,omitempty"`
}

type FormulaStep struct {
	Text  string `json:"text"`
	Latex string `json:"latex,omitempty"`
}

type Variable struct {
	Symbol  string `json:"symbol"`
	Meaning string `json:"meaning"`
}

type Confusion struct {
	Term string `json:"term"`
	Note string `json:"note"`
}

type Source struct {
	File       string      `json:"file"`
	Title      string      `json:"title,omitempty"`
	Path       string      `json:"path,omitempty"`
	Pages      []int       `json:"pages"`
	PageLabel  string      `json:"page_label,omitempty"`
	PageRanges []PageRange `json:"page_ranges,omitempty"`
	URL        string      `json:"url,omitempty"`
	Available  bool        `json:"available"`
}

type PageRange struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}
