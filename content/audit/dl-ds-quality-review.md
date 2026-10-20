# DL/DS content review

As of: 2026-10-12. Based on the four registered DL course PDFs.
The illustrations and worked calculations were created for the learning app;
they are not unmodified screenshots or claimed original figures from the slides.

## Findings and changes

- 24 of 30 topics previously had no example text.
- Many explanations consisted of three short sentences.
- 16 of the 104 generated questions had identical answer options.
- Other options were factually true but were treated as wrong only because they
  belonged to a different topic. Topic choice is not a subject-matter refutation
  of a statement.
- All 30 topics now contain five explanatory paragraphs, a step sequence, a
  worked example, common confusions and two described local SVG illustrations.
  The illustrations are also visible in the Today learning step.
- All 104 generated questions were reworked. Eight of the older questions were
  made more precise: stride/padding/dilation, grouped convolution, effect of the
  learning rate, attention scaling and unmasked softmax prerequisites.
- The remaining older questions were read and their answers checked.
- All 34 numeric questions are tested with independent calculations.
- The content loader rejects answer texts and matching targets that are
  identical after normalization. Illustration data is structurally validated;
  executable markup is not part of the content format.

## Scope and persistence

| Area | Result |
|---|---:|
| DL/DS topics | 30 |
| Concept and example illustrations | 60 |
| Active DL/DS questions | 134 |
| Single choice | 29 |
| Multiple select | 28 |
| True/false | 30 |
| Ordering | 7 |
| Matching | 6 |
| Numeric choice | 34 |
| Duplicate active prompts | 0 |
| Identical answer options | 0 |
| Total app topics | 58 |
| Total active questions | 256 |
| Original PDFs | 13 |

The 28 ML topics and 122 ML questions were compared with the starting state and
are unchanged in content. Topic IDs, question IDs, priorities and enabled flags
are preserved. Exam pools remain ML 122/104 and DL/DS 134/110. Database schema 7
needs no new migration.

Historical reviews keep the score stored at the time. They are not re-scored
or deleted retroactively based on new answer options. Existing mastery may
therefore still reflect the earlier question quality.

## Subject-matter clarifications

The five learning ingredients also include the metric. CNN examples state the
dimension assumptions explicitly. BatchNorm explains the difference between
training and inference. LSTM has three named gates and offers no general
guarantee against vanishing gradients. Attention distinguishes scores, scaling,
masking, weights and values. With sinusoidal positional encoding, the frequency
decreases with the dimension pair index.

The source ranges for classic attention, positional information and transformer
components were aligned with the actual slides. Additional cross-checks:

- [Attention Is All You Need](https://arxiv.org/html/1706.03762v7)
- [Deep Residual Learning](https://arxiv.org/abs/1512.03385)
- [Stanford CS231n: Convolutions](https://cs231n.github.io/convolutional-networks/)
- [Original LSTM paper](https://www.bioinf.jku.at/publications/older/2604.pdf)

## Validation

Go tests, race detector, vet and the Node tests pass.
The Chrome test checks all 30 topics at 390 and 1440 px in light and dark:
120 page states with 240 rendered illustrations, without horizontal page
scrolling or labels extending beyond the SVG edge. Today learning also shows
both illustrations. Two mobile example graphics were additionally checked
visually. Other browsers and screen readers were not tested manually.

## Addendum 1.5.0: both exam areas, understanding and math

The existing 28 ML topics have been fully revised didactically, not just
lengthened with generic extra text. Each has its own explanations, step
sequences, a worked or subject-matter example, common confusions and two local
illustrations. The 30 already deepened DL topics additionally get prerequisites
and help with reading formulas. There are 116 illustrations in total. Formulas
explain the symbols they use; there is no new math library.

Particular attention to:

- dataset, case, feature, label, model, parameter and hyperparameter;
- training, validation, test, cross-validation and leakage during preprocessing;
- ReLU as the maximum of two numbers, not as an interval: negative inputs become
  zero, positive ones are unchanged; sigmoid/tanh and their open ranges;
- a full calculation through weight, bias, activation, loss, chain rule and
  gradient update; input/hidden/output roles and parameter counting;
- statistics, scaling, regression/least squares, threshold decisions, confusion
  matrix, precision/recall/F1, Gini, ensembles and cluster centers;
- CNN shapes, shared weights, batch/epoch arithmetic, states and gates,
  weighted attention sums and matrix dimensions.

All 122 active ML questions have new or deeper explanations. Redundant recall
questions and slide counts were turned into transfer and calculation tasks.
29 ML and 34 DL calculation questions have independently calculated answer
tests; for the new ML calculation tasks, wrong calculation paths are explained
as well. 27 DL questions were additionally made more precise, including the
unnecessary question about the historical embedding size, which now tests a
projection calculation. Not every question should involve calculation:
foundations, application and math still complement each other across all six
question types.

One important clarification: with fixed scores, recall does not decrease when
a threshold is lowered. Precision, by contrast, is not guaranteed to rise with
a higher threshold. The learning page includes a fully counted counterexample.
Basis: [official threshold documentation](https://scikit-learn.org/stable/modules/classification_threshold.html).
Fitting the scaler and imputer on training data only was checked against the
[official leakage documentation](https://scikit-learn.org/stable/common_pitfalls.html).
Supplementary sources: [metrics documentation](https://scikit-learn.org/stable/modules/model_evaluation.html),
[ReLU definition](https://docs.pytorch.org/docs/2.12/generated/torch.nn.ReLU.html),
[autograd explanation](https://docs.pytorch.org/tutorials/beginner/basics/autogradqs_tutorial.html)
and [random forest documentation](https://scikit-learn.org/stable/modules/ensemble.html).
These links are documentation sources, not external runtime dependencies.
New examples are purpose-made teaching examples, not verbatim exam questions;
the 13 course PDFs and their page references are preserved.

### Preserved data and limitations

Topic/question IDs, assignments, priorities, order, enabled status and sources
were verified as unchanged against the state before this change.
58 topics, 256 active questions, 13 PDFs; exam pools ML 122/104 and DL 134/110.
Auth, logout, theme behavior, scheduler, mastery, exam and PDF implementation
were not changed. Schema stays at 7; no migration. An additional backup test
confirms that content sync does not change personal data and that old
scores/answer data are still exported and imported.

Historical scores are not recalculated retroactively. For changed questions,
existing mastery may therefore still be based on the earlier question.
Historical views partly use current content texts; the earlier wording is
therefore not guaranteed as an immutable exam snapshot. This version deepens
the existing course scope but makes no claim to cover the entire research field
or unknown original exams.

### Validation 1.5.0

Go test suite, race detector, vet, build and Node tests passed. The numeric
answer tests do not just check stored IDs but test every answer option against
a separately calculated result. All 256 questions are also graded with four
shuffle seeds across their existing six question types.

Real Chrome test in a local read-only/non-root container: 58 topics at
320/375/390/1024/1440 px, each in light and dark, 580 page states in total with
1160 illustrations, without page overflow or text extending out of the SVG.
Today shows both figures. Mobile example images for thresholds and activations
were additionally checked visually. Login, session reload, desktop/mobile
logout, all theme modes and system switching passed the separate browser test.
Login and app show v1.5.0 in the footer.

During the first long stress test, Chromium ignored rapid renderer-initiated
hash navigations (IPC flood protection). The test now navigates under browser
control; no artificial app delay or weakening of auth was introduced.
Other browsers, real devices and screen readers were not tested manually.
Test volumes are separate from existing learning data. No push or deployment.

### Changed files

- all 58 JSON files in `content/topics/` and all six existing question files
  in `content/questions/`;
- central version `web/ui-state.js`, asset versions `web/index.html`;
- tests `internal/content/exam_preparation_test.go` (new),
  `internal/quiz/ml_content_test.go` (new), `internal/quiz/dl_content_test.go`,
  `internal/db/backup_test.go`, `web/embed_test.go`,
  `web/ui-state.test.js`, `web/learning-visuals.test.js`;
- browser checks `tools/browser-smoke.mjs`, `tools/dl-browser-smoke.mjs`;
- `README.md`, `DEPLOYMENT.md` and this review.

Final local image tag: `localhost/mldl-exam-trainer:1.5.0`.
