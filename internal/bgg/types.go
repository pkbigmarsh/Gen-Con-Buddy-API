package bgg

import "fmt"

// BGGGame holds all fields from the BGG CSV for a single game.
type BGGGame struct {
	ID             string
	Name           string
	NormalizedName string // Normalize(Name), pre-computed at load time
	YearPublished  string
	IsExpansion    bool
	Rank           int // 0 = unranked
	BayesAverage   float64
	Average        float64
	UsersRated     int
	AbstractsRank  string
	CGSRank        string
	ChildrensRank  string
	FamilyRank     string
	PartyRank      string
	StrategyRank   string
	ThematicRank   string
	WarRank        string
}

// Corpus holds the full BGG dataset split by expansion status.
type Corpus struct {
	BaseGames  []BGGGame
	Expansions []BGGGame
}

// GenConCombo is one unique (GameSystem, RulesEdition) pair from BGG-eligible events
// (board games, RPGs, miniature games, and card games).
type GenConCombo struct {
	GameSystem   string
	RulesEdition string
	RepTitle     string // most common Title across events sharing this combo
	EventCount   int
}

// MappingKey generates the expected key for [MappingFile] based on the format
// "GameSystem|RulesEdition"
func (g GenConCombo) MappingKey() string {
	return fmt.Sprintf("%s|%s", g.GameSystem, g.RulesEdition)
}

// MatchResult is the output of a Match call. BGGID is empty when no match was found.
type MatchResult struct {
	BGGID string
	Name  string
	Game  *BGGGame
}

// MappingEntry holds the BGG match for one (GameSystem, RulesEdition) pair.
type MappingEntry struct {
	BGGID        string  `json:"bgg_id"`
	BGGName      string  `json:"bgg_name"` // human-readable label for the mapping file
	BGGRank      int     `json:"bgg_rank,omitempty"`
	BGGAvgRating float64 `json:"bgg_avg_rating,omitempty"`
}

// MappingFile is the bgg_mapping.json format.
// Mappings maps "GameSystem|RulesEdition" → MappingEntry.
type MappingFile struct {
	GeneratedAt string                  `json:"generated_at"`
	TotalCombos int                     `json:"total_combos"`
	Matched     int                     `json:"matched"`
	Mappings    map[string]MappingEntry `json:"mappings"`
}
