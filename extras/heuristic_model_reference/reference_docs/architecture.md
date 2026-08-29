# AlphaJEE: Complete System Architecture & Technical Roadmap
**Adaptive Learning Ecosystem for JEE Advanced & SAT Preparation**
*2PL IRT · Glicko-2 · Reinforcement Learning (PPO) · Cloudflare Edge*

**Version 3.0 | Confidential | Internal Use Only**

---

## Table of Contents
1. **Project Overview & Philosophy**
2. **Mathematical Core — IRT, Glicko-2, Fisher Information**
3. **Practice Mode — The AI Coach**
4. **RL Agent — Deep Technical Breakdown**
5. **RL Agent — Build Roadmap Step by Step**
6. **Mock / Adaptive Mode — The Ruler**
7. **Challenge / Lobby Mode — The Arena**
8. **Trust Score System**
9. **Monetization & Free Tier Strategy**
10. **Anti-Scraping & Security**
11. **Data Pipeline & Infrastructure**
12. **Database Schema**
13. **Monorepo Structure**

---

## 1. Project Overview & Philosophy
AlphaJEE is a high-precision adaptive learning ecosystem for JEE Advanced and SAT preparation. It combines three distinct, purpose-built modes — each powered by a different engine — into a single cohesive product. The fundamental design principle is that measurement, competition, and learning are separate problems that require separate solutions. They are never mixed.

### 1.1 The Three Modes at a Glance

| Mode | Engine & Purpose |
| :--- | :--- |
| **Practice Mode** | The AI Coach. A reinforcement learning agent (PPO) dynamically controls difficulty to keep every student in their personal flow state. Fully personalised via Learning DNA. |
| **Mock / Adaptive Mode** | The Ruler. An exact replica of JEE Advanced and SAT paper structures. Pure IRT math. Zero AI interference. Produces defensible, valid scores. |
| **Challenge Mode** | The Arena. Scheduled multiplayer competitions. Any user can create a challenge. Glicko-2 ratings update after each challenge. No host advantage. |

### 1.2 Core Design Principles
- Each mode uses a different engine — they never interfere with each other.
- The mock is a ruler — it must never change based on who it measured.
- The RL agent is proprietary — it is the product's core moat.
- Free users experience the full product — they are rate-limited, not feature-limited.
- Free users are also training data — their sessions fund the RL agent's improvement.
- Trust score protects data integrity silently — cheaters are never told they are flagged.

---

## 2. Mathematical Core

### 2.1 Two-Parameter Logistic IRT (2PL)
2PL Item Response Theory is the psychometric foundation of the entire system. It models the probability that a student of ability θ correctly answers a question with difficulty b and discrimination a.

> **Formula:** $P(\theta) = \frac{1}{1 + e^{-a \cdot (\theta - b)}}$ where $\theta \in [-3, +3]$

| Parameter | Meaning & Range |
| :--- | :--- |
| **θ (Theta)** | Student ability on a standardised scale. −3 is very weak, 0 is average, +3 is exceptional. Estimated via Newton-Raphson MLE after every response. |
| **b (Difficulty)** | Item difficulty in logit units. b = 0 means a student of average ability has exactly 50% chance of success. Range: [−4, +4]. |
| **a (Discrimination)** | How sharply the question separates strong from weak students. High a means the question is highly informative. Clipped to [0.3, 3.5]. |
| **P(θ)** | The model's predicted probability of a correct answer. Used by the RL agent, the Midnight Crunch, and the adaptive mock branching logic. |

### 2.2 Newton-Raphson MLE for θ Estimation
After every question response in Practice mode, and after every module in Mock mode, theta is updated iteratively using the Newton-Raphson method.
- Gradient (first derivative) — measures direction and magnitude of ability change.
- Hessian (second derivative) — measures curvature, used to scale the step size.
- Step is clipped to [−1, +1] to prevent theta from jumping wildly on a single response.
- Converges in 15–20 iterations for typical response patterns.

### 2.3 Glicko-2 Rating System
Glicko-2 is used for all competitive ratings — student ratings in Challenge mode and question ratings in the question bank.

| Component | Description |
| :--- | :--- |
| **Rating (μ)** | The estimated strength. Students start at 1200. Questions are seeded from their b_value: Rating = 1500 + (b × 150). |
| **Rating Deviation (RD)** | Uncertainty in the rating. Starts high (350) and shrinks as more games or attempts are recorded. |
| **Volatility (σ)** | How erratic performance is over time. Caps how fast rating can swing after an unexpected result. Default: 0.06. |

**Why Glicko-2 over Elo for questions:** A question seen by 10,000 students should barely move after one more attempt. A brand-new question with only 5 attempts should move dramatically. RD handles this automatically.

### 2.4 Fisher Information — Optimal Question Selection
Used in Mock mode to select the most informative next question. Information is maximised when question difficulty matches student ability.

> **Formula:** $I(\theta) = a^2 \cdot P(\theta) \cdot (1 - P(\theta))$ → maximised when $P(\theta) \approx 0.5$

---

## 3. Practice Mode — The AI Coach
Practice mode is the only mode where AI directly controls the student experience. A PPO reinforcement learning agent acts as a real-time tutor targeting a 75% success rate.

### 3.1 What the Agent Observes — State Vector

| State Component | Source & Meaning |
| :--- | :--- |
| **θ (Current Ability)** | Student's latest Glicko-2 converted theta. Updated via Newton-Raphson. |
| **Fatigue Score** | Real-time Eye Aspect Ratio (EAR) from MediaPipe webcam feed. |
| **Typing Entropy** | Variance in inter-key latency. Signals cognitive overload or distraction. |
| **Solve Velocity** | Student's time-to-solve relative to population average. |
| **Recent Accuracy** | Rolling accuracy over the last 5 questions. |
| **Current b_value** | Difficulty of the question just answered. |
| **Fatigue Threshold** | From Learning DNA — personal limit before performance degrades. |
| **Frustration Threshold** | From Learning DNA — tolerance for consecutive wrong answers. |

### 3.2 What the Agent Decides — Action
The agent outputs a target b_value for the next question.
> **Action Space:** target_b ∈ [−3.0, +3.0]

### 3.3 How the Agent is Rewarded
- **+1.0 (Full Reward):** Correct answer at appropriate difficulty. Accuracy near 75%.
- **0.0 (Wasted):** Correct answer but question was too easy.
- **−1.0 (Penalty):** Incorrect answer combined with high fatigue.
- **Shaped (Intermediate):** Partial reward proportional to deviation from 75% target accuracy.

### 3.4 Learning DNA — Personalisation
Stored as a JSONB profile:
- `fatigue_threshold`: EAR value where accuracy drops.
- `frustration_threshold`: Wrong answers before disengagement.
- `velocity_baseline`: Natural solve speed.
- `concept_mastery`: Rolling accuracy per concept.

---

## 4. RL Agent — Deep Technical Breakdown

### 4.1 What is Reinforcement Learning Here
The agent learns a policy (mapping observations to actions) by interacting with a simulated student environment.

### 4.2 Why PPO (Proximal Policy Optimisation)
PPO is chosen for its stability in continuous control problems. It clips policy updates to prevent "catastrophic forgetting."

### 4.3 The Libraries
- **Gymnasium:** Defines the environment (`reset`, `step`).
- **Stable Baselines3 (SB3):** Provides the PPO implementation and training loop.
- **PyTorch:** Deep learning backend.
- **ONNX:** Portable format to run models on Cloudflare AI Workers.
- **Weights & Biases (wandb):** Experiment tracking and reward logging.

### 4.4 Policy Network Architecture
- **Input:** 8 neurons (State Vector).
- **Hidden:** 2 layers of 64 neurons (tanh activation).
- **Output:** 1 neuron (target b_value).

---

## 5. RL Agent — Build Roadmap

**Repo Name:** `alphajee-rl-agent`

### 5.1 Repository Structure
```text
alphajee-rl-agent/
├── env/
│   ├── alphajee_env.py      # Custom Gymnasium environment
│   ├── student_sampler.py   # Synthetic student generator
│   └── irt_utils.py         # 2PL math
├── training/
│   ├── train.py             # Main PPO script
│   └── config.yaml          # Hyperparameters
├── export/
│   ├── export_onnx.py       # PyTorch -> ONNX
│   └── validate_onnx.py     # Check accuracy
├── crunch/
│   ├── download_logs.py     # Pull from Supabase
│   └── retrain.py           # Fine-tuning
└── models/                  # Saved .onnx files
```

### 5.2 Build Steps
1. **Step 1:** Write the `AlphaJEEEnv`. Simulate fatigue, solve velocity, and 2PL responses.
2. **Step 2:** Build the Synthetic Sampler based on real JEE calibration distributions.
3. **Step 3:** Train with PPO using `stable-baselines3`. Target `ep_rew_mean` of 0.7+.
4. **Step 4:** Evaluate against held-out synthetic students.
5. **Step 5:** Export to ONNX (`opset_version=11`).
6. **Step 6:** Deploy to Cloudflare AI Workers.
7. **Step 7:** Implement Midnight Crunch for nightly fine-tuning on real user data.

---

## 6. Mock / Adaptive Mode — The Ruler

### 6.1 JEE Advanced Structure
| Question Type | Description | Marking |
| :--- | :--- | :--- |
| **Single Correct** | One correct option from four. | +3 / −1 |
| **Multiple Correct** | One or more correct options. Partial marks. | +4 / −2 |
| **Integer Type** | Numerical answer. No options. | +4 / 0 |
| **Paragraph Based** | Two questions sharing a common reading passage. | +3 / −1 |

### 6.2 SAT Adaptive Structure
- **Module 1:** Fixed set for everyone.
- **Branch Decision:** If theta > threshold → Hard Module 2. Else → Easy Module 2.
- **Score Equating:** Final score derived from combined theta estimate using IRT equating.

---

## 7. Challenge / Lobby Mode — The Arena

### 7.1 Challenge Lifecycle
1. **Creation:** Creator sets parameters. Ticket consumed.
2. **Lobby Open:** Players join via code/browser. Trust score checked.
3. **Auto-Start:** System fires at scheduled time. Trimmed weighted mean theta calculated.
4. **Live Challenge:** Global timer. Simultaneous question access.
5. **Time Expires:** Server-side lock. Glicko-2 updates applied in one transaction.

### 7.2 Question Selection
Uses a **Trimmed Weighted Mean**:
- Drop top/bottom 10% of ratings.
- Weight remaining players by Trust Score.
- Match difficulty (b) to the resulting mean theta.
- High spread lobby = wider difficulty range. Low spread = tight range.

---

## 8. Trust Score System
Hidden value ($T_u$) between 0.0 and 1.0.

### 8.1 Signals
- Solve velocity (faster than population average).
- Answer consistency (getting hard right, easy wrong).
- Lobby drops or bot-like timing patterns.

### 8.2 Effects
- **Low Trust (<0.5):** Cannot create public challenges.
- **Very Low Trust (<0.15):** Hard-blocked from joining challenges.
- **Shadow-banning:** Moved to a "Sandbox" lobby with only other low-trust users.
- **Weighting:** Cheater responses are mathematically down-weighted in IRT updates.

---

## 9. Monetization & Free Tier Strategy

### 9.1 Comparison
| Feature | Free | Paid |
| :--- | :--- | :--- |
| Practice Questions | 20 per day | Unlimited |
| Mock Attempts | 1 per week | Unlimited |
| Challenge Creation | 3 Tickets/day | Unlimited |
| RL Personalisation | Full | Full |

### 9.2 Pricing
- **1 Month:** ₹199
- **3 Months:** ₹479 (Save 20% - Recommended)
- **1 Year:** ₹1,299 (Save 46%)

---

## 10. Anti-Scraping & Security
- **Image Serving:** Questions delivered as signed, expiring URLs (5 min).
- **No QIDs:** Clients only see temporary session tokens.
- **Rate Limiting:** Max 5 questions/min per user.
- **Server-Side Scoring:** Client-side score calculation is ignored.

---

## 11. Data Pipeline & Infrastructure

### 11.1 Calibration Pipeline
1. `01_preprocess.ipynb`: Clean and score raw response data.
2. `02_irt_calibration.ipynb`: Fit 2PL models for a/b parameters.
3. `03_glicko_seed.ipynb`: Seed initial question ratings.
4. `midnight_crunch.py`: Nightly updates.

### 11.2 Tech Stack
- **Frontend:** Next.js / Flutter.
- **Edge Backend:** Cloudflare Workers (Logic) / Cloudflare AI (RL Inference).
- **Database:** Supabase (PostgreSQL + Realtime).
- **ML Training:** Stable Baselines3 + PyTorch.

---

## 12. Database Schema

### Table: questions
- `qid`: Unique ID (hidden).
- `a_value`, `b_value`: IRT parameters.
- `glicko_rating`, `rd`, `volatility`.
- `subject`, `chapter`, `concept`.

### Table: students
- `user_id`: PK.
- `glicko_rating` (Global & Subject-specific).
- `trust_score`: [0.0 - 1.0].
- `learning_dna`: JSONB.

### Table: attempts
- `user_id`, `qid`, `is_correct`, `time_taken_sec`.
- `fatigue_score`: From Practice mode.
- `mode`: practice/mock/challenge.

---

## 13. Monorepo Structure

### alphajee/ (Main)
- `packages/workers/`: Practice, Mock, and Challenge logic.
- `packages/web/`: Next.js app.
- `packages/mobile/`: Flutter app.
- `calibration/`: Scripts for IRT parameter estimation.
- `supabase/`: Migrations and seed files.

### alphajee-rl-agent/ (Separate)
- `env/`: The Simulation environment.
- `training/`: PPO training logic.
- `export/`: ONNX export tools.
- `models/`: Versioned AI models.

---
*End of Document*
**AlphaJEE · Version 3.0 · Confidential**
