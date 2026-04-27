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
