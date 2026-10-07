package boxes

import "strings"

// quotePair holds the open and close quotes of each nesting level.
type quotePair struct{ open, close []string }

// defaultQuotes are the quotes of languages that have no entry in
// langQuotes.
var defaultQuotes = quotePair{[]string{"“", "‘"}, []string{"”", "’"}}

// langQuotes are the quotes of the languages of the Unicode CLDR, in the
// order they are looked up by prefix: a language matches the last entry
// that it starts with.
var langQuotes = []struct {
	lang   string
	quotes quotePair
}{
	{"ab", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"agq", quotePair{[]string{"„", "‚"}, []string{"”", "’"}}},
	{"am", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"an", quotePair{[]string{"«", "”"}, []string{"»", "”"}}},
	{"ar", quotePair{[]string{"”", "’"}, []string{"“", "‘"}}},
	{"ast", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"az_Arab", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"az_Cyrl", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"bas", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"be", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"bg", quotePair{[]string{"„"}, []string{"“"}}},
	{"blo", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"bm", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"br", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"bs", quotePair{[]string{"„", "‘"}, []string{"”", "’"}}},
	{"bs_Cyrl", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"ca", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"co", quotePair{[]string{"«"}, []string{"»"}}},
	{"cs", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"cu", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"cv", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"de", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"dsb", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"dua", quotePair{[]string{"«", "‘"}, []string{"»", "’"}}},
	{"dyo", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"el", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"el_POLYTON", quotePair{[]string{"«", "‘"}, []string{"»", "’"}}},
	{"es_US", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"et", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"eu", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"ewo", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"fa", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"ff", quotePair{[]string{"„", "‚"}, []string{"”", "’"}}},
	{"fi", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"fr", quotePair{[]string{"«"}, []string{"»"}}},
	{"fr_CA", quotePair{[]string{"«", "”"}, []string{"»", "“"}}},
	{"fr_CH", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"fur", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"gsw", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"he", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"hr", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"hsb", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"hu", quotePair{[]string{"„", "»"}, []string{"”", "«"}}},
	{"hy", quotePair{[]string{"«"}, []string{"»"}}},
	{"ia", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"ie", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"is", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"it", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"it_CH", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"ja", quotePair{[]string{"「", "『"}, []string{"」", "』"}}},
	{"jgo", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"ka", quotePair{[]string{"„", "«"}, []string{"“", "»"}}},
	{"kab", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"kk", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"kkj", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"kl", quotePair{[]string{"»", "›"}, []string{"«", "‹"}}},
	{"ksf", quotePair{[]string{"«", "‘"}, []string{"»", "’"}}},
	{"ksh", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"ky", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"lag", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"lb", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"lij", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"lt", quotePair{[]string{"„"}, []string{"“"}}},
	{"luy", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"mg", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"mk", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"ms_Arab", quotePair{[]string{"”", "’"}, []string{"“", "‘"}}},
	{"mua", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"mzn", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"nds", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"nl", quotePair{[]string{"‘"}, []string{"’"}}},
	{"nmg", quotePair{[]string{"„", "«"}, []string{"”", "»"}}},
	{"nnh", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"no", quotePair{[]string{"«", "‘"}, []string{"»", "’"}}},
	{"nr", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"nso", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"oc", quotePair{[]string{"«"}, []string{"»"}}},
	{"oc_ES", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"os", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"pl", quotePair{[]string{"„", "«"}, []string{"”", "»"}}},
	{"prg", quotePair{[]string{"„"}, []string{"“"}}},
	{"pt_PT", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"rm", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"rn", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"ro", quotePair{[]string{"„", "«"}, []string{"”", "»"}}},
	{"ru", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"rw", quotePair{[]string{"«", "‘"}, []string{"»", "’"}}},
	{"sah", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"sc", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"sdh", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"se", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"sg", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"shi", quotePair{[]string{"«", "„"}, []string{"»", "”"}}},
	{"sk", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"sl", quotePair{[]string{"„", "‚"}, []string{"“", "‘"}}},
	{"sn", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"sq", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"sr", quotePair{[]string{"„", "‘"}, []string{"“", "‘"}}},
	{"sr_Latn", quotePair{[]string{"„", "‘"}, []string{"“", "‘"}}},
	{"ss", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"st", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"sv", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"syr", quotePair{[]string{"”", "’"}, []string{"“", "‘"}}},
	{"szl", quotePair{[]string{"„", "»"}, []string{"”", "«"}}},
	{"tg", quotePair{[]string{"»", "‘"}, []string{"«", "’"}}},
	{"ti", quotePair{[]string{"«", "“"}, []string{"»", "”"}}},
	{"ti_ER", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"tk", quotePair{[]string{"“"}, []string{"”"}}},
	{"tn", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"ts", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"ug", quotePair{[]string{"»", "›"}, []string{"«", "‹"}}},
	{"uk", quotePair{[]string{"«", "„"}, []string{"»", "“"}}},
	{"ur", quotePair{[]string{"”", "’"}, []string{"“", "‘"}}},
	{"uz", quotePair{[]string{"“", "’"}, []string{"”", "‘"}}},
	{"ve", quotePair{[]string{"‘", "“"}, []string{"’", "”"}}},
	{"wae", quotePair{[]string{"«", "‹"}, []string{"»", "›"}}},
	{"yav", quotePair{[]string{"«"}, []string{"»"}}},
	{"yi", quotePair{[]string{"”", "’"}, []string{"”", "’"}}},
	{"yue", quotePair{[]string{"「", "『"}, []string{"」", "』"}}},
	{"zgh", quotePair{[]string{"«", "„"}, []string{"»", "”"}}},
	{"zh_Hant", quotePair{[]string{"「", "『"}, []string{"」", "』"}}},
}

// quotesForLang returns the quotes used for a language when quotes is auto
// (CSS Generated Content 3 §2.2.2).
func quotesForLang(lang string) quotePair {
	for _, q := range langQuotes {
		if q.lang == lang {
			return q.quotes
		}
	}
	for i := len(langQuotes) - 1; i >= 0; i-- {
		if q := langQuotes[i]; strings.HasPrefix(lang, q.lang) {
			return q.quotes
		}
	}
	return defaultQuotes
}
