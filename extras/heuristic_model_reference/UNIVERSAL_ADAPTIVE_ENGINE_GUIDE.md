# Universal Adaptive Engine Guide

## Introduction
This guide is designed for AI agents tasked with constructing an adaptive practice platform for any domain (e.g., medical training, corporate compliance, language learning, or technical skills). The provided codebase (in `offline_pipeline`, `runtime_engine`, and `reference_docs`) implements a generalized Adaptive Learning Engine.

## Universal Terminology
To adapt this codebase to a new domain, mentally map the following terms:
- **Questions / Items:** The discrete units of assessment or practice.
- **Learners / Users:** The individuals interacting with the platform.
- **Knowledge Graph / Syllabus:** A hierarchical tree representing the domain.
- **Domains / Topics / Subtopics:** Nodes within the Knowledge Graph.
- **Theta ($\theta$):** A learner's latent ability level in a specific topic.
- **Difficulty ($b$):** The inherent difficulty of a question/item.

## 1. Offline Pipeline (`offline_pipeline/`)
The offline pipeline prepares raw item data for the runtime engine. It involves data synthesis, embedding generation, and database seeding.

### 1.1 Rating Synthesis (Glicko-2 Calibration)
- **What it does:** Calibrates the initial difficulty ($b$) of items. If you have historical interaction data (correct/incorrect answers by learners), the pipeline uses the Glicko-2 rating system to determine the relative difficulty of items.
- **How to use it:** Feed historical attempt data into the synthesis script. The output is a stabilized difficulty score for each item, which is crucial for the IRT model during runtime.

### 1.2 Embedding Generation & Text Search
- **What it does:** Converts item text, explanations, and metadata into dense vector embeddings (e.g., using a text embedding model). 
- **How to use it:** These embeddings are stored in a vector database (like pgvector) and allow the runtime engine to find semantically similar items (e.g., "Find an item similar to this one, but slightly easier"). Text search capabilities (like FTS or Elasticsearch) are also prepared here for keyword-based retrieval.

### 1.3 Database Seeding
- **What it does:** Populates the relational and vector databases with the processed items, initial difficulties, embeddings, and the Knowledge Graph structure.
- **How to use it:** Run the seed scripts to initialize the application database before starting the runtime engine.

## 2. Runtime Engine (`runtime_engine/`)
The runtime engine is the core backend service that manages active practice sessions, serves items, evaluates responses, and updates learner ability profiles.

### 2.1 The Knowledge Graph (SyllabusGraph)
- **What it does:** An in-memory, domain-agnostic hierarchical tree. It defines the relationships between topics and subtopics.
- **How to use it:** The engine uses this graph for scope resolution (e.g., "The learner is practicing Topic A, which includes Subtopics X, Y, and Z"). Ensure your domain's taxonomy is loaded into this graph format.

### 2.2 Session State Machine
- **What it does:** Manages the lifecycle of a practice session (Start -> Submit Item -> Pick Next Item -> ... -> Close).
- **How to use it:** The state machine ensures transactional integrity and context retention during a session.

### 2.3 IRT (Theta) Ladder
- **What it does:** Estimates the learner's ability ($\theta$) dynamically during a session. It relies on a 1-Parameter Logistic (1PL) Rasch Model: $P(Correct) = \frac{1}{1 + e^{-(\theta - b)}}$.
- **How to use it:** After every submitted answer, the engine updates the learner's temporary session $\theta$ using Maximum Likelihood Estimation (MLE) or a Bayesian update. This $\theta$ is used to target the difficulty of the next item.

### 2.4 The 6-Factor Scoring Heuristic
- **What it does:** A robust algorithm to select the optimal next item for the learner. It evaluates candidate items based on:
  1. **Target Difficulty:** Proximity of item difficulty ($b$) to the learner's current $\theta$.
  2. **Exposure Penalty:** Penalizes items the learner has seen recently.
  3. **Content Diversity:** Ensures the learner sees items from various subtopics within the active scope.
  4. **Format Variety:** Avoids showing too many items of the same type (e.g., Multiple Choice vs. Fill-in-the-blank) consecutively.
  5. **Review Priority:** Occasionally injects older items the learner struggled with (Spaced Repetition).
  6. **Vector Similarity:** If the learner fails an item, the engine may fetch a semantically similar item (using embeddings) to reinforce the specific concept.
- **How to use it:** Tune the weights of these 6 factors based on the pedagogical goals of your new domain.

### 2.5 Glicko-2 Rating Updates
- **What it does:** Unlike the rapid, volatile $\theta$ updates during a session, the engine uses Glicko-2 to permanently update the learner's global rating and the items' global difficulties at the *end* of a session.
- **How to use it:** When a session closes, the engine processes all interactions as a "match" between the learner and the items, updating the Glicko-2 ratings (Rating, Rating Deviation, Volatility) for both in the database.

## 3. Reference Docs (`reference_docs/`)
- **What they contain:** Detailed architectural decisions (ADRs), system design diagrams, and data schemas.
- **Universal Applicability:** The patterns documented here—such as CQRS for read/write separation, the choice of vector database, and the scaling strategy for the in-memory graph—apply regardless of the content domain. Consult these docs when modifying the core architecture or scaling the system.

## Summary for AI Implementation
1. Parse your new domain's taxonomy into the universal Knowledge Graph format.
2. Run your raw items through the `offline_pipeline` to generate embeddings and initial difficulty ratings.
3. Deploy the `runtime_engine` to serve sessions, tuning the 6-factor heuristic weights as needed.
4. Let the IRT and Glicko-2 systems automatically calibrate learners and items over time.
