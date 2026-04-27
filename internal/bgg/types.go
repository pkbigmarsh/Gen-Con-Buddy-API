package bgg

// BGGGame holds all fields from the BGG CSV for a single game.
type BGGGame struct {
	ID             string
	Name           string
	NormalizedName string // Normalize(Name), pre-computed at load time
	YearPublished  string
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
