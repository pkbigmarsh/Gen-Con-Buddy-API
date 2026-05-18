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
