package catalog

import "strings"

// synonyms maps short names gamers actually type to words that appear in
// product names. Trigram search fixes typos ("eldn ring"), but it can't know
// that "gta" means "Grand Theft Auto": those letters aren't in the name.
//
// Keys are single lowercase words; a query is expanded word by word, so
// "ac valhalla" becomes "assassin's creed valhalla".
var synonyms = map[string]string{
	"gta":      "grand theft auto",
	"gta5":     "grand theft auto v",
	"gtav":     "grand theft auto v",
	"cod":      "call of duty",
	"bo6":      "black ops 6",
	"rdr":      "red dead redemption",
	"rdr2":     "red dead redemption 2",
	"bg3":      "baldur's gate 3",
	"ac":       "assassin's creed",
	"re4":      "resident evil 4",
	"tw3":      "the witcher 3",
	"fifa":     "ea sports fc",
	"fc25":     "ea sports fc 25",
	"cp2077":   "cyberpunk 2077",
	"sf6":      "street fighter 6",
	"gow":      "god of war",
	"mk8":      "mario kart 8",
	"acnh":     "animal crossing new horizons",
	"r6":       "rainbow six",
	"civ":      "civilization",
	"civ7":     "civilization vii",
	"ps":       "playstation",
	"psplus":   "playstation plus",
	"xgp":      "xbox game pass",
	"gamepass": "game pass",
}

// expandQuery lowercases q and replaces known short names with full names.
// Words without a synonym are kept as typed.
func expandQuery(q string) string {
	words := strings.Fields(strings.ToLower(q))
	for i, w := range words {
		if full, ok := synonyms[w]; ok {
			words[i] = full
		}
	}
	return strings.Join(words, " ")
}
