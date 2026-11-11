package identify

import (
	"regexp"
	"sort"
	"strings"
)

// Movie editions
//
// Plex shows the versions of a movie as editions when the name has
// {edition-Name}: "Blade Runner (1982) {edition-Final Cut}". The edition
// comes from an explicit {edition-...} tag (Radarr, Plex) or from the usual
// release words after the year (Extended, Director's Cut, ...). Without a
// year only the explicit tag is used: before the year the same words can be
// part of the title ("The Final Cut (2004)").

var editionTagRegex = regexp.MustCompile(`(?i)\{edition-([^}]+)\}`)

// Release words and their edition name, matched on the name with dots and
// underscores turned into spaces
var editionWords = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`(?i)\bdirector'?s cut\b`), "Director's Cut"},
	{regexp.MustCompile(`(?i)\bextended(?: (?:cut|edition|version))?\b`), "Extended"},
	{regexp.MustCompile(`(?i)\btheatrical(?: (?:cut|edition|version))?\b`), "Theatrical"},
	{regexp.MustCompile(`(?i)\bunrated\b`), "Unrated"},
	{regexp.MustCompile(`(?i)\buncut\b`), "Uncut"},
	{regexp.MustCompile(`(?i)\bfinal cut\b`), "Final Cut"},
	{regexp.MustCompile(`(?i)\bultimate (?:cut|edition)\b`), "Ultimate Edition"},
	{regexp.MustCompile(`(?i)\bspecial edition\b`), "Special Edition"},
	{regexp.MustCompile(`(?i)\bcollector'?s edition\b`), "Collector's Edition"},
	{regexp.MustCompile(`(?i)\b(?:[0-9]+(?:th)? )?anniversary edition\b`), "Anniversary Edition"},
	{regexp.MustCompile(`(?i)\bremastered\b`), "Remastered"},
	{regexp.MustCompile(`(?i)\bimax\b`), "IMAX"},
	{regexp.MustCompile(`(?i)\bcriterion(?: collection)?\b`), "Criterion"},
	{regexp.MustCompile(`(?i)\bredux\b`), "Redux"},
	{regexp.MustCompile(`(?i)\bopen matte\b`), "Open Matte"},
}

// stripEditionTags removes {edition-...} from a name
func stripEditionTags(name string) string {
	return editionTagRegex.ReplaceAllString(name, " ")
}

// movieEdition returns the edition of a movie file name, or ""
func movieEdition(filename, year string) string {
	if m := editionTagRegex.FindStringSubmatch(filename); m != nil {
		return strings.TrimSpace(m[1])
	}
	if year == "" {
		return ""
	}
	name := strings.NewReplacer(".", " ", "_", " ").Replace(filename)
	i := strings.Index(name, year)
	if i < 0 {
		return ""
	}
	after := name[i+len(year):]

	// In the order they appear: "Extended Remastered"
	type found struct {
		pos  int
		name string
	}
	var hits []found
	for _, w := range editionWords {
		if loc := w.re.FindStringIndex(after); loc != nil {
			hits = append(hits, found{loc[0], w.name})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].pos < hits[j].pos })
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		names = append(names, h.name)
	}
	return strings.Join(names, " ")
}
