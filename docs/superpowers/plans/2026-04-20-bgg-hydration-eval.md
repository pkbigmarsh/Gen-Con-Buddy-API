# BGG Hydration Evaluation Tool — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go binary (`cmd/evalbgg`) that runs 18 BGG-matching strategies against Gen Con BGM events and outputs a comparison CSV for scoring matcher quality.

**Architecture:** Standalone `main` package under `cmd/evalbgg/` — not wired into the existing CLI. Loads both CSVs into memory, extracts unique `(Game System, Rules Edition)` combos, runs all 18 matchers against each combo, writes one output row per combo with each matcher's result and similarity score.

**Tech Stack:** Go standard library (`encoding/csv`, `strings`, `unicode`), `golang.org/x/text/encoding/charmap` (already in go.mod) for Windows-1252 Gen Con CSV decoding, `github.com/stretchr/testify` (already in go.mod) for tests.

---

## File Map

| File | Responsibility |
|------|---------------|
| `cmd/evalbgg/types.go` | `BGGGame`, `GenConCombo`, `MatchResult`, `Matcher` interface |
| `cmd/evalbgg/normalize.go` | `normalize()`, `isInformativeEdition()`, `extractTitleDerived()`, stopword/generic-term sets |
| `cmd/evalbgg/normalize_test.go` | Tests for all normalize.go functions |
| `cmd/evalbgg/score.go` | `similarityScore()` (Levenshtein), `jaccardScore()` (token Jaccard), helpers |
| `cmd/evalbgg/score_test.go` | Tests for similarity and Jaccard functions |
| `cmd/evalbgg/load.go` | `loadBGG()`, `loadGenConCombos()` — reads CSVs into typed structs |
| `cmd/evalbgg/matchers.go` | All 18 `Matcher` implementations + shared helpers (`exactMatch`, `bestScoredMatch`, `filterBySystem`, tiebreakers) |
| `cmd/evalbgg/matchers_test.go` | Tests for each matcher using a small in-memory BGG fixture |
| `cmd/evalbgg/output.go` | `writeCSV()` — writes the comparison CSV |
| `cmd/evalbgg/main.go` | Flag parsing, orchestration: load → deduplicate → run matchers → write output |

---

## Task 1: Package setup and types

**Files:**
- Create: `cmd/evalbgg/types.go`

- [ ] **Step 1: Create `cmd/evalbgg/types.go`**

```go
package main

// BGGGame holds all fields from the BGG CSV for a single game.
type BGGGame struct {
	ID            string
	Name          string
	YearPublished string
	Rank          int     // 0 = unranked
	BayesAverage  float64
	Average       float64
	UsersRated    int
	IsExpansion   bool
	AbstractsRank string
	CGSRank       string
	ChildrensRank string
	FamilyRank    string
	PartyRank     string
	StrategyRank  string
	ThematicRank  string
	WarRank       string
}

// GenConCombo is one unique (Game System, Rules Edition) pair from BGM events.
type GenConCombo struct {
	GameSystem   string
	RulesEdition string
	RepTitle     string // most common Title across events sharing this combo
	EventCount   int
}

// MatchResult is the output of a single Matcher for a single combo.
// BGGGame is nil when no match was found.
// Score is the raw similarity value (1.0 for exact matchers, [0,1] for fuzzy/token).
type MatchResult struct {
	BGGGame *BGGGame
	Score   float64
}

// Matcher is the interface all 18 strategies implement.
type Matcher interface {
	Name() string
	Match(combo GenConCombo, candidates []BGGGame) MatchResult
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd cmd/evalbgg && go build . 2>&1 || true
```

Expected: fails with "no Go files" or "no main function" — that's fine, we just want no syntax errors in types.go.

- [ ] **Step 3: Commit**

```bash
git add cmd/evalbgg/types.go
git commit -m "feat(evalbgg): add types"
```

---

## Task 2: Normalization functions (TDD)

**Files:**
- Create: `cmd/evalbgg/normalize.go`
- Create: `cmd/evalbgg/normalize_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/evalbgg/normalize_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Axis & Allies", "axis & allies"},
		{"Axis & Allies: 1942", "axis & allies  1942"},  // colon becomes space
		{"Ticket to Ride!", "ticket to ride"},
		{"  Extra   Spaces  ", "extra spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, normalize(tt.input))
		})
	}
}

func TestIsInformativeEdition(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1st", false},
		{"2nd", false},
		{"Global 1942 2nd", true},
		{"Prison Outbreak", true},
		{"1941", true},
		{"Revised", false},
		{"Classic", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, isInformativeEdition(tt.input))
		})
	}
}

func TestExtractTitleDerived(t *testing.T) {
	tests := []struct {
		system string
		title  string
		want   string
	}{
		{
			system: "Axis & Allies",
			title:  "Axis & Allies Global 1942",
			want:   "global 1942",
		},
		{
			system: "Terraforming Mars",
			title:  "Terraforming Mars: Ares Expedition Mini Tournament",
			want:   "ares expedition",
		},
		{
			system: "Wingspan",
			title:  "Wingspan Tournament Finals",
			want:   "", // all tokens are stopwords or system tokens
		},
		{
			system: "Twilight Imperium",
			title:  "Twilight Imperium with Prophecy of Kings",
			want:   "prophecy kings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			require.Equal(t, tt.want, extractTitleDerived(tt.system, tt.title))
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cmd/evalbgg && go test ./... -run "TestNormalize|TestIsInformative|TestExtractTitle" -v 2>&1
```

Expected: compile error — functions not defined yet.

- [ ] **Step 3: Create `cmd/evalbgg/normalize.go`**

```go
package main

import (
	"strings"
	"unicode"
)

var titleStopwords = map[string]bool{
	"tournament": true, "finals": true, "final": true, "qualifier": true,
	"round": true, "semi-final": true, "beginner": true, "beginners": true,
	"experienced": true, "advanced": true, "mini": true, "open": true,
	"championship": true, "preliminary": true, "non-qualifier": true,
	"event": true, "demo": true, "intro": true, "introduction": true,
	"teach": true, "teaching": true, "with": true, "for": true,
	"to": true, "the": true, "a": true, "an": true, "of": true,
	"in": true, "and": true, "by": true, "at": true, "upgraded": true,
	"components": true, "expansion": true,
}

var genericEditionTerms = map[string]bool{
	"1st": true, "2nd": true, "3rd": true, "4th": true, "5th": true,
	"first": true, "second": true, "third": true, "revised": true,
	"standard": true, "deluxe": true, "basic": true, "classic": true,
}

// normalize lowercases s, strips punctuation (keeping & and alphanumerics),
// and collapses whitespace.
func normalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r == '&' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// isInformativeEdition returns true if edition contains tokens beyond bare ordinals.
func isInformativeEdition(edition string) bool {
	for _, w := range strings.Fields(normalize(edition)) {
		if !genericEditionTerms[w] {
			return true
		}
	}
	return false
}

// extractTitleDerived strips game system tokens and title stopwords from title,
// returning the remaining edition-like tokens joined by spaces.
func extractTitleDerived(gameSystem, title string) string {
	sysTokens := make(map[string]bool)
	for _, w := range strings.Fields(normalize(gameSystem)) {
		sysTokens[w] = true
	}

	var result []string
	for _, w := range strings.Fields(normalize(title)) {
		if !sysTokens[w] && !titleStopwords[w] {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestNormalize|TestIsInformative|TestExtractTitle" -v
```

Expected: all pass. If `TestNormalize` fails on the colon case, adjust the expected value to match actual output (the test expectation is illustrative — what matters is that colons become spaces and output is collapsed).

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/normalize.go cmd/evalbgg/normalize_test.go
git commit -m "feat(evalbgg): add normalization functions"
```

---

## Task 3: Scoring functions (TDD)

**Files:**
- Create: `cmd/evalbgg/score.go`
- Create: `cmd/evalbgg/score_test.go`

- [ ] **Step 1: Write failing tests**

Create `cmd/evalbgg/score_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64 // score must be >= min
		max  float64 // score must be <= max
	}{
		{"axis & allies", "axis & allies", 1.0, 1.0},
		{"", "", 1.0, 1.0},
		{"axis & allies", "completely different", 0.0, 0.3},
		{"axis & allies 1942", "axis & allies  1942", 0.9, 1.0},
		{"wingspan", "wingspam", 0.8, 1.0}, // one char different
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := similarityScore(tt.a, tt.b)
			require.GreaterOrEqual(t, score, tt.min)
			require.LessOrEqual(t, score, tt.max)
		})
	}
}

func TestJaccardScore(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"axis allies 1942", "axis allies 1942", 1.0, 1.0},
		{"axis allies", "axis allies 1942", 0.6, 0.8},
		{"wingspan", "ark nova", 0.0, 0.1},
		{"", "", 1.0, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := jaccardScore(tt.a, tt.b)
			require.GreaterOrEqual(t, score, tt.min)
			require.LessOrEqual(t, score, tt.max)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cmd/evalbgg && go test ./... -run "TestSimilarity|TestJaccard" -v 2>&1
```

Expected: compile error — functions not defined.

- [ ] **Step 3: Create `cmd/evalbgg/score.go`**

```go
package main

import "strings"

// similarityScore returns normalized Levenshtein similarity in [0,1].
func similarityScore(a, b string) float64 {
	if a == b {
		return 1.0
	}
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 && lb == 0 {
		return 1.0
	}
	if la == 0 || lb == 0 {
		return 0.0
	}
	dist := levenshtein(ra, rb)
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func levenshtein(a, b []rune) int {
	la, lb := len(a), len(b)
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// jaccardScore returns token-set Jaccard similarity in [0,1].
func jaccardScore(a, b string) float64 {
	tokA := tokenSet(a)
	tokB := tokenSet(b)
	if len(tokA) == 0 && len(tokB) == 0 {
		return 1.0
	}
	intersection := 0
	for t := range tokA {
		if tokB[t] {
			intersection++
		}
	}
	union := len(tokA) + len(tokB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func tokenSet(s string) map[string]bool {
	tokens := strings.Fields(s)
	set := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		set[t] = true
	}
	return set
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestSimilarity|TestJaccard" -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/score.go cmd/evalbgg/score_test.go
git commit -m "feat(evalbgg): add fuzzy and token scoring functions"
```

---

## Task 4: CSV loading

**Files:**
- Create: `cmd/evalbgg/load.go`

- [ ] **Step 1: Create `cmd/evalbgg/load.go`**

```go
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// loadBGG reads the BGG CSV and returns all non-expansion games.
func loadBGG(path string) ([]BGGGame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open bgg csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read bgg headers: %w", err)
	}
	idx := headerIndex(headers)

	var games []BGGGame
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read bgg row: %w", err)
		}
		g := BGGGame{
			ID:            field(row, idx, "id"),
			Name:          field(row, idx, "name"),
			YearPublished: field(row, idx, "yearpublished"),
			IsExpansion:   field(row, idx, "is_expansion") == "1",
			Rank:          parseInt(field(row, idx, "rank")),
			BayesAverage:  parseFloat(field(row, idx, "bayesaverage")),
			Average:       parseFloat(field(row, idx, "average")),
			UsersRated:    parseInt(field(row, idx, "usersrated")),
			AbstractsRank: field(row, idx, "abstracts_rank"),
			CGSRank:       field(row, idx, "cgs_rank"),
			ChildrensRank: field(row, idx, "childrensgames_rank"),
			FamilyRank:    field(row, idx, "familygames_rank"),
			PartyRank:     field(row, idx, "partygames_rank"),
			StrategyRank:  field(row, idx, "strategygames_rank"),
			ThematicRank:  field(row, idx, "thematic_rank"),
			WarRank:       field(row, idx, "wargames_rank"),
		}
		if !g.IsExpansion {
			games = append(games, g)
		}
	}
	return games, nil
}

// loadGenConCombos reads the Gen Con CSV and returns unique (GameSystem, RulesEdition)
// combos from BGM events, with representative title and event count.
func loadGenConCombos(path string) ([]GenConCombo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open gencon csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(transform.NewReader(f, charmap.Windows1252.NewDecoder()))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read gencon headers: %w", err)
	}
	idx := headerIndex(headers)

	type comboKey struct{ system, edition string }
	titleCounts := make(map[comboKey]map[string]int)
	counts := make(map[comboKey]int)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read gencon row: %w", err)
		}
		evtType := field(row, idx, "Event Type")
		if !strings.HasPrefix(evtType, "BGM") {
			continue
		}
		key := comboKey{
			system:  field(row, idx, "Game System"),
			edition: field(row, idx, "Rules Edition"),
		}
		title := field(row, idx, "Title")
		counts[key]++
		if titleCounts[key] == nil {
			titleCounts[key] = make(map[string]int)
		}
		titleCounts[key][title]++
	}

	var combos []GenConCombo
	for key, count := range counts {
		combos = append(combos, GenConCombo{
			GameSystem:   key.system,
			RulesEdition: key.edition,
			RepTitle:     mostCommon(titleCounts[key]),
			EventCount:   count,
		})
	}
	return combos, nil
}

func headerIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}
	return m
}

func field(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func mostCommon(counts map[string]int) string {
	var best string
	var bestCount int
	for k, v := range counts {
		if v > bestCount || (v == bestCount && k < best) {
			best = k
			bestCount = v
		}
	}
	return best
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd cmd/evalbgg && go build . 2>&1 || true
```

Expected: may fail with "no main" — that's fine. No type errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/evalbgg/load.go
git commit -m "feat(evalbgg): add CSV loaders for BGG and Gen Con data"
```

---

## Task 5: Matcher helpers and matchers 1–4 (system signal)

**Files:**
- Create: `cmd/evalbgg/matchers.go`
- Create: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Write failing tests for matchers 1–4**

Create `cmd/evalbgg/matchers_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// fixture is a small BGG dataset for matcher tests.
var fixture = []BGGGame{
	{ID: "1", Name: "Wingspan", Rank: 10, UsersRated: 50000, IsExpansion: false},
	{ID: "2", Name: "Wingspan: European Expansion", Rank: 0, UsersRated: 10000, IsExpansion: false},
	{ID: "3", Name: "Ark Nova", Rank: 2, UsersRated: 60000, IsExpansion: false},
	{ID: "4", Name: "Axis & Allies: 1941", Rank: 100, UsersRated: 5000, IsExpansion: false},
	{ID: "5", Name: "Axis & Allies: 1942", Rank: 80, UsersRated: 8000, IsExpansion: false},
	{ID: "6", Name: "Axis & Allies", Rank: 200, UsersRated: 20000, IsExpansion: false},
}

func TestExactSystemRank(t *testing.T) {
	m := exactSystemRank{}
	require.Equal(t, "exact-system-rank", m.Name())

	// exact match on "Wingspan" → picks ID 1 (only exact match)
	result := m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)
	require.Equal(t, 1.0, result.Score)

	// no match for unknown game
	result = m.Match(GenConCombo{GameSystem: "Unknown Game XYZ", RulesEdition: "1st", RepTitle: "Unknown"}, fixture)
	require.Nil(t, result.BGGGame)
}

func TestFuzzySystemRank(t *testing.T) {
	m := fuzzySystemRank{}
	require.Equal(t, "fuzzy-system-rank", m.Name())

	// fuzzy match on "Wingspaan" (typo) → should still find Wingspan
	result := m.Match(GenConCombo{GameSystem: "Wingspaan", RulesEdition: "1st", RepTitle: "Wingspaan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)
	require.Greater(t, result.Score, 0.7)

	// "Axis & Allies" fuzzy → multiple candidates; picks best rank (ID 5, rank 80)
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	// best rank among axis & allies entries is ID 5 (rank 80)
	require.Equal(t, "5", result.BGGGame.ID)
}

func TestFuzzySystemRated(t *testing.T) {
	m := fuzzySystemRated{}
	require.Equal(t, "fuzzy-system-rated", m.Name())

	// "Axis & Allies" fuzzy → picks most rated (ID 6, usersRated 20000)
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestTokenSystemRank(t *testing.T) {
	m := tokenSystemRank{}
	require.Equal(t, "token-system-rank", m.Name())

	// token match on "Axis Allies" (missing &) → finds axis & allies entries
	result := m.Match(GenConCombo{GameSystem: "Axis Allies", RulesEdition: "1st", RepTitle: "Axis Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Greater(t, result.Score, 0.5)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cmd/evalbgg && go test ./... -run "TestExact|TestFuzzy|TestToken" -v 2>&1 | head -20
```

Expected: compile error — matcher types not defined.

- [ ] **Step 3: Create `cmd/evalbgg/matchers.go` with helpers and matchers 1–4**

```go
package main

// tiebreakByRank returns true if a is a better pick than b by BGG rank.
// Rank 0 means unranked — always loses to a ranked game.
func tiebreakByRank(a, b BGGGame) bool {
	if a.Rank == 0 {
		return false
	}
	if b.Rank == 0 {
		return true
	}
	return a.Rank < b.Rank
}

// tiebreakByRated returns true if a has more users rated than b.
func tiebreakByRated(a, b BGGGame) bool {
	return a.UsersRated > b.UsersRated
}

// exactMatch finds all BGG games whose normalized name exactly matches query,
// then picks the best using tiebreak.
func exactMatch(query string, candidates []BGGGame, tiebreak func(a, b BGGGame) bool) MatchResult {
	var best *BGGGame
	for i := range candidates {
		c := &candidates[i]
		if normalize(c.Name) == query {
			if best == nil || tiebreak(*c, *best) {
				best = c
			}
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGGame: best, Score: 1.0}
}

// bestScoredMatch finds the BGG game with the highest score > 0 using scoreFn,
// using tiebreak when scores are equal.
func bestScoredMatch(query string, candidates []BGGGame, scoreFn func(a, b string) float64, tiebreak func(a, b BGGGame) bool) MatchResult {
	var best *BGGGame
	var bestScore float64
	for i := range candidates {
		c := &candidates[i]
		score := scoreFn(query, normalize(c.Name))
		if score <= 0 {
			continue
		}
		if best == nil || score > bestScore || (score == bestScore && tiebreak(*c, *best)) {
			best = c
			bestScore = score
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGGame: best, Score: bestScore}
}

// filterBySystem returns candidates whose normalized name fuzzy-matches
// gameSystem at or above threshold.
func filterBySystem(gameSystem string, candidates []BGGGame, threshold float64) []BGGGame {
	query := normalize(gameSystem)
	var filtered []BGGGame
	for _, c := range candidates {
		if similarityScore(query, normalize(c.Name)) >= threshold {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- Matchers 1–4: System signal ---

type exactSystemRank struct{}

func (exactSystemRank) Name() string { return "exact-system-rank" }
func (exactSystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return exactMatch(normalize(combo.GameSystem), candidates, tiebreakByRank)
}

type fuzzySystemRank struct{}

func (fuzzySystemRank) Name() string { return "fuzzy-system-rank" }
func (fuzzySystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, similarityScore, tiebreakByRank)
}

type fuzzySystemRated struct{}

func (fuzzySystemRated) Name() string { return "fuzzy-system-rated" }
func (fuzzySystemRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, similarityScore, tiebreakByRated)
}

type tokenSystemRank struct{}

func (tokenSystemRank) Name() string { return "token-system-rank" }
func (tokenSystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, jaccardScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactSystem|TestFuzzySystem|TestTokenSystem" -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matcher helpers and matchers 1-4"
```

---

## Task 6: Matchers 5–7 (always edition signal)

**Files:**
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Add failing tests for matchers 5–7**

Append to `cmd/evalbgg/matchers_test.go`:

```go
func TestExactAlwaysEditionRank(t *testing.T) {
	m := exactAlwaysEditionRank{}
	require.Equal(t, "exact-always-edition-rank", m.Name())

	// "Axis & Allies" + "1941" → matches "Axis & Allies: 1941" (ID 4)
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// "Wingspan" + "1st" → no exact match (BGG name is just "Wingspan", not "Wingspan 1st")
	result = m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.Nil(t, result.BGGGame)
}

func TestFuzzyAlwaysEditionRank(t *testing.T) {
	m := fuzzyAlwaysEditionRank{}
	require.Equal(t, "fuzzy-always-edition-rank", m.Name())

	// "Axis & Allies" + "1941" → fuzzy matches "Axis & Allies: 1941" (ID 4) better than others
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestTokenAlwaysEditionRank(t *testing.T) {
	m := tokenAlwaysEditionRank{}
	require.Equal(t, "token-always-edition-rank", m.Name())

	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactAlways|TestFuzzyAlways|TestTokenAlways" -v 2>&1 | head -10
```

Expected: compile error.

- [ ] **Step 3: Append matchers 5–7 to `cmd/evalbgg/matchers.go`**

```go
// --- Matchers 5–7: Always edition signal (System + Edition always concatenated) ---

type exactAlwaysEditionRank struct{}

func (exactAlwaysEditionRank) Name() string { return "exact-always-edition-rank" }
func (exactAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return exactMatch(query, candidates, tiebreakByRank)
}

type fuzzyAlwaysEditionRank struct{}

func (fuzzyAlwaysEditionRank) Name() string { return "fuzzy-always-edition-rank" }
func (fuzzyAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, similarityScore, tiebreakByRank)
}

type tokenAlwaysEditionRank struct{}

func (tokenAlwaysEditionRank) Name() string { return "token-always-edition-rank" }
func (tokenAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, jaccardScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactAlways|TestFuzzyAlways|TestTokenAlways" -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matchers 5-7 (always edition)"
```

---

## Task 7: Matchers 8–11 (smart edition signal)

**Files:**
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Add failing tests for matchers 8–11**

Append to `cmd/evalbgg/matchers_test.go`:

```go
func TestExactSmartEditionRank(t *testing.T) {
	m := exactSmartEditionRank{}
	require.Equal(t, "exact-smart-edition-rank", m.Name())

	// informative edition → uses System + Edition
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// generic edition ("1st") → falls back to System only → matches "Axis & Allies" (ID 6)
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestFuzzySmartEditionRank(t *testing.T) {
	m := fuzzySmartEditionRank{}
	require.Equal(t, "fuzzy-smart-edition-rank", m.Name())

	// informative edition → fuzzy on System + Edition
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestFuzzySmartEditionRated(t *testing.T) {
	m := fuzzySmartEditionRated{}
	require.Equal(t, "fuzzy-smart-edition-rated", m.Name())

	// generic edition → fuzzy on System alone → most rated axis & allies (ID 6, 20000)
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestTokenSmartEditionRank(t *testing.T) {
	m := tokenSmartEditionRank{}
	require.Equal(t, "token-smart-edition-rank", m.Name())

	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactSmart|TestFuzzySmart|TestTokenSmart" -v 2>&1 | head -10
```

- [ ] **Step 3: Append matchers 8–11 to `cmd/evalbgg/matchers.go`**

```go
// smartQuery returns System+Edition if edition is informative, else System alone.
func smartQuery(combo GenConCombo) string {
	if isInformativeEdition(combo.RulesEdition) {
		return normalize(combo.GameSystem + " " + combo.RulesEdition)
	}
	return normalize(combo.GameSystem)
}

// --- Matchers 8–11: Smart edition signal ---

type exactSmartEditionRank struct{}

func (exactSmartEditionRank) Name() string { return "exact-smart-edition-rank" }
func (exactSmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return exactMatch(smartQuery(combo), candidates, tiebreakByRank)
}

type fuzzySmartEditionRank struct{}

func (fuzzySmartEditionRank) Name() string { return "fuzzy-smart-edition-rank" }
func (fuzzySmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, similarityScore, tiebreakByRank)
}

type fuzzySmartEditionRated struct{}

func (fuzzySmartEditionRated) Name() string { return "fuzzy-smart-edition-rated" }
func (fuzzySmartEditionRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, similarityScore, tiebreakByRated)
}

type tokenSmartEditionRank struct{}

func (tokenSmartEditionRank) Name() string { return "token-smart-edition-rank" }
func (tokenSmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, jaccardScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactSmart|TestFuzzySmart|TestTokenSmart" -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matchers 8-11 (smart edition)"
```

---

## Task 8: Matcher 12 (pure title)

**Files:**
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Add failing test for matcher 12**

Append to `cmd/evalbgg/matchers_test.go`:

```go
func TestFuzzyTitleRank(t *testing.T) {
	m := fuzzyTitleRank{}
	require.Equal(t, "fuzzy-title-rank", m.Name())

	// RepTitle "Wingspan" → fuzzy matches "Wingspan" (ID 1)
	result := m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)

	// RepTitle "Axis & Allies 1941 for Beginners" → should find something about axis & allies
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941 for Beginners"}, fixture)
	require.NotNil(t, result.BGGGame)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd cmd/evalbgg && go test ./... -run "TestFuzzyTitle" -v 2>&1 | head -10
```

- [ ] **Step 3: Append matcher 12 to `cmd/evalbgg/matchers.go`**

```go
// --- Matcher 12: Pure title signal ---

type fuzzyTitleRank struct{}

func (fuzzyTitleRank) Name() string { return "fuzzy-title-rank" }
func (fuzzyTitleRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.RepTitle), candidates, similarityScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestFuzzyTitle" -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matcher 12 (pure title)"
```

---

## Task 9: Matchers 13–14 (title-derived always, two-stage)

**Files:**
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Add failing tests for matchers 13–14**

Append to `cmd/evalbgg/matchers_test.go`:

```go
func TestExactTitleDerivedAlwaysRank(t *testing.T) {
	m := exactTitleDerivedAlwaysRank{}
	require.Equal(t, "exact-title-derived-always-rank", m.Name())

	// System filter finds axis & allies entries; derived "1941" matches "axis & allies  1941" (ID 4)
	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "Global 1942",
		RepTitle:     "Axis & Allies 1941 for Beginners",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// No system match → empty
	result = m.Match(GenConCombo{
		GameSystem:   "Completely Unknown Game",
		RulesEdition: "1st",
		RepTitle:     "Some Random Title",
	}, fixture)
	require.Nil(t, result.BGGGame)
}

func TestFuzzyTitleDerivedAlwaysRank(t *testing.T) {
	m := fuzzyTitleDerivedAlwaysRank{}
	require.Equal(t, "fuzzy-title-derived-always-rank", m.Name())

	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies 1941 Championship",
	}, fixture)
	require.NotNil(t, result.BGGGame)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactTitleDerivedAlways|TestFuzzyTitleDerivedAlways" -v 2>&1 | head -10
```

- [ ] **Step 3: Append matchers 13–14 to `cmd/evalbgg/matchers.go`**

```go
// --- Matchers 13–14: Two-stage, title-derived edition always ---

const systemFilterThreshold = 0.5

type exactTitleDerivedAlwaysRank struct{}

func (exactTitleDerivedAlwaysRank) Name() string { return "exact-title-derived-always-rank" }
func (exactTitleDerivedAlwaysRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return MatchResult{}
	}
	return exactMatch(normalize(derived), filtered, tiebreakByRank)
}

type fuzzyTitleDerivedAlwaysRank struct{}

func (fuzzyTitleDerivedAlwaysRank) Name() string { return "fuzzy-title-derived-always-rank" }
func (fuzzyTitleDerivedAlwaysRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(normalize(derived), filtered, similarityScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactTitleDerivedAlways|TestFuzzyTitleDerivedAlways" -v
```

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matchers 13-14 (title-derived always, two-stage)"
```

---

## Task 10: Matchers 15–18 (title-derived smart, two-stage)

**Files:**
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`

- [ ] **Step 1: Add failing tests for matchers 15–18**

Append to `cmd/evalbgg/matchers_test.go`:

```go
func TestExactTitleDerivedSmartRank(t *testing.T) {
	m := exactTitleDerivedSmartRank{}
	require.Equal(t, "exact-title-derived-smart-rank", m.Name())

	// derived "1941" is informative → uses it; filtered axis & allies → ID 4
	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies 1941 for Beginners",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// derived empty/generic → falls back: exact on system among filtered → ID 6 ("Axis & Allies")
	result = m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies Tournament Finals",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestFuzzyTitleDerivedSmartRank(t *testing.T) {
	m := fuzzyTitleDerivedSmartRank{}
	require.Equal(t, "fuzzy-title-derived-smart-rank", m.Name())

	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies 1941 for Beginners",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestFuzzyTitleDerivedSmartRated(t *testing.T) {
	m := fuzzyTitleDerivedSmartRated{}
	require.Equal(t, "fuzzy-title-derived-smart-rated", m.Name())

	// generic derived → fallback fuzzy on system, rated tiebreak → most rated axis entry (ID 6)
	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies Tournament",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestTokenTitleDerivedSmartRank(t *testing.T) {
	m := tokenTitleDerivedSmartRank{}
	require.Equal(t, "token-title-derived-smart-rank", m.Name())

	result := m.Match(GenConCombo{
		GameSystem:   "Axis & Allies",
		RulesEdition: "1st",
		RepTitle:     "Axis & Allies 1941 for Beginners",
	}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}
```

- [ ] **Step 2: Run to verify failure**

```bash
cd cmd/evalbgg && go test ./... -run "TestExactTitleDerivedSmart|TestFuzzyTitleDerivedSmart|TestTokenTitleDerivedSmart" -v 2>&1 | head -10
```

- [ ] **Step 3: Append matchers 15–18 to `cmd/evalbgg/matchers.go`**

```go
// smartTitleDerivedQuery returns the title-derived edition if informative,
// else falls back to the game system alone. Used within the already-filtered candidate set.
func smartTitleDerivedQuery(combo GenConCombo) string {
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived != "" && isInformativeEdition(derived) {
		return normalize(derived)
	}
	return normalize(combo.GameSystem)
}

// --- Matchers 15–18: Two-stage, title-derived edition smart ---

type exactTitleDerivedSmartRank struct{}

func (exactTitleDerivedSmartRank) Name() string { return "exact-title-derived-smart-rank" }
func (exactTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return exactMatch(smartTitleDerivedQuery(combo), filtered, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRank struct{}

func (fuzzyTitleDerivedSmartRank) Name() string { return "fuzzy-title-derived-smart-rank" }
func (fuzzyTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, similarityScore, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRated struct{}

func (fuzzyTitleDerivedSmartRated) Name() string { return "fuzzy-title-derived-smart-rated" }
func (fuzzyTitleDerivedSmartRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, similarityScore, tiebreakByRated)
}

type tokenTitleDerivedSmartRank struct{}

func (tokenTitleDerivedSmartRank) Name() string { return "token-title-derived-smart-rank" }
func (tokenTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, jaccardScore, tiebreakByRank)
}
```

- [ ] **Step 4: Run all matcher tests — expect all pass**

```bash
cd cmd/evalbgg && go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/evalbgg/matchers.go cmd/evalbgg/matchers_test.go
git commit -m "feat(evalbgg): add matchers 15-18 (title-derived smart, two-stage)"
```

---

## Task 11: CSV output writer

**Files:**
- Create: `cmd/evalbgg/output.go`

- [ ] **Step 1: Create `cmd/evalbgg/output.go`**

The output has one row per combo. For each of the 18 matchers there are 3 columns: `{name}_id`, `{name}_name`, `{name}_score`. Exact matchers write `1.00` for score when matched, empty when not. Followed by summary columns and the empty `correct_bgg_id` scoring column.

```go
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// writeCSV writes the full comparison CSV to w.
func writeCSV(w io.Writer, combos []GenConCombo, matchers []Matcher, results [][]MatchResult) error {
	cw := csv.NewWriter(w)

	// Build header
	header := []string{
		"game_system",
		"rules_edition",
		"edition_informative",
		"event_count",
		"representative_title",
	}
	for _, m := range matchers {
		n := m.Name()
		header = append(header, n+"_id", n+"_name", n+"_score")
	}
	header = append(header, "agreement_count", "consensus_id", "consensus_name", "correct_bgg_id")

	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, combo := range combos {
		row := []string{
			combo.GameSystem,
			combo.RulesEdition,
			strconv.FormatBool(isInformativeEdition(combo.RulesEdition)),
			strconv.Itoa(combo.EventCount),
			combo.RepTitle,
		}

		// Count votes for consensus
		votes := make(map[string]int)
		for j, m := range matchers {
			r := results[i][j]
			if r.BGGGame != nil {
				votes[r.BGGGame.ID]++
				_ = m // used for header
			}
		}

		for j := range matchers {
			r := results[i][j]
			if r.BGGGame == nil {
				row = append(row, "", "", "")
			} else {
				row = append(row,
					r.BGGGame.ID,
					r.BGGGame.Name,
					fmt.Sprintf("%.4f", r.Score),
				)
			}
		}

		consensusID, consensusName := consensus(votes, results[i])
		agreementCount := 0
		if consensusID != "" {
			agreementCount = votes[consensusID]
		}

		row = append(row,
			strconv.Itoa(agreementCount),
			consensusID,
			consensusName,
			"", // correct_bgg_id — left blank for manual scoring
		)

		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	cw.Flush()
	return cw.Error()
}

// consensus returns the BGG id and name that the most matchers agreed on.
// Returns empty strings if no matcher returned a result.
func consensus(votes map[string]int, rowResults []MatchResult) (id, name string) {
	var bestID string
	var bestCount int
	for vid, count := range votes {
		if count > bestCount {
			bestID = vid
			bestCount = count
		}
	}
	if bestID == "" {
		return "", ""
	}
	// Find name for bestID from any result
	for _, r := range rowResults {
		if r.BGGGame != nil && r.BGGGame.ID == bestID {
			return bestID, r.BGGGame.Name
		}
	}
	return bestID, ""
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd cmd/evalbgg && go build . 2>&1 || true
```

Expected: fails with "no main" only.

- [ ] **Step 3: Commit**

```bash
git add cmd/evalbgg/output.go
git commit -m "feat(evalbgg): add CSV output writer"
```

---

## Task 12: Main orchestration + smoke test

**Files:**
- Create: `cmd/evalbgg/main.go`

- [ ] **Step 1: Create `cmd/evalbgg/main.go`**

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	genconPath := flag.String("gencon", "", "path to Gen Con events CSV (required)")
	bggPath := flag.String("bgg", "", "path to BGG CSV (required)")
	outputPath := flag.String("output", "bgg_eval.csv", "path for output CSV")
	flag.Parse()

	if *genconPath == "" || *bggPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Println("Loading BGG data...")
	bggGames, err := loadBGG(*bggPath)
	if err != nil {
		log.Fatalf("failed to load BGG CSV: %v", err)
	}
	log.Printf("Loaded %d non-expansion BGG games", len(bggGames))

	log.Println("Loading Gen Con combos...")
	combos, err := loadGenConCombos(*genconPath)
	if err != nil {
		log.Fatalf("failed to load Gen Con CSV: %v", err)
	}
	log.Printf("Loaded %d unique (Game System, Rules Edition) combos", len(combos))

	matchers := []Matcher{
		exactSystemRank{},
		fuzzySystemRank{},
		fuzzySystemRated{},
		tokenSystemRank{},
		exactAlwaysEditionRank{},
		fuzzyAlwaysEditionRank{},
		tokenAlwaysEditionRank{},
		exactSmartEditionRank{},
		fuzzySmartEditionRank{},
		fuzzySmartEditionRated{},
		tokenSmartEditionRank{},
		fuzzyTitleRank{},
		exactTitleDerivedAlwaysRank{},
		fuzzyTitleDerivedAlwaysRank{},
		exactTitleDerivedSmartRank{},
		fuzzyTitleDerivedSmartRank{},
		fuzzyTitleDerivedSmartRated{},
		tokenTitleDerivedSmartRank{},
	}
	log.Printf("Running %d matchers across %d combos...", len(matchers), len(combos))

	// results[i][j] = matcher j's result for combo i
	results := make([][]MatchResult, len(combos))
	for i, combo := range combos {
		results[i] = make([]MatchResult, len(matchers))
		for j, m := range matchers {
			results[i][j] = m.Match(combo, bggGames)
		}
		if (i+1)%100 == 0 {
			log.Printf("  processed %d/%d combos", i+1, len(combos))
		}
	}

	log.Printf("Writing output to %s...", *outputPath)
	f, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	if err := writeCSV(f, combos, matchers, results); err != nil {
		log.Fatalf("failed to write CSV: %v", err)
	}

	fmt.Printf("Done. Output written to %s\n", *outputPath)
}
```

- [ ] **Step 2: Build the binary**

```bash
cd cmd/evalbgg && go build -o evalbgg .
```

Expected: builds successfully with no errors.

- [ ] **Step 3: Run smoke test against the real CSVs**

```bash
cd /home/myasonik/Workspace/Gen-Con-Buddy-API
./cmd/evalbgg/evalbgg \
  --gencon data.csv \
  --bgg boardgames_ranks.csv \
  --output bgg_eval.csv
```

Expected output (approximate):
```
Loading BGG data...
Loaded ~134000 non-expansion BGG games
Loading Gen Con combos...
Loaded ~867 unique (Game System, Rules Edition) combos
Running 18 matchers across 867 combos...
  processed 100/867 combos
  ...
Done. Output written to bgg_eval.csv
```

- [ ] **Step 4: Spot-check the output**

```bash
head -2 bgg_eval.csv | cut -c1-200
wc -l bgg_eval.csv
```

Expected: header row + 867 data rows = 868 lines. Header should start with `game_system,rules_edition,edition_informative,event_count,representative_title,exact-system-rank_id,...`

- [ ] **Step 5: Check a known game appears correctly**

```bash
grep "^Wingspan," bgg_eval.csv | head -5
```

Expected: rows for Wingspan combos with BGG id `266192` (Wingspan's BGG id) appearing in at least the exact-system-rank columns.

- [ ] **Step 6: Run all tests one final time**

```bash
cd cmd/evalbgg && go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add cmd/evalbgg/main.go
# also add the binary if you want it tracked — but typically binaries are gitignored
git commit -m "feat(evalbgg): add main orchestration, smoke test passes"
```

---

## Self-Review Notes

**Spec coverage check:**
- ✅ 18 matchers implemented
- ✅ All matchers exclude expansions (filtered before Match() is called)
- ✅ Normalization: lowercase, strip punctuation, keep `&`
- ✅ Title stopwords defined and used in extractTitleDerived
- ✅ Informative edition check covers all generic terms from spec
- ✅ System filter threshold 0.5 for two-stage matchers (constant `systemFilterThreshold`)
- ✅ Fuzzy and token matchers output raw similarity score
- ✅ Exact matchers output 1.0 when matched, empty when not
- ✅ Output: one row per combo, all 18 matcher columns, agreement_count, consensus, correct_bgg_id
- ✅ Representative title = most common title per combo
- ✅ edition_informative column derived from isInformativeEdition(RulesEdition)
- ✅ Windows-1252 decoding for Gen Con CSV (charmap.Windows1252)

**One gap to watch:** The `filterBySystem` in two-stage matchers uses `similarityScore` (Levenshtein) as the stage-1 filter regardless of which algorithm the matcher uses for stage-2. This is intentional — the filter is always permissive fuzzy, stage-2 is where the algorithm varies.
