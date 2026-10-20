# ML/DL Exam Trainer – V1 Specification

As of: 3 September 2026

This is the original V1 specification. Later releases changed some decisions (for example, login, HTTPS behind a proxy and container deployment were added); see [README.md](README.md) for the current state. The app's user interface is German; the UI mockups below are shown in English for readability.

## 1. Goal

A small local learning web app for exam preparation alongside a full-time job, built from the existing course materials.

The app should turn many dry lecture slides into a compact, clearly structured learning trainer. It should work in the browser on desktop and phone, without npm, Node.js, a cloud account or AI grading.

Current course basis:

- 13 PDF files
- 621 original slides
- two separate exam areas:
  - **Machine Learning**
  - **Deep Learning & Data Science**

The original slides remain the authoritative source. The learning app condenses and structures them but does not replace them.

---

## 2. Main principle

Do not learn by PDF file or slide number.

Instead:

```text
621 original slides
        ↓
identify topics
        ↓
KEEP / MERGE / EXAMPLE / REFERENCE
        ↓
compact learning pages
        ↓
objectively gradable exercises
        ↓
spaced repetition
        ↓
exam simulation
```

A learning page should explain one mental concept, not reproduce an original slide.

---

# 3. Technical stack

## Backend

- Go
- a single binary
- HTTP server
- SQLite
- single user
- no login
- no cloud dependency
- no external API required

## Frontend

- HTML5
- CSS
- vanilla JavaScript
- no npm
- no Node.js
- no React
- no Vue
- no Svelte
- no frontend build pipeline

## Operation

Server: a host on the local network, port `8080`.

URL:

```text
http://<server-ip>:8080
```

Go listener:

```text
0.0.0.0:8080
```

Explicitly for V1:

- no HTTPS
- no certificate management
- no PWA
- no service worker
- no offline mode

Reason: the app should run as simply and reliably as possible before the exam. HTTPS and PWA can be added later in V2.

---

# 4. Deployment

Preferred:

```text
./mldl-exam-trainer
```

The static frontend files should be embedded with Go `embed` where possible.

Persistent data lives outside the binary:

```text
./data/exam.db
./content/
```

Recommended project structure:

```text
mldl-exam-trainer/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── db/
│   ├── scheduler/
│   ├── quiz/
│   ├── content/
│   └── http/
├── web/
│   ├── index.html
│   ├── app.js
│   └── styles.css
├── content/
│   ├── topics/
│   └── pdfs/
├── data/
│   └── exam.db
├── go.mod
└── README.md
```

Codex may simplify the exact Go package structure as it sees fit, as long as the app stays small and understandable.

---

# 5. Keep exam areas strictly separate

Technically the app is one project, but didactically it is two separate exam trainers.

## A. Machine Learning

Its own:

- start card
- modules
- learning pages
- questions
- reviews
- progress display
- weak spots
- exam simulation

## B. Deep Learning & Data Science

Also its own:

- start card
- modules
- learning pages
- questions
- reviews
- progress display
- weak spots
- exam simulation

Progress must never be combined.

---

# 6. Start page

The start page should be very calm.

Mobile and desktop show two large exam areas:

```text
ML/DL EXAM TRAINER

MACHINE LEARNING
██████████░░░░  68 %
7 reviews
3 uncertain core topics

[ Study ML today ]


DEEP LEARNING & DATA SCIENCE
██████░░░░░░░░  42 %
11 reviews
6 uncertain core topics

[ Study DL & DS today ]
```

Optionally below:

```text
38 days until the exam
```

If separate exam dates are configured later, each exam area should get its own countdown.

Not on the start page:

- pie charts
- leaderboards
- badges
- XP
- streak flames
- confetti
- social features
- large statistics dashboards

---

# 7. Main navigation

## Mobile

Four sections:

```text
⌂ Today
▦ Topics
? Practice
✓ Progress
```

Bottom navigation, always easy to reach.

## Desktop

Left sidebar:

```text
Today
Machine Learning
Deep Learning & Data Science
Practice
Progress
```

No hamburger menu on desktop.

---

# 8. UX principles

The interface should use as little working memory as possible.

Key rules:

1. one main task per screen
2. little visual distraction
3. clear hierarchy
4. compact text
5. reveal content step by step
6. graphics and explanation directly together
7. no decorative images without a learning purpose
8. no unnecessary animations
9. large touch targets
10. keyboard operation on desktop
11. color is never the only signal of meaning
12. do not show the original slide by default

---

# 9. Light and dark mode

## Default

**Light mode**

Options:

```text
● Light
○ Dark
○ System
```

The choice is stored server-side where possible.

---

# 10. Color palette

## Light mode

```css
--bg:              #F6F8FB;
--surface:         #FFFFFF;
--surface-soft:    #F1F5F9;

--text:            #172033;
--text-secondary:  #5F6B7A;

--primary:         #087EA4;
--primary-soft:    #E8F6FB;

--success:         #16803C;
--warning:         #B85C00;
--danger:          #C62828;

--border-subtle:   #D8E0EA;
--border-control:  #7C8A9A;
```

## Dark mode

```css
--bg:              #0B1220;
--surface:         #111827;
--surface-soft:    #182235;

--text:            #F1F5F9;
--text-secondary:  #AAB6C5;

--primary:         #38BDF8;
--primary-soft:    #0B3043;

--success:         #4ADE80;
--warning:         #FBBF24;
--danger:          #FB7185;

--border-subtle:   #2B3A50;
--border-control:  #607089;
```

Meaning:

```text
Blue / cyan → action and key information
Green       → correct and confident
Orange      → uncertain and caution
Red         → wrong and error
Gray        → neutral
```

No red for normal core material.

Core material is shown, for example, like this:

```text
◆ CORE
```

---

# 11. Typography

System font, no web fonts needed.

```css
font-family:
    system-ui,
    -apple-system,
    "Segoe UI",
    sans-serif;
```

Recommended sizes on mobile:

```text
Page title           26–30 px
Topic heading        22–24 px
Body text            17–19 px
Metadata             15–16 px
Line height          about 1.5
```

---

# 12. Cards, surfaces and borders

Do not put every piece of information in a card.

Use cards only for real meaning, e.g.:

- key idea
- warning
- answer feedback
- quiz options

Example:

```text
┌──────────────────────────────────┐
│ 🧠 KEY IDEA                      │
│                                  │
│ Backpropagation computes the     │
│ gradients of the error.          │
└──────────────────────────────────┘
```

Recommended:

```css
border: 1px solid var(--border-subtle);
border-radius: 12px;
```

Interactive controls must use a clearer border:

```css
border-color: var(--border-control);
```

Shadows only very subtle or none at all.

---

# 13. Accessibility

Must be considered:

- normal text at least WCAG AA contrast
- UI controls clearly recognizable
- meaning never conveyed by color alone
- visible keyboard focus
- large touch targets
- respect `prefers-reduced-motion`

Focus example:

```css
:focus-visible {
    outline: 3px solid var(--primary);
    outline-offset: 3px;
}
```

Answer cards on mobile should be at least about 44–48 px tall.

---

# 14. Animations

No distraction from learning.

Allowed:

- very short UI transitions
- about 100–150 ms

Not wanted:

- confetti
- flying cards
- pulsing buttons
- animated statistics during a question
- unnecessary micro-animations

---

# 15. Learning page

Each learning page covers one concept.

Example:

```text
← Neural Networks                  7 / 11

◆ CORE

BACKPROPAGATION

┌──────────────────────────────────┐
│ KEY IDEA                         │
│                                  │
│ Backpropagation computes the     │
│ gradients of the error with      │
│ respect to weights and biases.   │
└──────────────────────────────────┘

HOW IT WORKS

Forward → Loss
             ↓
       Backpropagation
             ↓
          Gradient

EXAMPLE

short concrete explanation

⚠ DO NOT CONFUSE

Backpropagation
→ computes gradients

Gradient Descent
→ uses gradients

❓ CAN YOU DO IT?

[ Test now ]

📎 Original slides 42–44
```

---

# 16. Progressive disclosure

Do not show everything at once.

Order:

1. key idea
2. how it works
3. example
4. important formula, if needed
5. do not confuse
6. quiz
7. original source

The key idea should be at most 1–2 sentences.

---

# 17. Content schema per learning page

Recommended data structure:

```json
{
  "id": "backpropagation",
  "exam": "deep-learning-data-science",
  "module": "training",
  "title": "Backpropagation",
  "priority": "core",
  "summary": "Short key idea.",
  "explanation": [
    "Point 1",
    "Point 2"
  ],
  "steps": [],
  "example": "",
  "formula": "",
  "confusions": [
    {
      "topic_id": "gradient-descent",
      "note": "Backpropagation computes gradients, gradient descent uses them."
    }
  ],
  "sources": [
    {
      "file": "DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo(1).pdf",
      "pages": [42, 43, 44]
    }
  ]
}
```

JSON or Markdown with front matter are both acceptable.

Important:

- do not hard-code learning content in HTML
- store content separately
- always make the original source referenceable

---

# 18. Priorities

Three content levels:

```text
CORE       = must be actively understood and tested
SUPPORT    = understand and place in context
REFERENCE  = look up only
```

Display:

```text
◆ CORE
● SUPPORT
○ REFERENCE
```

Learner status:

```text
NOT LEARNED
VIEWED
IN PRACTICE
CONFIDENT
UNCERTAIN
```

Do not mix this with content priority.

---

# 19. Progress

Progress must not mean:

```text
80 % opened = 80 % learned
```

Progress is based on mastery.

Example internal scoring:

```text
NOT LEARNED             0
VIEWED                  1
IN PRACTICE             2
CORRECT SEVERAL TIMES   3
CONFIDENT               4
```

Weight core material more heavily:

```text
CORE       weight 2
SUPPORT    weight 1
REFERENCE  weight 0
```

Display:

```text
Machine Learning 67 %

32 topics confident
11 in practice
7 uncertain
```

---

# 20. Today view

After a working day, the app should decide what makes sense.

Example:

```text
TODAY

How much time do you have?

[ 10 min ] [ 20 min ] [ 30 min ] [ 45 min ]
```

## 10 minutes

- only due core questions
- no new topics

## 20 minutes

- reviews
- one weak spot

## 30 minutes

- reviews
- one new concept
- short mixed practice

## 45 minutes

- reviews
- two new concepts
- mixed questions

---

# 21. Exercise types in V1

Free text is explicitly not part of V1.

No AI grading.

Allowed objectively gradable types:

1. single choice
2. multiple select
3. true / false
4. ordering
5. matching
6. numeric multiple choice

---

# 22. Multiple-choice rules

Default:

- usually 4 answer options
- shuffle answer positions randomly
- plausible distractors
- no deliberate trick questions
- no phrasing like "all of the above"
- avoid "none of the answers" where possible
- clearly label multiple select as such
- always show a short explanation after answering
- also explain important wrong answers

Example:

```text
What is the task of backpropagation?

○ Initialize weights randomly
○ Compute gradients
○ Normalize data
○ Determine clusters

[ Check answer ]
```

---

# 23. Answer feedback in learning mode

After a correct answer:

```text
✓ Correct

Backpropagation computes the
gradients with respect to weights and biases.

Gradient descent then uses these
gradients to adjust the
parameters.

[ Next ]
```

After a wrong answer:

```text
✕ Wrong

You chose:
Gradient Descent

Correct:
Backpropagation

WHY?

Backpropagation
→ computes gradients

Gradient Descent
→ uses gradients

[ Review concept briefly ]
[ Next ]
```

---

# 24. True/false

Example:

```text
One epoch means that all
training examples have been processed once.

[ False ] [ True ]
```

---

# 25. Ordering

Example:

```text
Put the training steps in the correct order:

[ Loss ]
[ Forward pass ]
[ Update weights ]
[ Backpropagation ]
```

Correct:

```text
Forward pass
↓
Loss
↓
Backpropagation
↓
Update weights
```

On mobile, prefer arrows over drag and drop:

```text
↑  ↓
```

---

# 26. Matching

Example:

```text
Parameter
Hyperparameter
```

match to:

```text
learned during training
set before training
```

---

# 27. Numeric multiple-choice questions

Example:

```text
Training examples: 1'000
Batch size: 250

How many iterations does one epoch have?

○ 2
○ 4
○ 250
○ 1'000
```

---

# 28. Question data model

A topic has several questions.

Example:

```text
Backpropagation
├── What does backpropagation compute?
├── How does it differ from gradient descent?
├── In which direction does the computation run?
├── What role does the chain rule play?
└── Which quantities are differentiated?
```

Target for core concepts:

```text
about 3–6 questions per CORE topic
```

Long term, about:

```text
250–350 questions
```

V1 may start with fewer, as long as the data structure allows extension.

---

# 29. Question variants

The same knowledge should be tested in different ways.

Example:

```text
What is the purpose of the validation set?

Which dataset do you use to choose hyperparameters?

Which dataset may only be used for the final independent evaluation?
```

This way the concept is learned rather than the wording.

---

# 30. Spaced repetition

A simple scheduler oriented to the exam date is enough.

Example for correct answers:

```text
1. correct → in 2 days
2. correct → in 5 days
3. correct → in 10 days
4. correct → in 18 days
```

For a wrong answer:

```text
wrong
↓
again after a few other questions
↓
the next day
↓
after 3 days
```

Shortly before the exam, uncertain core questions should appear more often.

The concrete implementation may be cleanly abstracted so that the intervals can be made configurable later.

---

# 31. Separate learning mode and exam simulation

## Learning mode

```text
question
↓
answer
↓
immediate feedback
↓
explanation
↓
next question
```

## Exam simulation

```text
question 1
question 2
...
question N
↓
no solutions in between
↓
evaluation at the end
```

---

# 32. Exam simulation

Configuration:

```text
EXAM SIMULATION

Machine Learning

Questions:
○ 10
○ 20
● 30

Content:
● all
○ core only
○ weak spots only

[ Start exam ]
```

Evaluation:

```text
23 / 30
77 %

STRONG
✓ Regression
✓ K-Means
✓ EDA

WEAK
⚠ Model Selection
⚠ Decision Trees
⚠ Bias / Variance

[ Practice mistakes again ]
```

---

# 33. Original slides

Original material is always available but collapsed by default.

Example:

```text
📎 ORIGINAL MATERIAL

Deep Learning Day 1
Slides 42–44

[ Open original slides ]
```

Desktop may optionally offer a split view.

Mobile shows the original slide below or after the learning content.

Never show the original slide by default during a quiz question.

---

# 34. PDF display

V1 may use the browser's built-in PDF viewer if that is simplest.

Alternatively, a single original page can be opened as a PDF link.

No complex PDF.js integration is needed if it makes V1 unnecessarily larger.

---

# 35. Backup

SQLite holds the learning progress.

Settings:

```text
DATA

[ Download backup ]
[ Import backup ]
```

At least JSON export/import.

Optionally also download of the SQLite file.

No cloud backup in V1.

---

# 36. Database – recommended tables

Codex may improve the schema, but functionally roughly the following entities are needed:

```text
exams
modules
topics
topic_sources
questions
question_options
reviews
topic_progress
settings
study_sessions
```

## exams

Example:

```text
machine-learning
deep-learning-data-science
```

Fields at minimum:

```text
id
name
exam_date optional
sort_order
```

## modules

```text
id
exam_id
name
sort_order
```

## topics

```text
id
exam_id
module_id
title
priority
summary
content
status
sort_order
```

## questions

```text
id
topic_id
type
prompt
explanation
difficulty
enabled
```

## question_options

```text
id
question_id
text
is_correct
sort_order
explanation optional
```

## reviews

```text
id
question_id
answered_at
correct
response_data
```

## topic_progress

```text
topic_id
mastery_level
last_reviewed_at
next_review_at
correct_count
wrong_count
```

## settings

At minimum:

```text
theme
default_session_minutes
```

---

# 37. API – minimal proposal

Codex may also use server-rendered endpoints internally.

If a JSON API is used:

```text
GET  /api/exams
GET  /api/exams/:id
GET  /api/exams/:id/modules
GET  /api/topics/:id

GET  /api/today?exam=...&minutes=30

GET  /api/questions
POST /api/answers

GET  /api/progress
GET  /api/progress/:exam

POST /api/settings

GET  /api/backup
POST /api/restore
```

No unnecessary API complexity.

---

# 38. Machine Learning – modules

## Module 1: ML foundations

About 5 learning pages:

1. AI → Machine Learning → Deep Learning
2. What is machine learning?
3. Supervised vs. unsupervised learning
4. Features, target and data types
5. ML development lifecycle

## Module 2: EDA

About 5 learning pages:

6. What is EDA and why before training?
7. Understanding data quality
8. Univariate analysis
9. Multivariate analysis
10. Detecting missing values and outliers

## Module 3: Preprocessing

About 5 learning pages:

11. Why preprocessing?
12. Data cleaning
13. Data transformation
14. Encoding
15. Data reduction and feature selection

## Module 4: Regression

About 4 learning pages:

16. Regression vs. classification
17. Linear regression
18. Multiple linear regression
19. Parameters and prediction

## Module 5: Logistic regression

About 2 learning pages:

20. Logistic regression
21. Linear vs. logistic regression

## Module 6: Training and model selection

About 9 learning pages:

22. Loss function
23. Training a model
24. Generalization
25. Underfitting and bias
26. Overfitting and variance
27. Reducing overfitting
28. Parameters vs. hyperparameters
29. Train, validation and test
30. Cross-validation

## Module 7: Decision trees

About 4 learning pages:

31. Decision tree basic idea
32. Root, node, split and leaf
33. Impurity and Gini
34. Pruning

## Module 8: Ensemble learning

About 3 learning pages:

35. Why multiple models?
36. Bagging
37. Random forest

## Module 9: Clustering

About 2 learning pages:

38. Supervised learning vs. clustering
39. K-means

Machine Learning therefore comes to about:

```text
39 learning pages
```

---

# 39. Deep Learning & Data Science – modules

Important:

The files available so far mainly cover deep learning.

Do not invent additional data science topics if they do not appear in the official exam materials.

## Module 1: Deep learning foundations

About 3 learning pages:

40. Placing deep learning in context
41. Ingredients of an ML/DL algorithm
42. Dataset, model, loss, optimization and metric

## Module 2: Neural networks

About 4 learning pages:

43. Understanding a neural network
44. Neuron, weight and bias
45. Activation functions
46. Input, hidden and output layers

## Module 3: Training neural networks

About 5 learning pages:

47. Forward pass
48. Loss and gradient
49. Gradient descent
50. Backpropagation
51. Backpropagation vs. gradient descent
52. Batch, iteration and epoch
53. The full training cycle

## Module 4: Computer vision and CNN

About 7 learning pages:

54. Why CNNs?
55. Convolution
56. Filter and kernel
57. Feature map
58. Stride
59. Padding
60. Pooling

## Module 5: Sequence models

About 5 learning pages:

61. Why sequence models?
62. RNN basic idea
63. Backpropagation through time
64. The problem of long-range dependencies
65. LSTM basic idea

## Module 6: Seq2Seq

About 2 learning pages:

66. Seq2Seq basic idea
67. Encoder, context vector and decoder

## Module 7: Attention

About 2 learning pages:

68. Why attention?
69. Attention weights

## Module 8: Transformer

About 5 learning pages:

70. Self-attention
71. Query, key and value
72. Multi-head attention
73. Transformer architecture
74. Residual connections and layer normalization

So currently about:

```text
35 learning pages
```

Overall target:

```text
about 70–80 compact learning pages
```

The exact number is not sacred. A complex concept may need several learning pages; a simple concept may be merged with another.

---

# 40. Source PDFs

The app should know the following original files as course sources.

## Machine Learning

```text
MachineLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf
MachineLearning_Tag1_EDA_2026_ShaoJuWoo.pdf
MachineLearning_Tag1_Preprocessing_2026_ShaoJuWoo.pdf
MachineLearning_Tag2_Lineare_Regression_2026_ShaoJuWoo.pdf
Machine_Learning_Tag2_Logistic_Regression_2026_ShaoJuWoo.pdf
MachineLearning_Tag2_ModelTraining_Selection_2026_ShaoJuWoo.pdf
Machine_Learning_Tag3_Decision_Trees_2026_ShaoJuWoo.pdf
Machine_Learning_Tag3_Ensemble_Learning_2026_ShaoJuWoo.pdf
Machine_Learning_Tag3_Clustering_2026_ShaoJuWoo.pdf
```

## Deep Learning & Data Science

```text
DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo(1).pdf
DeepLearning_Tag2_ComputerVision_2026.pdf
DeepLearning_Tag3_2026_Sequence-to-Sequence Modelle.pdf
DeepLearning_Tag3_Transformer_2026.pdf
```

---

# 41. Transformation rules for original slides

During content creation, each original slide is mentally assigned to a category:

## KEEP

Contains a standalone, testable concept.

## MERGE

Explains a concept that belongs on one learning page together with several other slides.

## EXAMPLE

Is a good example or diagram for a learning page.

## REFERENCE

Stays linked as an original source but gets no learning page of its own.

## DROP FROM LEARNING VIEW

Is not presented as learning content.

Typical examples:

- title slides
- agenda
- preface
- pure source/citation slides
- repeated lifecycle overviews
- long historical digressions with no exam value
- installation steps
- redundant screenshots
- multiple identical definitions

Important:

`DROP FROM LEARNING VIEW` never means the original slide is deleted.

---

# 42. Content rules

Learning pages must stick to the official course materials.

Do not silently:

- invent additional theory
- correct statements from the course
- mix in external content
- pretend to know exam weighting

If content from external research is added, it must be clearly marked as a supplement.

The original source should always be traceable.

---

# 43. Formulas

Only show formulas that are relevant for understanding or for the exam.

A formula always needs:

- meaning
- variables
- short explanation
- a mini example where possible

No formula collection without context.

---

# 44. "Do not confuse"

Especially important comparison pairs:

```text
AI vs. ML vs. DL
Supervised vs. unsupervised
Regression vs. classification
Linear vs. logistic regression
Parameters vs. hyperparameters
Training vs. validation vs. test
Underfitting vs. overfitting
Bias vs. variance
Decision tree vs. random forest
Bagging vs. a single model
Batch vs. iteration vs. epoch
Backpropagation vs. gradient descent
RNN vs. LSTM
Attention vs. self-attention
Query vs. key vs. value
```

These comparisons should appear both as learning content and as quiz questions.

---

# 45. Non-goals for V1

Do not build:

- npm
- Node.js
- React
- Vue
- Svelte
- cloud
- login
- user management
- OAuth
- social features
- leaderboard
- points and XP
- badges
- AI chat
- AI answer grading
- free-text exams
- HTTPS
- PWA
- service worker
- offline sync
- push notifications
- complex PDF.js application
- microservices
- Kubernetes
- external database

---

# 46. V1 acceptance criteria

V1 is considered successful when:

1. `go build` produces a runnable application
2. the app starts on `0.0.0.0:8080`
3. access via `http://<server-ip>:8080` on the local network works
4. desktop and phone are cleanly responsive
5. light and dark mode work
6. the two exam areas are completely separate
7. modules and topics can be loaded from content files
8. a learning page can show key idea, explanation, example and source
9. single choice works
10. multiple select works
11. true/false works
12. ordering works
13. matching works
14. numeric MC questions work
15. answer order is shuffled randomly
16. learning mode shows feedback immediately
17. exam simulation shows feedback only at the end
18. progress is stored in SQLite
19. due reviews are calculated automatically
20. "Today" builds sessions for 10, 20, 30 or 45 minutes
21. ML progress and DL/DS progress stay separate
22. backup export works
23. backup import works
24. original PDFs are reachable from topics
25. the application needs no external internet connection

---

# 47. Implementation order for Codex

## Phase 1 – Skeleton

- Go app
- SQLite
- static files
- routing
- light/dark theme
- responsive layout
- start page

## Phase 2 – Exam structure

- exams
- modules
- topics
- content loader
- learning pages

## Phase 3 – Quiz

- single choice
- multiple select
- true/false
- ordering
- matching
- numeric MC
- feedback

## Phase 4 – Learning progress

- reviews
- mastery
- scheduler
- Today view
- weak spots

## Phase 5 – Exam simulation

- selection by exam
- number of questions
- all / core / weak spots
- final evaluation

## Phase 6 – Original sources

- PDF links
- page references
- simple original view

## Phase 7 – Backup and polish

- JSON backup
- restore
- accessibility
- keyboard
- mobile UX
- tests

---

# 48. Tests

Test at least:

## Go

- scheduler
- progress calculation
- quiz grading
- backup/restore
- content parsing

## Browser

- light mode
- dark mode
- mobile width
- desktop width
- keyboard navigation
- correct/wrong quiz answer
- random answer order
- exam simulation without intermediate feedback

No heavy E2E test platform needed if it would require npm.

Go tests plus small browser-side tests without a build tool are enough for V1.

---

# 49. How Codex should work

Codex should:

1. read this specification completely first
2. not add new frameworks without a compelling reason
3. deliberately keep V1 small
4. not introduce any npm dependency
5. not implement any features from the non-goals
6. work in small, testable steps
7. build and test after each phase
8. maintain the README with start instructions
9. not invent learning content that does not come from the course material
10. cleanly separate content and application code

---

# 50. Definition of done

V1 is done when, after work, the user can open on their phone:

```text
http://<server-ip>:8080
```

then choose:

```text
Machine Learning
or
Deep Learning & Data Science
```

start a short learning session, learn compact concepts, answer multiple-choice questions, get meaningful feedback immediately, see their learning progress saved, and later have exactly the uncertain content repeated automatically.

The app should draw as little attention to itself as possible while learning.

The focus is on the material, not on the software.
