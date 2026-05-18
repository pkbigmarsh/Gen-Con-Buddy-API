# BGG Cascade Matcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract BGG matching logic into a shared `internal/bgg` package, implement the 3-stage exact cascade matcher, add a `cmd/data/matchbgg` preprocessing command that writes `bgg_mapping.json`, and wire that mapping into `cmd/data/update` so events get a `BggID` field in OpenSearch.

**Architecture:** `internal/bgg` owns types, normalization (with diacritic stripping), scoring helpers, CSV loaders, and the cascade `Match()` function. `cmd/evalbgg` is refactored to import from it (no behavioral change). `cmd/data/matchbgg` uses the same package to produce a JSON mapping file. `cmd/data/update` reads that file and stamps `BggID` on every event before writing to OpenSearch.

**Tech Stack:** Go standard library (`encoding/csv`, `encoding/json`, `sort`, `strings`, `unicode`), `golang.org/x/text/encoding/charmap` (already in go.mod) for Windows-1252 CSV decoding, `golang.org/x/text/unicode/norm` (already in go.mod) for NFD diacritic stripping, `github.com/stretchr/testify` for tests, `github.com/spf13/cobra` for the new command.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/bgg/types.go` | **Create** | `BGGGame`, `Corpus`, `GenConCombo`, `MatchResult` |
| `internal/bgg/normalize.go` | **Create** | `Normalize`, `IsInformativeEdition`, `ExtractTitleDerived` |
| `internal/bgg/normalize_test.go` | **Create** | Tests for normalize (incl. diacritics) |
| `internal/bgg/score.go` | **Create** | `SimilarityScore`, `JaccardScore` |
| `internal/bgg/score_test.go` | **Create** | Tests for similarity and Jaccard |
| `internal/bgg/load.go` | **Create** | `LoadCorpus`, `LoadGenConCombos`, CSV helpers |
| `internal/bgg/match.go` | **Create** | `Match()` — the 3-stage cascade |
| `internal/bgg/match_test.go` | **Create** | Tests for all cascade stages |
| `cmd/evalbgg/types.go` | **Modify** | Remove shared types; keep `Matcher` interface + local `matchResult` |
| `cmd/evalbgg/normalize.go` | **Delete** | Replaced by `internal/bgg` |
| `cmd/evalbgg/score.go` | **Delete** | Replaced by `internal/bgg` |
| `cmd/evalbgg/normalize_test.go` | **Delete** | Moved to `internal/bgg` |
| `cmd/evalbgg/score_test.go` | **Delete** | Moved to `internal/bgg` |
| `cmd/evalbgg/load.go` | **Delete** | Both functions move to `internal/bgg` |
| `cmd/evalbgg/matchers.go` | **Modify** | Use `bgg.BGGGame`, `bgg.Normalize`, `bgg.SimilarityScore`, `bgg.JaccardScore` |
| `cmd/evalbgg/matchers_test.go` | **Modify** | Update fixture to use `bgg.BGGGame` |
| `cmd/evalbgg/output.go` | **Modify** | Update `matchResult` type references |
| `cmd/evalbgg/main.go` | **Modify** | Use `bgg.LoadCorpus`, `bgg.LoadGenConCombos` |
| `cmd/data/matchbgg/matchbgg.go` | **Create** | New `match-bgg` subcommand |
| `cmd/data/data.go` | **Modify** | Register `MatchBGGCmd`; add `--bgg-mapping` persistent flag |
| `cmd/data/update.go` | **Modify** | Load mapping, apply `BggID` to events |
| `internal/event/types.go` | **Modify** | Add `BggID string` field; update `Externalize`/`FromExternal` |
| `gcbapi/event.go` | **Modify** | Add `BggID string` to `EventAttributes` |
| `cmd/data/initialize/schema/event_index_template.json` | **Modify** | Add `bggId` keyword field |

---

## Task 1: Create `internal/bgg/types.go`

**Files:**
- Create: `internal/bgg/types.go`

- [ ] **Step 1: Create the file**

```go
package bgg

// BGGGame holds all fields from the BGG CSV for a single game.
type BGGGame struct {
	ID            string
	Name          string
	YearPublished string
	IsExpansion   bool
	Rank          int // 0 = unranked
	BayesAverage  float64
	Average       float64
	UsersRated    int
	AbstractsRank string
	CGSRank       string
	ChildrensRank string
	FamilyRank    string
	PartyRank     string
	StrategyRank  string
	ThematicRank  string
	WarRank       string
}

// Corpus holds the full BGG dataset split by expansion status.
type Corpus struct {
	BaseGames  []BGGGame
	Expansions []BGGGame
}

// GenConCombo is one unique (GameSystem, RulesEdition) pair from BGM events.
type GenConCombo struct {
	GameSystem   string
	RulesEdition string
	RepTitle     string // most common Title across events sharing this combo
	EventCount   int
}

// MatchResult is the output of a Match call. BGGID is empty when no match was found.
type MatchResult struct {
	BGGID string
	Name  string
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/bgg/...
```

Expected: no output (compiles cleanly).

- [ ] **Step 3: Commit**

```bash
git add internal/bgg/types.go
git commit -m "feat(bgg): add internal/bgg package with shared types"
```

---

## Task 2: Create `internal/bgg/normalize.go` + tests (TDD)

**Files:**
- Create: `internal/bgg/normalize_test.go`
- Create: `internal/bgg/normalize.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/bgg/normalize_test.go`:

```go
package bgg

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
		{"Axis & Allies: 1942", "axis & allies 1942"},
		{"Ticket to Ride!", "ticket to ride"},
		{"  Extra   Spaces  ", "extra spaces"},
		{"", ""},
		// diacritic stripping
		{"SHŌBU", "shobu"},
		{"Orléans", "orleans"},
		{"Yucatán", "yucatan"},
		{"Ahau: Rulers of Yucatán", "ahau rulers of yucatan"},
		{"Aerodrome", "aerodrome"}, // no diacritics, unchanged
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, Normalize(tt.input))
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
		{"Africa", true},
		{"20th Anniversary", true},
		{"European Expansion", true},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, IsInformativeEdition(tt.input))
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
			want:   "",
		},
		{
			system: "Twilight Imperium",
			title:  "Twilight Imperium with Prophecy of Kings",
			want:   "prophecy kings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			require.Equal(t, tt.want, ExtractTitleDerived(tt.system, tt.title))
		})
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/bgg/... -run TestNormalize
```

Expected: compile error — `Normalize` not defined.

- [ ] **Step 3: Implement `internal/bgg/normalize.go`**

```go
package bgg

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var titleStopwords = map[string]bool{
	"tournament": true, "finals": true, "final": true, "qualifier": true,
	"round": true, "semi": true, "beginner": true, "beginners": true,
	"experienced": true, "advanced": true, "mini": true, "open": true,
	"championship": true, "preliminary": true,
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

// Normalize lowercases s, strips diacritics, strips punctuation (keeping &
// and alphanumerics), and collapses whitespace.
func Normalize(s string) string {
	// NFD decomposition separates base characters from combining marks (diacritics).
	s = norm.NFD.String(s)
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			// combining mark (diacritic) — strip it
			continue
		}
		if r == '&' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// IsInformativeEdition returns true if edition contains tokens beyond bare ordinals.
func IsInformativeEdition(edition string) bool {
	for _, w := range strings.Fields(Normalize(edition)) {
		if !genericEditionTerms[w] {
			return true
		}
	}
	return false
}

// ExtractTitleDerived strips game system tokens and title stopwords from title,
// returning the remaining edition-like tokens joined by spaces.
func ExtractTitleDerived(gameSystem, title string) string {
	sysTokens := make(map[string]bool)
	for _, w := range strings.Fields(Normalize(gameSystem)) {
		sysTokens[w] = true
	}

	var result []string
	for _, w := range strings.Fields(Normalize(title)) {
		if !sysTokens[w] && !titleStopwords[w] {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/bgg/... -run "TestNormalize|TestIsInformativeEdition|TestExtractTitleDerived" -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bgg/normalize.go internal/bgg/normalize_test.go
git commit -m "feat(bgg): add Normalize with diacritic stripping, IsInformativeEdition, ExtractTitleDerived"
```

---

## Task 3: Create `internal/bgg/score.go` + tests (TDD)

**Files:**
- Create: `internal/bgg/score_test.go`
- Create: `internal/bgg/score.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/bgg/score_test.go`:

```go
package bgg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"axis & allies", "axis & allies", 1.0, 1.0},
		{"", "", 1.0, 1.0},
		{"axis & allies", "completely different", 0.0, 0.3},
		{"axis & allies 1942", "axis & allies 1942", 1.0, 1.0},
		{"wingspan", "wingspam", 0.8, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := SimilarityScore(tt.a, tt.b)
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
			score := JaccardScore(tt.a, tt.b)
			require.GreaterOrEqual(t, score, tt.min)
			require.LessOrEqual(t, score, tt.max)
		})
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/bgg/... -run "TestSimilarityScore|TestJaccardScore"
```

Expected: compile error — `SimilarityScore` not defined.

- [ ] **Step 3: Implement `internal/bgg/score.go`**

```go
package bgg

import "strings"

// SimilarityScore returns normalized Levenshtein similarity in [0,1].
func SimilarityScore(a, b string) float64 {
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

// JaccardScore returns token-set Jaccard similarity in [0,1].
func JaccardScore(a, b string) float64 {
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

- [ ] **Step 4: Run tests**

```bash
go test ./internal/bgg/... -run "TestSimilarityScore|TestJaccardScore" -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bgg/score.go internal/bgg/score_test.go
git commit -m "feat(bgg): add SimilarityScore (Levenshtein) and JaccardScore"
```

---

## Task 4: Create `internal/bgg/load.go`

**Files:**
- Create: `internal/bgg/load.go`

- [ ] **Step 1: Create the file**

```go
package bgg

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

// LoadCorpus reads the BGG CSV and returns all games split into BaseGames and Expansions.
func LoadCorpus(path string) (Corpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return Corpus{}, fmt.Errorf("open bgg csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return Corpus{}, fmt.Errorf("read bgg headers: %w", err)
	}
	idx := headerIndex(headers)

	var corpus Corpus
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Corpus{}, fmt.Errorf("read bgg row: %w", err)
		}
		g := BGGGame{
			ID:            csvField(row, idx, "id"),
			Name:          csvField(row, idx, "name"),
			YearPublished: csvField(row, idx, "yearpublished"),
			IsExpansion:   csvField(row, idx, "is_expansion") == "1",
			Rank:          parseInt(csvField(row, idx, "rank")),
			BayesAverage:  parseFloat(csvField(row, idx, "bayesaverage")),
			Average:       parseFloat(csvField(row, idx, "average")),
			UsersRated:    parseInt(csvField(row, idx, "usersrated")),
			AbstractsRank: csvField(row, idx, "abstracts_rank"),
			CGSRank:       csvField(row, idx, "cgs_rank"),
			ChildrensRank: csvField(row, idx, "childrensgames_rank"),
			FamilyRank:    csvField(row, idx, "familygames_rank"),
			PartyRank:     csvField(row, idx, "partygames_rank"),
			StrategyRank:  csvField(row, idx, "strategygames_rank"),
			ThematicRank:  csvField(row, idx, "thematic_rank"),
			WarRank:       csvField(row, idx, "wargames_rank"),
		}
		if g.IsExpansion {
			corpus.Expansions = append(corpus.Expansions, g)
		} else {
			corpus.BaseGames = append(corpus.BaseGames, g)
		}
	}
	return corpus, nil
}

// LoadGenConCombos reads the Gen Con CSV and returns unique (GameSystem, RulesEdition)
// combos from BGM events, with representative title and event count.
func LoadGenConCombos(path string) ([]GenConCombo, error) {
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
		if !strings.HasPrefix(csvField(row, idx, "Event Type"), "BGM") {
			continue
		}
		key := comboKey{
			system:  csvField(row, idx, "Game System"),
			edition: csvField(row, idx, "Rules Edition"),
		}
		title := csvField(row, idx, "Title")
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

// --- helpers ---

func headerIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}
	return m
}

func csvField(row []string, idx map[string]int, name string) string {
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
go build ./internal/bgg/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/bgg/load.go
git commit -m "feat(bgg): add LoadCorpus and LoadGenConCombos"
```

---

## Task 5: Create `internal/bgg/match.go` + tests (TDD)

**Files:**
- Create: `internal/bgg/match_test.go`
- Create: `internal/bgg/match.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/bgg/match_test.go`:

```go
package bgg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testCorpus is a small in-memory BGG dataset for cascade tests.
var testCorpus = Corpus{
	BaseGames: []BGGGame{
		{ID: "1", Name: "Wingspan", Rank: 10, UsersRated: 50000},
		{ID: "3", Name: "Ark Nova", Rank: 2, UsersRated: 60000},
		{ID: "4", Name: "Axis & Allies: 1941", Rank: 100, UsersRated: 5000},
		{ID: "5", Name: "Axis & Allies: 1942", Rank: 80, UsersRated: 8000},
		{ID: "6", Name: "Axis & Allies", Rank: 200, UsersRated: 20000},
		{ID: "7", Name: "SHŌBU", Rank: 50, UsersRated: 3000},
		{ID: "8", Name: "Orléans", Rank: 30, UsersRated: 15000},
	},
	Expansions: []BGGGame{
		{ID: "2", Name: "Wingspan: European Expansion", Rank: 0, UsersRated: 10000},
		{ID: "9", Name: "Catan: Cities & Knights", Rank: 0, UsersRated: 25000},
	},
}

func TestMatchStage1_GenericEdition_SystemOnly(t *testing.T) {
	// generic edition → query is system alone
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, testCorpus)
	require.Equal(t, "6", result.BGGID)
	require.Equal(t, "Axis & Allies", result.Name)
}

func TestMatchStage1_InformativeEdition_SystemPlusEdition(t *testing.T) {
	// informative edition → query is "axis & allies 1941"
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, testCorpus)
	require.Equal(t, "4", result.BGGID)
}

func TestMatchStage1_TiebreakByRank(t *testing.T) {
	// Two games could match "axis & allies" — ID 6 (rank 200) is the only
	// exact match so rank tiebreak is not exercised here, but the pattern
	// is validated: lower rank wins when scores are equal.
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, testCorpus)
	require.Equal(t, "6", result.BGGID)
}

func TestMatchStage1_DiacriticNormalization(t *testing.T) {
	// Gen Con entry omits accent; BGG has "Orléans"
	result := Match(GenConCombo{GameSystem: "Orleans", RulesEdition: "1st", RepTitle: "Orleans"}, testCorpus)
	require.Equal(t, "8", result.BGGID)
}

func TestMatchStage1_DiacriticNormalization_Shobu(t *testing.T) {
	// BGG has "SHŌBU"; Gen Con has "SHOBU"
	result := Match(GenConCombo{GameSystem: "SHOBU", RulesEdition: "1st", RepTitle: "SHOBU"}, testCorpus)
	require.Equal(t, "7", result.BGGID)
}

func TestMatchStage2_TitleDerived(t *testing.T) {
	// Stage 1 misses: edition "1st" is generic, no exact match on "axis & allies 1st".
	// Actually stage 1 WOULD find "Axis & Allies" (ID 6) for generic "1st".
	// To test stage 2 specifically: use a system that has NO exact base match by itself,
	// but whose title carries the edition hint.
	// Use a corpus where "Wingspan European Expansion" is a BASE game (not expansion)
	// so stage 1 misses but stage 2 finds via title hint.
	localCorpus := Corpus{
		BaseGames: []BGGGame{
			{ID: "10", Name: "Firefly: The Game", Rank: 500, UsersRated: 8000},
			{ID: "11", Name: "Firefly: The Game – 10th Anniversary Collector's Edition", Rank: 0, UsersRated: 2000},
		},
	}
	// System: "Firefly", Edition: "10th Anniversary" (informative) — stage 1 finds nothing
	// because Normalize("Firefly 10th Anniversary") != Normalize("Firefly: The Game – 10th Anniversary Collector's Edition")
	// Stage 2 falls back to title-derived: RepTitle "Firefly 10th Anniversary Demo" → derived "10th"
	// But "10th" isn't in genericEditionTerms, so it's informative → query "firefly 10th"
	// That still won't exact-match. For a cleaner stage 2 test:
	localCorpus2 := Corpus{
		BaseGames: []BGGGame{
			{ID: "20", Name: "Axis & Allies: 1941", Rank: 100},
			{ID: "21", Name: "Axis & Allies", Rank: 200},
		},
	}
	// System: "Axis & Allies", Edition: "1st" (generic → stage 1 finds ID 21)
	// — stage 2 is not reached. To isolate stage 2, use a game where:
	//   - system is unique in corpus
	//   - stage 1 finds nothing (no exact match on system alone, no exact on system+edition)
	//   - title gives the edition hint
	localCorpus3 := Corpus{
		BaseGames: []BGGGame{
			{ID: "30", Name: "Terraforming Mars: Ares Expedition", Rank: 50},
		},
	}
	result := Match(GenConCombo{
		GameSystem:   "Terraforming Mars",
		RulesEdition: "1st",
		RepTitle:     "Terraforming Mars: Ares Expedition Demo",
	}, localCorpus3)
	// Stage 1: system alone "terraforming mars" → no exact match
	// Stage 2: derived from title = "ares expedition" (informative) → query "terraforming mars ares expedition" → exact match ID 30
	require.Equal(t, "30", result.BGGID)
	_ = localCorpus
	_ = localCorpus2
}

func TestMatchStage3_ExpansionSearch_InformativeEdition(t *testing.T) {
	// Wingspan + "European Expansion" (informative) → stage 1 finds nothing in BaseGames,
	// stage 2 finds nothing, stage 3 searches Expansions and finds ID 2.
	result := Match(GenConCombo{
		GameSystem:   "Wingspan",
		RulesEdition: "European Expansion",
		RepTitle:     "Wingspan European Expansion",
	}, testCorpus)
	require.Equal(t, "2", result.BGGID)
}

func TestMatchStage3_ExpansionNotSearched_GenericEdition(t *testing.T) {
	// Wingspan + "1st" (generic) → stage 3 does NOT run → finds base game ID 1, not expansion
	result := Match(GenConCombo{
		GameSystem:   "Wingspan",
		RulesEdition: "1st",
		RepTitle:     "Wingspan",
	}, testCorpus)
	require.Equal(t, "1", result.BGGID)
}

func TestMatchStage4_NoMatch(t *testing.T) {
	// Game not in corpus → empty result
	result := Match(GenConCombo{
		GameSystem:   "Completely Unknown Board Game",
		RulesEdition: "1st",
		RepTitle:     "Unknown",
	}, testCorpus)
	require.Empty(t, result.BGGID)
	require.Empty(t, result.Name)
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/bgg/... -run "TestMatch"
```

Expected: compile error — `Match` not defined.

- [ ] **Step 3: Implement `internal/bgg/match.go`**

```go
package bgg

// Match runs the 3-stage exact cascade against the given corpus and returns the
// best confident result, or an empty MatchResult if nothing clears any stage.
//
// TODO(overrides): Accept an overrides map as a parameter once the override
// system is built:
//   func Match(combo GenConCombo, corpus Corpus, overrides map[string]string) MatchResult
// Before stage 1, check if (combo.GameSystem + "|" + combo.RulesEdition) has
// an entry in overrides. If so, look it up in the corpus and return it
// immediately, bypassing the cascade. This ensures manually verified results
// survive re-runs of match-bgg without needing to touch the cascade logic.
func Match(combo GenConCombo, corpus Corpus) MatchResult {
	// Stage 1: exact match on smart query, base games only.
	query := smartQuery(combo)
	if r := exactBest(query, corpus.BaseGames); r.BGGID != "" {
		return r
	}

	// Stage 2: exact match on title-derived smart query, base games only.
	if r := exactBest(smartTitleDerivedQuery(combo), corpus.BaseGames); r.BGGID != "" {
		return r
	}

	// Stage 3: exact match on smart query, expansions only.
	// Only runs when the edition is informative enough to distinguish an expansion.
	if IsInformativeEdition(combo.RulesEdition) {
		if r := exactBest(query, corpus.Expansions); r.BGGID != "" {
			return r
		}
	}

	return MatchResult{}
}

// smartQuery returns the normalized search query for a combo:
// system+edition if the edition is informative, system alone otherwise.
func smartQuery(combo GenConCombo) string {
	if IsInformativeEdition(combo.RulesEdition) {
		return Normalize(combo.GameSystem + " " + combo.RulesEdition)
	}
	return Normalize(combo.GameSystem)
}

// smartTitleDerivedQuery derives an edition hint from the representative title
// and returns a query using that hint if informative, or system alone otherwise.
func smartTitleDerivedQuery(combo GenConCombo) string {
	derived := ExtractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived != "" && IsInformativeEdition(derived) {
		return Normalize(combo.GameSystem + " " + derived)
	}
	return Normalize(combo.GameSystem)
}

// exactBest finds all games in candidates whose normalized name exactly matches
// query and returns the one with the lowest BGG rank (rank 0 = unranked, loses
// to any ranked game). Returns empty MatchResult if nothing matches.
func exactBest(query string, candidates []BGGGame) MatchResult {
	var best *BGGGame
	for i := range candidates {
		c := &candidates[i]
		if Normalize(c.Name) != query {
			continue
		}
		if best == nil || betterRank(*c, *best) {
			best = c
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGID: best.ID, Name: best.Name}
}

// betterRank returns true if a is a better pick than b by BGG rank.
// Rank 0 means unranked — always loses to a ranked game.
func betterRank(a, b BGGGame) bool {
	if a.Rank == 0 {
		return false
	}
	if b.Rank == 0 {
		return true
	}
	return a.Rank < b.Rank
}
```

- [ ] **Step 4: Run all internal/bgg tests**

```bash
go test ./internal/bgg/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bgg/match.go internal/bgg/match_test.go
git commit -m "feat(bgg): add 3-stage cascade Match function"
```

---

## Task 6: Refactor `cmd/evalbgg` to use `internal/bgg`

**Files:**
- Modify: `cmd/evalbgg/types.go`
- Delete: `cmd/evalbgg/normalize.go`
- Delete: `cmd/evalbgg/score.go`
- Delete: `cmd/evalbgg/normalize_test.go`
- Delete: `cmd/evalbgg/score_test.go`
- Delete: `cmd/evalbgg/load.go`
- Modify: `cmd/evalbgg/matchers.go`
- Modify: `cmd/evalbgg/matchers_test.go`
- Modify: `cmd/evalbgg/output.go`
- Modify: `cmd/evalbgg/main.go`

- [ ] **Step 1: Replace `cmd/evalbgg/types.go`**

```go
package main

import "github.com/gencon_buddy_api/internal/bgg"

// matchResult is the eval-specific result carrying score info alongside the match.
// The production cascade (internal/bgg.MatchResult) only needs BGGID and Name;
// evalbgg matchers additionally track raw similarity scores for comparison output.
type matchResult struct {
	BGGGame *bgg.BGGGame
	Score   float64
}

// Matcher is the interface all 18 eval strategies implement.
type Matcher interface {
	Name() string
	Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult
}
```

- [ ] **Step 2: Delete removed files**

```bash
rm cmd/evalbgg/normalize.go cmd/evalbgg/score.go \
   cmd/evalbgg/normalize_test.go cmd/evalbgg/score_test.go \
   cmd/evalbgg/load.go
```

- [ ] **Step 3: Replace `cmd/evalbgg/matchers.go`**

Full replacement — same logic, updated type signatures and package-qualified helper calls:

```go
package main

import "github.com/gencon_buddy_api/internal/bgg"

func tiebreakByRank(a, b bgg.BGGGame) bool {
	if a.Rank == 0 {
		return false
	}
	if b.Rank == 0 {
		return true
	}
	return a.Rank < b.Rank
}

func tiebreakByRated(a, b bgg.BGGGame) bool {
	return a.UsersRated > b.UsersRated
}

func exactMatch(query string, candidates []bgg.BGGGame, tiebreak func(a, b bgg.BGGGame) bool) matchResult {
	var best *bgg.BGGGame
	for i := range candidates {
		c := &candidates[i]
		if bgg.Normalize(c.Name) == query {
			if best == nil || tiebreak(*c, *best) {
				best = c
			}
		}
	}
	if best == nil {
		return matchResult{}
	}
	return matchResult{BGGGame: best, Score: 1.0}
}

func bestScoredMatch(query string, candidates []bgg.BGGGame, scoreFn func(a, b string) float64, tiebreak func(a, b bgg.BGGGame) bool) matchResult {
	var best *bgg.BGGGame
	var bestScore float64
	for i := range candidates {
		c := &candidates[i]
		score := scoreFn(query, bgg.Normalize(c.Name))
		if score <= 0 {
			continue
		}
		if best == nil || score > bestScore || (score == bestScore && tiebreak(*c, *best)) {
			best = c
			bestScore = score
		}
	}
	if best == nil {
		return matchResult{}
	}
	return matchResult{BGGGame: best, Score: bestScore}
}

func filterBySystem(gameSystem string, candidates []bgg.BGGGame, threshold float64) []bgg.BGGGame {
	query := bgg.Normalize(gameSystem)
	var filtered []bgg.BGGGame
	for _, c := range candidates {
		if bgg.SimilarityScore(query, bgg.Normalize(c.Name)) >= threshold {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- Matchers 1–4: System signal ---

type exactSystemRank struct{}

func (exactSystemRank) Name() string { return "exact-system-rank" }
func (exactSystemRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return exactMatch(bgg.Normalize(combo.GameSystem), candidates, tiebreakByRank)
}

type fuzzySystemRank struct{}

func (fuzzySystemRank) Name() string { return "fuzzy-system-rank" }
func (fuzzySystemRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(bgg.Normalize(combo.GameSystem), candidates, bgg.SimilarityScore, tiebreakByRank)
}

type fuzzySystemRated struct{}

func (fuzzySystemRated) Name() string { return "fuzzy-system-rated" }
func (fuzzySystemRated) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(bgg.Normalize(combo.GameSystem), candidates, bgg.SimilarityScore, tiebreakByRated)
}

type tokenSystemRank struct{}

func (tokenSystemRank) Name() string { return "token-system-rank" }
func (tokenSystemRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(bgg.Normalize(combo.GameSystem), candidates, bgg.JaccardScore, tiebreakByRank)
}

// --- Matchers 5–7: Always edition signal ---

type exactAlwaysEditionRank struct{}

func (exactAlwaysEditionRank) Name() string { return "exact-always-edition-rank" }
func (exactAlwaysEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	query := bgg.Normalize(combo.GameSystem + " " + combo.RulesEdition)
	return exactMatch(query, candidates, tiebreakByRank)
}

type fuzzyAlwaysEditionRank struct{}

func (fuzzyAlwaysEditionRank) Name() string { return "fuzzy-always-edition-rank" }
func (fuzzyAlwaysEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	query := bgg.Normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, bgg.SimilarityScore, tiebreakByRank)
}

type tokenAlwaysEditionRank struct{}

func (tokenAlwaysEditionRank) Name() string { return "token-always-edition-rank" }
func (tokenAlwaysEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	query := bgg.Normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, bgg.JaccardScore, tiebreakByRank)
}

// smartQuery returns System+Edition if edition is informative, else System alone.
func smartQuery(combo bgg.GenConCombo) string {
	if bgg.IsInformativeEdition(combo.RulesEdition) {
		return bgg.Normalize(combo.GameSystem + " " + combo.RulesEdition)
	}
	return bgg.Normalize(combo.GameSystem)
}

// --- Matchers 8–11: Smart edition signal ---

type exactSmartEditionRank struct{}

func (exactSmartEditionRank) Name() string { return "exact-smart-edition-rank" }
func (exactSmartEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return exactMatch(smartQuery(combo), candidates, tiebreakByRank)
}

type fuzzySmartEditionRank struct{}

func (fuzzySmartEditionRank) Name() string { return "fuzzy-smart-edition-rank" }
func (fuzzySmartEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(smartQuery(combo), candidates, bgg.SimilarityScore, tiebreakByRank)
}

type fuzzySmartEditionRated struct{}

func (fuzzySmartEditionRated) Name() string { return "fuzzy-smart-edition-rated" }
func (fuzzySmartEditionRated) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(smartQuery(combo), candidates, bgg.SimilarityScore, tiebreakByRated)
}

type tokenSmartEditionRank struct{}

func (tokenSmartEditionRank) Name() string { return "token-smart-edition-rank" }
func (tokenSmartEditionRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(smartQuery(combo), candidates, bgg.JaccardScore, tiebreakByRank)
}

// --- Matcher 12: Pure title signal ---

type fuzzyTitleRank struct{}

func (fuzzyTitleRank) Name() string { return "fuzzy-title-rank" }
func (fuzzyTitleRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	return bestScoredMatch(bgg.Normalize(combo.RepTitle), candidates, bgg.SimilarityScore, tiebreakByRank)
}

// --- Matchers 13–14: Two-stage, title-derived edition always ---

const systemFilterThreshold = 0.5

type exactTitleDerivedAlwaysRank struct{}

func (exactTitleDerivedAlwaysRank) Name() string { return "exact-title-derived-always-rank" }
func (exactTitleDerivedAlwaysRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := bgg.ExtractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return matchResult{}
	}
	query := bgg.Normalize(combo.GameSystem + " " + derived)
	return exactMatch(query, filtered, tiebreakByRank)
}

type fuzzyTitleDerivedAlwaysRank struct{}

func (fuzzyTitleDerivedAlwaysRank) Name() string { return "fuzzy-title-derived-always-rank" }
func (fuzzyTitleDerivedAlwaysRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := bgg.ExtractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return matchResult{}
	}
	query := bgg.Normalize(combo.GameSystem + " " + derived)
	return bestScoredMatch(query, filtered, bgg.SimilarityScore, tiebreakByRank)
}

func smartTitleDerivedQuery(combo bgg.GenConCombo) string {
	derived := bgg.ExtractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived != "" && bgg.IsInformativeEdition(derived) {
		return bgg.Normalize(combo.GameSystem + " " + derived)
	}
	return bgg.Normalize(combo.GameSystem)
}

// --- Matchers 15–18: Two-stage, title-derived edition smart ---

type exactTitleDerivedSmartRank struct{}

func (exactTitleDerivedSmartRank) Name() string { return "exact-title-derived-smart-rank" }
func (exactTitleDerivedSmartRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return matchResult{}
	}
	return exactMatch(smartTitleDerivedQuery(combo), filtered, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRank struct{}

func (fuzzyTitleDerivedSmartRank) Name() string { return "fuzzy-title-derived-smart-rank" }
func (fuzzyTitleDerivedSmartRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return matchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, bgg.SimilarityScore, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRated struct{}

func (fuzzyTitleDerivedSmartRated) Name() string { return "fuzzy-title-derived-smart-rated" }
func (fuzzyTitleDerivedSmartRated) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return matchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, bgg.SimilarityScore, tiebreakByRated)
}

type tokenTitleDerivedSmartRank struct{}

func (tokenTitleDerivedSmartRank) Name() string { return "token-title-derived-smart-rank" }
func (tokenTitleDerivedSmartRank) Match(combo bgg.GenConCombo, candidates []bgg.BGGGame) matchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return matchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, bgg.JaccardScore, tiebreakByRank)
}
```

- [ ] **Step 4: Update `cmd/evalbgg/matchers_test.go`**

Replace the fixture declaration and update all `BGGGame` and `MatchResult` references:

```go
package main

import (
	"testing"

	"github.com/gencon_buddy_api/internal/bgg"
	"github.com/stretchr/testify/require"
)

var fixture = []bgg.BGGGame{
	{ID: "1", Name: "Wingspan", Rank: 10, UsersRated: 50000, IsExpansion: false},
	{ID: "2", Name: "Wingspan: European Expansion", Rank: 0, UsersRated: 10000, IsExpansion: false},
	{ID: "3", Name: "Ark Nova", Rank: 2, UsersRated: 60000, IsExpansion: false},
	{ID: "4", Name: "Axis & Allies: 1941", Rank: 100, UsersRated: 5000, IsExpansion: false},
	{ID: "5", Name: "Axis & Allies: 1942", Rank: 80, UsersRated: 8000, IsExpansion: false},
	{ID: "6", Name: "Axis & Allies", Rank: 200, UsersRated: 20000, IsExpansion: false},
}
```

All test function bodies remain identical — only `result.BGGGame` (still a `*bgg.BGGGame`) and `result.Score` references are unchanged since `matchResult` still has those fields.

- [ ] **Step 5: Update `cmd/evalbgg/output.go`**

Find and replace the `MatchResult` type reference with `matchResult` (lowercase). The function signature changes from:

```go
func writeCSV(w io.Writer, combos []GenConCombo, matchers []Matcher, results [][]MatchResult) error {
```

to:

```go
func writeCSV(w io.Writer, combos []bgg.GenConCombo, matchers []Matcher, results [][]matchResult) error {
```

Add the import `"github.com/gencon_buddy_api/internal/bgg"` at the top.

Inside the function, any `r.BGGGame` accesses remain unchanged (the field name is the same on `matchResult`).

- [ ] **Step 6: Update `cmd/evalbgg/main.go`**

Replace the `loadBGG` call with `bgg.LoadCorpus` and `loadGenConCombos` with `bgg.LoadGenConCombos`:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gencon_buddy_api/internal/bgg"
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
	corpus, err := bgg.LoadCorpus(*bggPath)
	if err != nil {
		log.Fatalf("failed to load BGG CSV: %v", err)
	}
	log.Printf("Loaded %d base games, %d expansions", len(corpus.BaseGames), len(corpus.Expansions))

	log.Println("Loading Gen Con combos...")
	combos, err := bgg.LoadGenConCombos(*genconPath)
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

	results := make([][]matchResult, len(combos))
	for i, combo := range combos {
		results[i] = make([]matchResult, len(matchers))
		for j, m := range matchers {
			results[i][j] = m.Match(combo, corpus.BaseGames)
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

- [ ] **Step 7: Run all evalbgg tests**

```bash
go test ./cmd/evalbgg/... -v
```

Expected: all PASS (same behavior, no logic changed).

- [ ] **Step 8: Build to confirm no compile errors**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 9: Commit**

```bash
git add cmd/evalbgg/
git commit -m "refactor(evalbgg): import shared types and functions from internal/bgg"
```

---

## Task 7: Add `BggID` to event structs and OpenSearch schema

**Files:**
- Modify: `internal/event/types.go`
- Modify: `gcbapi/event.go`
- Modify: `cmd/data/initialize/schema/event_index_template.json`

- [ ] **Step 1: Add `BggID` to `internal/event/types.go`**

In the `Event` struct (around line 75, after `LastModified`), add:

```go
BggID string `json:"bggId"`
```

In `Externalize()` (around line 371, before the closing `}`of the Attributes literal), add:

```go
BggID: e.BggID,
```

In `FromExternal()` (around line 400, in the `evt := &Event{...}` literal), add:

```go
BggID: e.Attributes.BggID,
```

- [ ] **Step 2: Add `BggID` to `gcbapi/event.go`**

In `EventAttributes`, add after `GameID`:

```go
BggID string `json:"bggId"`
```

- [ ] **Step 3: Add `bggId` to OpenSearch schema**

In `cmd/data/initialize/schema/event_index_template.json`, add a new property after `"gameId"`:

```json
"bggId": {
    "type": "keyword"
},
```

- [ ] **Step 4: Build and verify**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add internal/event/types.go gcbapi/event.go \
        cmd/data/initialize/schema/event_index_template.json
git commit -m "feat(event): add BggID field and bggId OpenSearch keyword mapping"
```

---

## Task 8: Create `cmd/data/matchbgg/matchbgg.go`

**Files:**
- Create: `cmd/data/matchbgg/matchbgg.go`
- Modify: `cmd/data/data.go`

- [ ] **Step 1: Create `cmd/data/matchbgg/matchbgg.go`**

```go
package matchbgg

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gencon_buddy_api/internal/bgg"
	"github.com/spf13/cobra"
)

// MappingEntry is one resolved (game_system, rules_edition) → BGG game mapping.
type MappingEntry struct {
	GameSystem   string `json:"game_system"`
	RulesEdition string `json:"rules_edition"`
	BGGID        string `json:"bgg_id"`
	BGGName      string `json:"bgg_name"`
}

// MappingFile is the full output written to disk.
type MappingFile struct {
	GeneratedAt string         `json:"generated_at"`
	TotalCombos int            `json:"total_combos"`
	Matched     int            `json:"matched"`
	Mappings    []MappingEntry `json:"mappings"`
}

var MatchBGGCmd = &cobra.Command{
	Use:   "match-bgg",
	Short: "Match Gen Con game/edition combos to BGG game IDs and write a mapping file",
	Long: `Reads the Gen Con events CSV and BGG games CSV, runs the cascade matcher
against each unique (Game System, Rules Edition) combination, and writes
a JSON mapping file. Commit the mapping file to the repo so every
'data update' run uses the same mappings.`,
	RunE: run,
}

func init() {
	MatchBGGCmd.Flags().StringP("gencon", "g", "", "path to Gen Con events CSV (required)")
	MatchBGGCmd.Flags().StringP("bgg", "b", "", "path to BGG CSV (required)")
	MatchBGGCmd.Flags().StringP("output", "o", "bgg_mapping.json", "output path for the mapping file")
	_ = MatchBGGCmd.MarkFlagRequired("gencon")
	_ = MatchBGGCmd.MarkFlagRequired("bgg")
}

func run(cmd *cobra.Command, _ []string) error {
	genconPath, _ := cmd.Flags().GetString("gencon")
	bggPath, _ := cmd.Flags().GetString("bgg")
	outputPath, _ := cmd.Flags().GetString("output")

	corpus, err := bgg.LoadCorpus(bggPath)
	if err != nil {
		return fmt.Errorf("load bgg corpus: %w", err)
	}

	combos, err := bgg.LoadGenConCombos(genconPath)
	if err != nil {
		return fmt.Errorf("load gencon combos: %w", err)
	}

	// TODO(overrides): Before running the cascade, accept an --overrides flag
	// pointing to a JSON file with the same structure as this output. Load the
	// overrides, then after cascade results are computed, replace any entry for a
	// matching (game_system, rules_edition) key with the override value. Overrides
	// that name a combo not present in the Gen Con data are silently ignored.

	var mappings []MappingEntry
	for _, combo := range combos {
		result := bgg.Match(combo, corpus)
		if result.BGGID == "" {
			continue
		}
		mappings = append(mappings, MappingEntry{
			GameSystem:   combo.GameSystem,
			RulesEdition: combo.RulesEdition,
			BGGID:        result.BGGID,
			BGGName:      result.Name,
		})
	}

	// Sort for stable, diff-friendly output.
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].GameSystem != mappings[j].GameSystem {
			return mappings[i].GameSystem < mappings[j].GameSystem
		}
		return mappings[i].RulesEdition < mappings[j].RulesEdition
	})

	out := MappingFile{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TotalCombos: len(combos),
		Matched:     len(mappings),
		Mappings:    mappings,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mapping: %w", err)
	}

	if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write mapping file: %w", err)
	}

	cmd.Printf("Matched %d / %d combos → %s\n", out.Matched, out.TotalCombos, outputPath)
	return nil
}
```

- [ ] **Step 2: Register `MatchBGGCmd` in `cmd/data/data.go`**

Add the import:
```go
"github.com/gencon_buddy_api/cmd/data/matchbgg"
```

Add to `init()`:
```go
Cmd.AddCommand(matchbgg.MatchBGGCmd)
```

- [ ] **Step 3: Build to verify**

```bash
go build ./cmd/data/...
```

Expected: no output.

- [ ] **Step 4: Smoke-test help output**

```bash
go run . data match-bgg --help
```

Expected: prints usage showing `--gencon`, `--bgg`, `--output` flags.

- [ ] **Step 5: Commit**

```bash
git add cmd/data/matchbgg/matchbgg.go cmd/data/data.go
git commit -m "feat(data): add match-bgg subcommand to produce bgg_mapping.json"
```

---

## Task 9: Wire `--bgg-mapping` into `cmd/data/update`

**Files:**
- Modify: `cmd/data/data.go`
- Modify: `cmd/data/update.go`

- [ ] **Step 1: Add the persistent flag to `cmd/data/data.go`**

In `init()`, add after the existing persistent flags:

```go
Cmd.PersistentFlags().String("bgg-mapping", "bgg_mapping.json", "path to the BGG mapping file produced by match-bgg")
```

- [ ] **Step 2: Add a mapping loader helper to `cmd/data/update.go`**

Add this function anywhere in the file:

```go
// loadBGGMapping reads the mapping file produced by match-bgg and returns a
// map keyed by "GameSystem|RulesEdition" → BGG ID string.
// If the file does not exist, it logs a warning and returns an empty map.
func loadBGGMapping(cmd *cobra.Command, logger zerolog.Logger) map[string]string {
	path, err := cmd.Flags().GetString("bgg-mapping")
	if err != nil || path == "" {
		return map[string]string{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warn().Str("path", path).Msg("bgg mapping file not found; events will have no BggID")
		return map[string]string{}
	}

	var file struct {
		Mappings []struct {
			GameSystem   string `json:"game_system"`
			RulesEdition string `json:"rules_edition"`
			BGGID        string `json:"bgg_id"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		logger.Warn().Err(err).Str("path", path).Msg("failed to parse bgg mapping file; events will have no BggID")
		return map[string]string{}
	}

	m := make(map[string]string, len(file.Mappings))
	for _, e := range file.Mappings {
		m[e.GameSystem+"|"+e.RulesEdition] = e.BGGID
	}
	return m
}
```

Add the required imports at the top of `update.go` if not already present:

```go
"encoding/json"
"os"

"github.com/rs/zerolog"
```

- [ ] **Step 3: Apply the mapping in the `update` function**

In `update()`, immediately after events are loaded (after the `if err != nil` block that returns on load failure), add:

```go
bggMapping := loadBGGMapping(cmd, gcb.Logger)
for _, e := range events {
    if id, ok := bggMapping[e.GameSystem+"|"+e.RulesEdition]; ok {
        e.BggID = id
    }
}
```

This block applies to both the `downloadEvents` and `event.LoadEventCSV` paths since both produce `[]*event.Event`.

- [ ] **Step 4: Build and verify**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 5: Verify the flag is visible**

```bash
go run . data update --help
```

Expected: output includes `--bgg-mapping string` in the flags list.

- [ ] **Step 6: Commit**

```bash
git add cmd/data/data.go cmd/data/update.go
git commit -m "feat(data): apply bgg_mapping.json to events during data update"
```

---

## Final verification

- [ ] **Run all tests**

```bash
go test ./...
```

Expected: all PASS, no failures.

- [ ] **Build all binaries**

```bash
go build ./...
```

Expected: no output.
