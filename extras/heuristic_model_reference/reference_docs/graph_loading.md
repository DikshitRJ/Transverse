# AlphaJEE — Knowledge Graph Architecture & Loading Strategy

## Overview

The knowledge graph is the visual centrepiece of the AlphaJEE student dashboard. It renders the entire JEE syllabus as a force-directed node graph, where every node represents a chapter and every edge represents a prerequisite dependency. The visual properties of each node and edge are driven entirely by the student's live psychometric data, making the graph a real-time diagnostic of where they stand across the full syllabus.

This document covers the complete architecture: how the graph is defined, loaded into memory, hydrated with user data, served to the frontend, and rendered visually.

---

## 1. Design Principle: Static Structure, Dynamic State

The graph has two completely separate layers that are intentionally kept apart:

**The Structure Layer** defines the shape of the graph: which chapters exist, which subject and group they belong to, and which chapters are prerequisites of which. This never changes at runtime. It reflects the fixed JEE syllabus curriculum and is authored once by hand.

**The State Layer** defines how each node looks for a specific user: how big it is, how bright, how far its edges stretch. This is computed at request time by merging the structure layer with the user's `learning_stats.chapters` JSONB.

Keeping these separate means the graph topology never touches the database. Only the user's performance data does.

---

## 2. The Static Knowledge Graph File

The graph structure lives in a single JSON file committed to the repository at `internal/graph/syllabus.json`. It is loaded entirely into Go's heap at server boot and never reloaded unless the process restarts.

### File Structure

Each subject key maps to chapter entries. The key is the slug (matches `questions.chapter` in the DB), and the `chapter` field is the human-readable display name:

```json
{
  "physics": {
    "units-and-measurements": {
      "chapter": "Units and Measurements",
      "group": "mechanics",
      "prerequisites": []
    },
    "motion-in-a-straight-line": {
      "chapter": "Motion in a Straight Line",
      "group": "mechanics",
      "prerequisites": ["units-and-measurements"]
    },
    "electrostatics": {
      "chapter": "Electrostatics",
      "group": "electricity",
      "prerequisites": ["maths/vector-algebra", "maths/differentiation"]
    },
    "current-electricity": {
      "chapter": "Current Electricity",
      "group": "electricity",
      "prerequisites": ["electrostatics"]
    }
  },
  "chemistry": {
    "some-basic-concepts-of-chemistry": {
      "chapter": "Some Basic Concepts of Chemistry",
      "group": "physical-chemistry",
      "prerequisites": []
    },
    "electrochemistry": {
      "chapter": "Electrochemistry",
      "group": "physical-chemistry",
      "prerequisites": ["solutions", "redox-reactions", "thermodynamics"]
    }
  },
  "maths": {
    "sets-and-relations": {
      "chapter": "Sets and Relations",
      "group": "algebra",
      "prerequisites": []
    },
    "vector-algebra": {
      "chapter": "Vector Algebra",
      "group": "coordinate-geometry",
      "prerequisites": ["sets-and-relations", "trigonometric-ratio-and-identites"]
    }
  }
}
```

### Cross-Subject Prerequisites

Notice that `electrostatics` lists `"maths/vector-algebra"` and `"maths/differentiation"` as prerequisites. The format `subject/chapter-slug` denotes a cross-subject dependency. The graph loader resolves these as edges between nodes in different subject clusters. This is the primary source of inter-subject strain edges visible on the dashboard.

---

## 3. Boot-Time Loading in Go

The graph file is parsed once at startup into an in-memory struct. All request handlers receive a pointer to this struct via dependency injection. There is zero file I/O after boot.

### Go Structs

```go
// internal/graph/graph.go

package graph

import (
    "encoding/json"
    "fmt"
    "os"
    "strings"
)

// ChapterNode is one entry from the syllabus JSON.
type ChapterNode struct {
    Chapter       string   `json:"chapter"`       // human-readable display name
    Group         string   `json:"group"`          // subject group (e.g. "mechanics")
    Prerequisites []string `json:"prerequisites"`  // slugs, may include "subject/" prefix
}

// SyllabusGraph is the fully parsed in-memory graph.
// Outer key: subject ("physics", "chemistry", "maths")
// Inner key: chapter slug (e.g. "electrostatics")
type SyllabusGraph map[string]map[string]ChapterNode

// Load reads the JSON file from disk and returns the parsed graph.
// Call once at boot. Fatal on failure — the graph is non-optional.
func Load(path string) (SyllabusGraph, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("graph: failed to read syllabus file: %w", err)
    }

    var g SyllabusGraph
    if err := json.Unmarshal(data, &g); err != nil {
        return nil, fmt.Errorf("graph: failed to parse syllabus JSON: %w", err)
    }

    return g, nil
}

// ResolvePrerequisite parses a prerequisite string.
// "units-and-measurements"     → same subject as the chapter being resolved
// "maths/vector-algebra"       → cross-subject reference
func ResolvePrerequisite(prereq, defaultSubject string) (subject, chapter string) {
    parts := strings.SplitN(prereq, "/", 2)
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return defaultSubject, prereq
}
```

### Wiring in `main.go`

```go
syllabusGraph, err := graph.Load(cfg.SyllabusGraphPath)
if err != nil {
    log.Fatalf("failed to load syllabus graph: %v", err)
}

// GraphService handles hydration — see section 4
graphSvc := services.NewGraphService(syllabusGraph, statsRepo, userRepo)
graphHandler := handlers.NewGraphHandler(graphSvc)
```

---

## 4. Hydration: GraphService

The `GraphService` (`internal/services/graph.go`) is the core hydration engine. It takes the static `SyllabusGraph` and the user's `learning_stats.chapters` JSONB and produces a fully personalised `GraphPayload`.

### Service Signature

```go
type GraphService struct {
    syllabusGraph graph.SyllabusGraph
    statsRepo     *repository.StatsRepo
    userRepo      *repository.UserRepo
}

func (s *GraphService) HydrateForUser(ctx, userID) (*models.GraphPayload, error)
```

### Hydration Flow

```
HydrateForUser(ctx, userID)
  1. EnsureDevUser — creates user + learning_stats row if first visit
  2. SELECT chapters FROM learning_stats WHERE user_id = $1
  3. If user is dev-user-001 AND chapters is empty {
       seedDemoData() — writes 87 chapters with realistic mastery to learning_stats
       re-fetch chapters
     }
  4. For each chapter in SyllabusGraph {
       lookup chapters[slug] from user stats
       if found → use real mastery, theta, glicko_rd, last_seen
       if not found → defaults: mastery=0, theta=1300, glicko_rd=350
       build GraphNode with chapter display name
     }
  5. For each prerequisite edge {
       compute strainIndex = prereq.mastery - child.mastery
       if cross-subject → set cross_subject flag
     }
  6. Return GraphPayload{ Nodes, Edges }
```

### Demo Data Seeding

When `dev-user-001` visits the graph for the first time with empty practice history, the service seeds all 87 chapters with realistic JEE student mastery data:

- **Strong areas** (70-83%): units-and-measurements, sets-and-relations, some-basic-concepts-of-chemistry, trigonometric-ratios, vector-algebra, functions, periodic-table
- **Moderate areas** (40-65%): electrostatics, current-electricity, laws-of-motion, differentiation, limits, probability, chemical-bonding, basics-of-organic-chemistry
- **Weak areas** (6-25%): rotational-motion, electromagnetic-waves, salt-analysis, differential-equations, coordination-compounds

Each seeded chapter gets: `theta`, `mastery_score`, `glicko_rating`, `glicko_rd=180`, `attempt_count`, `correct_count`, `avg_time_ms`, and a `last_practiced_at` date.

### Handler (`internal/handlers/graph_handler.go`)

The handler is intentionally thin — it extracts the user ID from context and delegates entirely to the service:

```go
func (h *GraphHandler) GetGraph(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.UserIDFromContext(r.Context())
    if !ok { /* 401 */ }

    payload, err := h.svc.HydrateForUser(r.Context(), userID)
    if err != nil { /* 500 */ }

    writeJSON(w, http.StatusOK, payload)
}
```

---

## 5. API Response Shape

**Endpoint:** `GET /api/v1/graph`

Returns all nodes and edges. The `chapter` field is the human-readable name from `syllabus.json`:

```json
{
  "nodes": [
    {
      "id":           "electrostatics",
      "chapter":      "Electrostatics",
      "subject":      "physics",
      "group":        "electricity",
      "mastery_score": 65.6,
      "glicko_rd":    180.0,
      "theta":        1850.0,
      "last_seen":    "2025-05-27T14:00:00Z"
    },
    {
      "id":           "current-electricity",
      "chapter":      "Current Electricity",
      "subject":      "physics",
      "group":        "electricity",
      "mastery_score": 53.3,
      "glicko_rd":    180.0,
      "theta":        1700.0,
      "last_seen":    "2025-05-26T10:00:00Z"
    }
  ],
  "edges": [
    {
      "from":          "physics/electrostatics",
      "to":            "physics/current-electricity",
      "strain_index":  12.3,
      "cross_subject": false
    },
    {
      "from":          "maths/vector-algebra",
      "to":            "physics/electrostatics",
      "strain_index":  4.4,
      "cross_subject": true
    }
  ]
}
```

### GraphNode Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Chapter slug, matches `questions.chapter` in DB |
| `chapter` | string | Human-readable name from `syllabus.json` |
| `subject` | string | `"physics"`, `"chemistry"`, or `"maths"` |
| `group` | string | Subject group (e.g. `"electricity"`, `"calculus"`) |
| `mastery_score` | float64 | 0–100, normalised from theta |
| `glicko_rd` | float64 | 30–350, uncertainty — controls node opacity |
| `theta` | float64 | Raw IRT rating (1300–2800 scale) |
| `last_seen` | string | ISO8601 timestamp or empty string |

### GraphEdge Fields

| Field | Type | Description |
|-------|------|-------------|
| `from` | string | `"subject/slug"` of the prerequisite chapter |
| `to` | string | `"subject/slug"` of the dependent chapter |
| `strain_index` | float64 | `prereq.mastery - child.mastery` — positive means gap |
| `cross_subject` | bool | True if edge crosses subject boundaries |

### Empty State

If the user has never practiced any chapter (all `mastery_score = 0`), the frontend checks this client-side and displays:

> 🌱 Start learning to view your personalised graph

---

## 6. Visual Mapping Reference

The frontend (see `graph-visualiser/001-graph.md`) maps backend fields to 3D visual properties:

| Backend Field | Visual Property | Formula |
|---|---|---|
| `mastery_score` | Sphere radius | `0.4 + (score / 100) * 1.0` |
| `glicko_rd` | Opacity | `max(0.3, 1.0 - ((rd-50)/350) * 0.7)` |
| `subject` | Colour | physics=`#3b82f6`, chemistry=`#22c55e`, maths=`#a855f7` |
| `strain_index > 50` | Edge colour | Red (`#ef4444`) — critical gap |
| `strain_index < 0` | Edge colour | Amber (`#f59e0b`) — unusual inversion |
| `cross_subject` | Edge opacity | Dimmer (`max(0.08, 0.25 - intensity * 0.15)`) |

---

## 7. Important Constraints

**Prerequisites are diagnostic only.** Per the original AlphaJEE specification, the prerequisite graph is never used to interrupt a student mid-session. If a student chooses to practice electrostatics while their vector-algebra mastery is low, the engine does not redirect them. The prerequisite edges appear exclusively on the post-session mind map to highlight structural weaknesses after the fact.

**The graph never writes to the database.** The graph service is purely a read-and-merge operation. All writes happen via the session and stats services. The only exception is the demo data seed for `dev-user-001`, which writes initial chapter stats so the visualiser has data to render.

**New chapters in the syllabus JSON require no migration.** If a chapter is added to `syllabus.json`, it appears on the graph immediately on next boot with `mastery = 0`. No schema changes, no data migrations.

**Chapter slugs must match the DB exactly.** The key used in `syllabus.json` must exactly match the `chapter` column values in the `questions` table (case-sensitive). Both are lowercase kebab-case in this system (e.g. `"electrostatics"`, not `"Electrostatics"`).

---

## 8. File Location Summary

```
alphajee/
├── internal/
│   ├── graph/
│   │   ├── graph.go            ← Load(), ResolvePrerequisite(), ChapterNode structs
│   │   └── syllabus.json       ← Static prerequisite graph definition (87 chapters)
│   ├── services/
│   │   └── graph.go            ← GraphService.HydrateForUser(), seedDemoData()
│   └── handlers/
│       └── graph_handler.go    ← GET /api/v1/graph — thin HTTP wrapper
├── graph-visualiser/
│   ├── flake.nix               ← Nix dev shell with nodejs_22
│   ├── .envrc                  ← direnv auto-activation
│   ├── package.json            ← Vite + React 18 + R3F + drei + d3-force
│   └── src/
│       ├── App.tsx             ← orchestrates data fetch → layout → render
│       ├── api/graph.ts        ← fetchGraph() → GraphPayload
│       ├── hooks/
│       │   ├── useGraphData.ts ← loading / loaded / empty / error state machine
│       │   └── useForceLayout.ts ← d3-force 2D sim with subject Z-layers
│       ├── components/
│       │   ├── GraphCanvas.tsx  ← R3F Canvas, controls, lighting
│       │   ├── GraphNode.tsx    ← sphere mesh + label
│       │   ├── GraphEdge.tsx    ← line between nodes
│       │   ├── EmptyState.tsx   ← "Start learning" landing
│       │   ├── NodeDetails.tsx  ← click-to-inspect panel
│       │   ├── Legend.tsx       ← subject colour key
│       │   └── Toolbar.tsx      ← search + subject filter
│       ├── types/graph.ts       ← TypeScript interfaces
│       └── utils/colors.ts      ← subject/mastery colour mapping
```

---

## 9. Boot Sequence Summary

```
Server starts
     ↓
graph.Load(cfg.SyllabusGraphPath)       // reads syllabus.json from disk
     ↓ parsed into SyllabusGraph struct in RAM
     ↓ fatal if file missing or malformed
     ↓
SyllabusGraph injected into GraphService (and LearnService)
     ↓
GET /api/v1/graph request arrives
     ↓
middleware.MockAuth → injects userID into context
     ↓
handlers/graph_handler.GetGraph()
     ↓ calls GraphService.HydrateForUser(userID)
          ↓
          userRepo.EnsureDevUser()  // creates row on first visit
          ↓
          statsRepo.Get(userID)     // SELECT chapters FROM learning_stats
          ↓
          if dev-user-001 and empty → seedDemoData() writes 87 chapters
          ↓
          buildPayload() iterates SyllabusGraph, merges with chapters
          ↓
          returns *models.GraphPayload
     ↓
handlers/graph_handler serialises to JSON
     ↓
Frontend (graph-visualiser) receives payload
     ↓
useGraphData → determines loading / loaded / empty / error
     ↓
useForceLayout → runs d3-force 2D simulation with Z-layers per subject
     ↓
R3F Canvas renders nodes, edges, labels, controls
```

Total DB queries per graph request: **one** (two on demo user first visit due to seed + re-fetch).
