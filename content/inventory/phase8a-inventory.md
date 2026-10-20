# Phase 8A – slide inventory (local primary sources)

As of: 2026-09-06. Based exclusively on the text of the 13 PDFs in
`content/pdfs/`; no external sources were used. The ranges are contiguous and
cover every PDF page exactly once. `MERGE` means that several slides are
planned as one mental concept for a single learning page.

## Machine Learning (`machine-learning`)

| PDF | Slide range | Concept | Status | Planned topic/module |
|---|---:|---|---|---|
| MachineLearning_Tag1_EDA_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, agenda | DROP_FROM_LEARNING_VIEW | EDA |
|  | 4–12 | Understanding data and data sources | MERGE | EDA foundations |
|  | 13–24 | Data exploration, distributions, visualizations | MERGE | Exploratory data analysis |
|  | 25–42 | Summary statistics, correlation and outliers | MERGE | Statistical EDA |
|  | 43–55 | Graphical methods and EDA workflow | MERGE | EDA workflow |
|  | 56–57 | Exercise/wrap-up | REFERENCE | EDA |
| MachineLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, agenda | DROP_FROM_LEARNING_VIEW | ML foundations |
|  | 4–12 | Placing AI/ML/DL and their development | MERGE | ML foundations |
|  | 13–24 | ML definition, supervised/unsupervised learning | KEEP | Supervised vs. unsupervised |
|  | 25–39 | Reinforcement learning and MDP | MERGE | Reinforcement learning |
|  | 40–51 | Analytics continuum and ML lifecycle | MERGE | ML lifecycle |
|  | 52–69 | Features, data types and data matrix | MERGE | Features & data types |
|  | 70–89 | Industry sectors and trading applications | REFERENCE | ML applications |
|  | 90 | Exercises | REFERENCE | ML foundations |
| MachineLearning_Tag1_Preprocessing_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, lifecycle | DROP_FROM_LEARNING_VIEW | Preprocessing |
|  | 4–17 | Cleaning, missing values, noise | MERGE | Data cleaning |
|  | 18–28 | Transformations, scaling, power/log | MERGE | Data transformation |
|  | 29–39 | Ordinal, nominal, one-hot, multi-hot, periodic | KEEP | Feature encoding |
|  | 40–41 | Collinearity and its detection | KEEP | Collinearity |
| MachineLearning_Tag2_Lineare_Regression_2026_ShaoJuWoo.pdf | 1–2 | Title, preface | DROP_FROM_LEARNING_VIEW | Regression |
|  | 3–7 | Regression as supervised learning, model and EDA | MERGE | Linear regression |
|  | 8–15 | Matrix form, least squares, bias term, scikit-learn | MERGE | Linear regression model |
|  | 16–18 | Multivariate regression and forecasting | KEEP | Regression – application |
| Machine_Learning_Tag2_Logistic_Regression_2026_ShaoJuWoo.pdf | 1–2 | Title, preface | DROP_FROM_LEARNING_VIEW | Logistic regression |
|  | 3–6 | Classification and the logistic function | MERGE | Logistic regression |
|  | 7–11 | Decision boundary, odds/logit and training | MERGE | Logistic regression |
|  | 12–14 | Example and summary | EXAMPLE | Logistic regression |
| MachineLearning_Tag2_ModelTraining_Selection_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, lifecycle | DROP_FROM_LEARNING_VIEW | Training & model selection |
|  | 4–17 | Loss, gradient descent, learning rate, convergence | MERGE | Loss and optimization |
|  | 18–35 | Overfitting, underfitting, splits, early stopping | KEEP | Train/validation/test and overfitting |
|  | 36–47 | Confusion matrix, precision, recall, F1, regression metrics | MERGE | Performance metrics |
|  | 48–58 | Model selection, hyperparameters, bias/variance | KEEP | Model selection |
|  | 59–71 | Hold-out, cross-validation, GridSearchCV | MERGE | Cross-validation |
| Machine_Learning_Tag3_Decision_Trees_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, agenda | DROP_FROM_LEARNING_VIEW | Decision trees |
|  | 4–14 | Tree structure, splits, Gini/entropy | MERGE | Decision trees |
|  | 15–27 | Training, pruning and overfitting | KEEP | Decision trees – training |
|  | 28–34 | Examples and summary | EXAMPLE | Decision trees |
| Machine_Learning_Tag3_Ensemble_Learning_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, agenda | DROP_FROM_LEARNING_VIEW | Ensemble learning |
|  | 4–10 | Ensemble idea, voting and bagging | MERGE | Ensemble foundations |
|  | 11–17 | Random forest and feature importance | KEEP | Random forest |
|  | 18–21 | Boosting and examples | MERGE | Boosting |
| Machine_Learning_Tag3_Clustering_2026_ShaoJuWoo.pdf | 1–2 | Title, preface | DROP_FROM_LEARNING_VIEW | Clustering |
|  | 3–7 | K-means, centers, iteration and distance | KEEP | K-means |
|  | 8 | Summary/exercise | EXAMPLE | K-means |

## Deep Learning & Data Science (`deep-learning-data-science`)

| PDF | Slide range | Concept | Status | Planned topic/module |
|---|---:|---|---|---|
| DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf | 1–3 | Title, preface, agenda | DROP_FROM_LEARNING_VIEW | Deep learning foundations |
|  | 4–14 | Context, data, model parameters | MERGE | Deep learning foundations |
|  | 15–24 | Loss, entropy, cross-entropy, softmax | KEEP | Loss functions |
|  | 25–40 | Neuron, activation, network architecture, digit example | MERGE | Neural networks |
|  | 41–65 | Backpropagation, gradient descent, batch/SGD | KEEP | Training neural networks |
|  | 66–82 | Vanishing gradient and activation functions | MERGE | Activations and gradients |
|  | 83–95 | Early stopping, BatchNorm, regularization, dropout | KEEP | Regularization |
|  | 96–118 | Hardware and application examples | REFERENCE | DL applications |
|  | 119–133 | PyTorch and framework comparison | REFERENCE | Tooling (reference) |
| DeepLearning_Tag2_ComputerVision_2026.pdf | 1–3 | Title, preface, objectives | DROP_FROM_LEARNING_VIEW | Computer vision/CNN |
|  | 4–15 | CNN idea, image as numbers, RGB, fully connected | MERGE | CNN foundations |
|  | 16–24 | Convolution, filters, feature maps | KEEP | Convolution |
|  | 25–28 | CNN/ConvNet and VGG16 | MERGE | CNN architectures |
|  | 29–43 | Spatial dimensions, padding, parameters | KEEP | CNN dimensions |
|  | 44–47 | Pooling, ResNet | MERGE | Pooling and residual networks |
|  | 48 | Homework | REFERENCE | Computer vision |
| DeepLearning_Tag3_2026_Sequence-to-Sequence Modelle.pdf | 1–2 | Title, contents | DROP_FROM_LEARNING_VIEW | Sequence models |
|  | 3–7 | Sequence modeling and applications | MERGE | Sequence models |
|  | 8–15 | RNN and Seq2Seq architecture | KEEP | RNN/Seq2Seq |
|  | 16–23 | Character model, multi-layer encoder-decoder | EXAMPLE | RNN/Seq2Seq |
|  | 24–25 | BPTT and vanishing gradient | KEEP | RNN training |
|  | 26 | Contents slide | DROP_FROM_LEARNING_VIEW | Sequence models |
|  | 27–35 | LSTM cell state and gates | KEEP | LSTM |
|  | 36–37 | Exercises/text representation | REFERENCE | Sequence models |
|  | 38 | References | DROP_FROM_LEARNING_VIEW | Sequence models |
| DeepLearning_Tag3_Transformer_2026.pdf | 1–2 | Title, contents | DROP_FROM_LEARNING_VIEW | Transformer |
|  | 3–9 | NLP and text representation | MERGE | NLP/text representation |
|  | 10–17 | Word embeddings and sequence representation | KEEP | Embeddings |
|  | 18–25 | Attention, query/key/value | KEEP | Attention |
|  | 26–38 | Self-attention and transformer block | KEEP | Self-attention/transformer |
|  | 39–46 | Multi-head attention, position, encoder/decoder | MERGE | Transformer architecture |
|  | 47–48 | Applications/references | REFERENCE | Transformer |

## Coverage and production notes

- PDF files: **13/13 present**.
- Counted with `pdfinfo`: **621 PDF pages** (the 13 extracted trailing form feeds are not pages).
- Every page is assigned to exactly one range; title/agenda/references are
  deliberately marked `DROP_FROM_LEARNING_VIEW`, examples `EXAMPLE`, and
  tool/application slides without their own learning concept `REFERENCE`.
- This assignment is the verified 8A working basis for 8B. Page numbers in the
  later topic sources are taken exclusively from these ranges; no pages are
  invented.
- 8B (learning pages) and 8C (questions) have **not yet started** at this step.
