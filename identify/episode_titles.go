package identify

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	log "github.com/sirupsen/logrus"
)

// Episode titles in the file name
//
// Release names often carry the episode title(s) after SxxEyy, e.g.
//
//	Taz-Mania - S01E02 - Like Father, Like Son + Frights of Passage HDTV-720p.mkv
//
// When the numbering of the release (often TheTVDB) differs from TMDB, the
// titles are a better guide than the number: here they are TMDB S01E02-E03.
// The titles are compared with every episode of the show on TMDB, in the
// olaris language and in English, with the same scoring as match_title in
// identify-episode.sh (coronas-scripts).

// Score thresholds, as in identify-episode.sh
const (
	scoreExact    = 1.0
	scoreContains = 0.9
	scoreJaccard  = 0.85 // multiplier of the shared-words ratio
	// A match is certain when it scores at least scoreCertain and leads the
	// second best episode by scoreMargin, or when it is the only exact one.
	scoreCertain = 0.85
	scoreMargin  = 0.15
	// Below this a title is not similar to any episode: no evidence either way
	scoreSimilar = 0.6
	scoreEpsilon = 1e-9
)

// TitleStatus is the outcome of comparing the titles in the file name with TMDB
type TitleStatus int

const (
	// TitlesNone: no titles in the file name, or nothing to compare them with
	TitlesNone TitleStatus = iota
	// TitlesKeep: the titles do not contradict SxxEyy, keep the file numbering
	TitlesKeep
	// TitlesResolved: every title found its TMDB episode, use them
	TitlesResolved
	// TitlesReview: titles and numbering disagree and cannot be reconciled
	TitlesReview
)

func (s TitleStatus) String() string {
	switch s {
	case TitlesKeep:
		return "keep"
	case TitlesResolved:
		return "resolved"
	case TitlesReview:
		return "review"
	}
	return "none"
}

// tmdbEpisode is one TMDB episode with its title in the olaris language and in English
type tmdbEpisode struct {
	Season, Episode int
	Name, NameEN    string
	norm, normEN    string
}

func newTmdbEpisode(season, episode int, name, nameEN string) tmdbEpisode {
	e := tmdbEpisode{Season: season, Episode: episode, Name: name, NameEN: nameEN}
	e.norm = normalizeTitle(name)
	e.normEN = normalizeTitle(nameEN)
	// "Episodio 10" & co. are TMDB placeholders, not titles
	if placeholderTitle.MatchString(e.norm) {
		e.norm = ""
	}
	if placeholderTitle.MatchString(e.normEN) {
		e.normEN = ""
	}
	return e
}

func (e tmdbEpisode) code() string {
	return fmt.Sprintf("S%02dE%02d", e.Season, e.Episode)
}

// TitleMatch is the TMDB episode chosen for one title of the file name
type TitleMatch struct {
	Title   string  `json:"title"`
	Season  int     `json:"season"`
	Episode int     `json:"episode"`
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Second  float64 `json:"second"`
	Certain bool    `json:"certain"`
}

// TitleResult is the outcome of the comparison, see resolveTitles
type TitleResult struct {
	Status  TitleStatus
	Season  int
	Start   int
	End     int
	Names   []string // TMDB names of Start..End (olaris language)
	Matches []TitleMatch
	Reason  string
}

var (
	// SxxEyy (with optional -Ezz/Ezz) or NxNN: the episode titles follow it
	episodeMarker = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(s\d{1,4}e\d{1,4}(?:-?e\d{1,4})*(?:-\d{1,4})?|\d{1,2}x\d{2,4}(?:-\d{1,4})?)(?:[^a-z0-9]|$)`)
	// First technical token after the titles (quality, resolution, codec, ...)
	releaseTail  = regexp.MustCompile(`(?i)(?:^|[\s\[\(._-])(?:\d{3,4}[pi]|[hps]dtv|hdtv\w*|web[-. ]?(?:dl|rip)\w*|web[-. ]\d{3,4}p|blu-?ray\w*|bd-?rip|br-?rip|dvd(?:rip|scr|r)?|hdrip|remux|raw-hd|x26[45]|h\.?26[45]|hevc|xvid|divx|proper|repack|aac\w*|ac3|e-?ac-?3|dd[p+]?\d\.?\d|10bit)(?:[\s\]\)._-]|$)`)
	languageTail = regexp.MustCompile(`(?:^|[\s\[\(._-])(?:iTA|ITA|ENG|SUBITA|Sub[- ]?ITA)(?:[\s\]\)._-]|$)`)
	// Separators between the titles of a multi-segment file. " & " also
	// appears inside titles ("War & Pieces"): see matchPiece.
	titleSeparator   = regexp.MustCompile(`\s+\+\s+`)
	ampersand        = regexp.MustCompile(`\s+&\s+`)
	nonAlnum         = regexp.MustCompile(`[^a-z0-9]+`)
	placeholderTitle = regexp.MustCompile(`^(?:episodio|episode|puntata|folge|capitulo|episodio n)\s*\d+$`)
	accentRemover    = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
)

// episodeTitlesFromFilename returns the episode titles written after SxxEyy,
// without quality, resolution and release group, split on " + ".
// "Show - S04E12 - HDTV-720p" has none.
func episodeTitlesFromFilename(filename string) []string {
	loc := episodeMarker.FindStringSubmatchIndex(filename)
	if loc == nil {
		return nil
	}
	rest := filename[loc[3]:]
	// Scene names use dots or underscores instead of spaces
	if !strings.Contains(rest, " ") {
		rest = strings.NewReplacer(".", " ", "_", " ").Replace(rest)
	}
	if m := releaseTail.FindStringIndex(rest); m != nil {
		rest = rest[:m[0]]
	}
	if m := languageTail.FindStringIndex(rest); m != nil {
		rest = rest[:m[0]]
	}
	rest = strings.Trim(rest, " -_.[(")
	if rest == "" {
		return nil
	}

	var titles []string
	for _, t := range titleSeparator.Split(rest, -1) {
		t = strings.Trim(t, " -_.")
		// "TBA", "Episode 5": the title was not known yet
		if n := normalizeTitle(t); n != "" && n != "tba" && n != "tbd" && !placeholderTitle.MatchString(n) {
			titles = append(titles, t)
		}
	}
	return titles
}

// normalizeTitle: lowercase ASCII words only, no accents or punctuation
func normalizeTitle(s string) string {
	s, _, err := transform.String(accentRemover, s)
	if err != nil {
		return ""
	}
	s = nonAlnum.ReplaceAllString(strings.ToLower(s), " ")
	return strings.TrimSpace(s)
}

// longWords: the set of words of 3+ letters of a normalized title
func longWords(s string) map[string]bool {
	words := map[string]bool{}
	for _, w := range strings.Fields(s) {
		if len(w) > 2 {
			words[w] = true
		}
	}
	return words
}

// titleScore: 1 exact, 0.9 one contains the other (TMDB sometimes joins
// segments as "A / B"), otherwise shared words (Jaccard, 3+ letters) * 0.85
func titleScore(q, name string) float64 {
	if q == "" || name == "" {
		return 0
	}
	if q == name {
		return scoreExact
	}
	if len(q) >= 4 && len(name) >= 4 && (strings.Contains(name, q) || strings.Contains(q, name)) {
		return scoreContains
	}
	qw, nw := longWords(q), longWords(name)
	inter, union := 0, len(qw)
	for w := range nw {
		if qw[w] {
			inter++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return scoreJaccard * float64(inter) / float64(union)
}

func episodeScore(q string, e tmdbEpisode) float64 {
	s := titleScore(q, e.norm)
	if t := titleScore(q, e.normEN); t > s {
		s = t
	}
	return s
}

func isCertain(best, second float64) bool {
	return (best >= scoreCertain-scoreEpsilon && best-second >= scoreMargin-scoreEpsilon) ||
		(best >= scoreExact-scoreEpsilon && second < scoreExact-scoreEpsilon)
}

// bestEpisode finds the best TMDB episode for one title and the score of the runner-up
func bestEpisode(title string, episodes []tmdbEpisode) (TitleMatch, bool) {
	q := normalizeTitle(title)
	m := TitleMatch{Title: title}
	found := false
	for _, e := range episodes {
		s := episodeScore(q, e)
		if s > m.Score {
			m.Second = m.Score
			m.Score = s
			m.Season, m.Episode, m.Name = e.Season, e.Episode, e.Name
			found = true
		} else if s > m.Second {
			m.Second = s
		}
	}
	m.Certain = found && isCertain(m.Score, m.Second)
	return m, found
}

// matchPiece matches one " + " piece. A piece with " & " can be one title
// ("War & Pieces") or several ("A & B"): every way of grouping its parts is
// tried, and the one where all groups are certain wins, with the fewest
// groups. Otherwise the whole piece is matched as a single title.
func matchPiece(piece string, episodes []tmdbEpisode) []TitleMatch {
	parts := ampersand.Split(piece, -1)
	if len(parts) > 1 && len(parts) <= 8 {
		var best []TitleMatch
		bestTotal := 0.0
		for mask := 0; mask < 1<<(len(parts)-1); mask++ {
			var groups []TitleMatch
			total, start, ok := 0.0, 0, true
			for i := 1; i <= len(parts) && ok; i++ {
				// bit i-1 set: split between parts[i-1] and parts[i]
				if i == len(parts) || mask&(1<<(i-1)) != 0 {
					m, _ := bestEpisode(strings.Join(parts[start:i], " & "), episodes)
					ok = m.Certain
					groups = append(groups, m)
					total += m.Score
					start = i
				}
			}
			if ok && (best == nil || len(groups) < len(best) || (len(groups) == len(best) && total > bestTotal)) {
				best, bestTotal = groups, total
			}
		}
		if best != nil {
			return best
		}
	}
	m, _ := bestEpisode(piece, episodes)
	return []TitleMatch{m}
}

// resolveTitles compares the titles of the file name with the TMDB episodes.
// season/start/end is the numbering of the file name (SxxEyy[-Ezz]).
func resolveTitles(titles []string, episodes []tmdbEpisode, season, start, end int) TitleResult {
	res := TitleResult{Status: TitlesNone, Season: season, Start: start, End: end}
	if len(titles) == 0 || len(episodes) == 0 {
		return res
	}

	for _, t := range titles {
		res.Matches = append(res.Matches, matchPiece(t, episodes)...)
	}

	allCertain := true
	for _, m := range res.Matches {
		allCertain = allCertain && m.Certain
	}

	if allCertain {
		// Distinct episodes (TMDB may hold two segments in one episode)
		seen := map[[2]int]bool{}
		var eps [][2]int
		for _, m := range res.Matches {
			k := [2]int{m.Season, m.Episode}
			if !seen[k] {
				seen[k] = true
				eps = append(eps, k)
			}
		}
		sort.Slice(eps, func(i, j int) bool {
			if eps[i][0] != eps[j][0] {
				return eps[i][0] < eps[j][0]
			}
			return eps[i][1] < eps[j][1]
		})
		consecutive := true
		for i := 1; i < len(eps); i++ {
			consecutive = consecutive && eps[i][0] == eps[0][0] && eps[i][1] == eps[i-1][1]+1
		}
		if consecutive {
			res.Status = TitlesResolved
			res.Season, res.Start, res.End = eps[0][0], eps[0][1], eps[len(eps)-1][1]
			byKey := map[[2]int]string{}
			for _, e := range episodes {
				byKey[[2]int{e.Season, e.Episode}] = e.Name
			}
			for _, k := range eps {
				res.Names = append(res.Names, byKey[k])
			}
			return res
		}
		res.Status = TitlesReview
		res.Reason = fmt.Sprintf("titoli → %s: episodi non consecutivi o di stagioni diverse, nome file → %s", matchCodes(res.Matches), rangeCode(season, start, end))
		return res
	}

	// Not every title is certain: keep the file numbering unless a title
	// points somewhere else.
	var fileEps []tmdbEpisode
	for _, e := range episodes {
		if e.Season == season && e.Episode >= start && e.Episode <= end {
			fileEps = append(fileEps, e)
		}
	}
	inFile := func(m TitleMatch) bool {
		return m.Season == season && m.Episode >= start && m.Episode <= end
	}

	unknown := 0
	var contradictions []string
	for _, m := range res.Matches {
		agree := 0.0
		q := normalizeTitle(m.Title)
		for _, e := range fileEps {
			if s := episodeScore(q, e); s > agree {
				agree = s
			}
		}
		switch {
		case m.Certain && inFile(m):
		case m.Certain:
			contradictions = append(contradictions, fmt.Sprintf("%q → S%02dE%02d", m.Title, m.Season, m.Episode))
		case agree >= scoreSimilar-scoreEpsilon:
		case m.Score >= scoreSimilar-scoreEpsilon:
			contradictions = append(contradictions, fmt.Sprintf("%q → forse S%02dE%02d", m.Title, m.Season, m.Episode))
		default:
			unknown++
		}
	}

	switch {
	case len(contradictions) > 0:
		res.Status = TitlesReview
		res.Reason = fmt.Sprintf("nome file %s ma %s", rangeCode(season, start, end), strings.Join(contradictions, ", "))
	case unknown > 0 && len(res.Matches) > end-start+1:
		res.Status = TitlesReview
		res.Reason = fmt.Sprintf("%d titoli nel nome file per %d episodi (%s), %d non trovati su TMDB", len(res.Matches), end-start+1, rangeCode(season, start, end), unknown)
	default:
		res.Status = TitlesKeep
	}
	return res
}

func rangeCode(season, start, end int) string {
	if end > start {
		return fmt.Sprintf("S%02dE%02d-E%02d", season, start, end)
	}
	return fmt.Sprintf("S%02dE%02d", season, start)
}

func matchCodes(ms []TitleMatch) string {
	codes := make([]string, len(ms))
	for i, m := range ms {
		codes[i] = fmt.Sprintf("S%02dE%02d", m.Season, m.Episode)
	}
	return strings.Join(codes, " + ")
}

// checkEpisodeTitles compares the titles in the file name with the episodes
// of the show and applies the outcome: TitlesResolved replaces season,
// episode and episode name; TitlesReview marks the file for review and keeps
// the file numbering; otherwise nothing changes.
func (p *ParsedFile) checkEpisodeTitles(showID, season int, ep *EpisodeInfo) TitleResult {
	titles := episodeTitlesFromFilename(p.Filename)
	if len(titles) == 0 {
		return TitleResult{Status: TitlesNone}
	}

	episodes, err := showEpisodes(showID, p.Options.TMDBLanguage)
	if err != nil {
		log.WithFields(log.Fields{"show": showID, "error": err}).Warnln("Could not fetch the TMDB episode list, not checking the episode titles")
		return TitleResult{Status: TitlesNone}
	}

	res := resolveTitles(titles, episodes, season, ep.Start, ep.End)
	if p.hasYearAsSeason && res.Status != TitlesResolved {
		// The year is translated to a season later: only certain titles help
		return TitleResult{Status: TitlesNone}
	}

	p.TitleCheck = res.Status.String()
	p.TitleMatches = res.Matches
	fields := log.Fields{"titles": titles, "file": rangeCode(season, ep.Start, ep.End), "status": p.TitleCheck}
	for i, m := range res.Matches {
		fields[fmt.Sprintf("match%d", i+1)] = fmt.Sprintf("S%02dE%02d %q score %.2f second %.2f certain %v", m.Season, m.Episode, m.Name, m.Score, m.Second, m.Certain)
	}

	switch res.Status {
	case TitlesResolved:
		if res.Season != season || res.Start != ep.Start || res.End != ep.End {
			p.FileSeason, p.FileEpisode = p.Season, p.Episode
			fields["tmdb"] = rangeCode(res.Season, res.Start, res.End)
			log.WithFields(fields).Infoln("The episode titles in the file name point to other TMDB episodes, renumbering.")
		} else {
			log.WithFields(fields).Debugln("The episode titles in the file name match its numbering.")
		}
		p.Season = fmt.Sprintf("%02d", res.Season)
		if res.End > res.Start {
			p.Episode = fmt.Sprintf("%02d-%02d", res.Start, res.End)
		} else {
			p.Episode = fmt.Sprintf("%02d", res.Start)
		}
		var names []string
		for _, n := range res.Names {
			if n != "" {
				names = append(names, n)
			}
		}
		p.EpisodeName = strings.Join(names, " & ")
		p.hasYearAsSeason = false
	case TitlesReview:
		p.NeedsReview = true
		p.ReviewReason = res.Reason
		log.WithFields(fields).Warnln("Episode titles and numbering of the file name disagree, the file needs a review: " + res.Reason)
	default:
		log.WithFields(fields).Debugln("The episode titles in the file name do not contradict its numbering.")
	}
	return res
}

// --- TMDB episode list ------------------------------------------------------

var (
	showEpisodesCache = map[string][]tmdbEpisode{}
	showEpisodesMutex sync.Mutex
)

// showEpisodes returns every episode of every season of a show, with titles
// in language and in English. Cached per run: a folder of a show costs one
// download.
func showEpisodes(showID int, language string) ([]tmdbEpisode, error) {
	key := fmt.Sprintf("%d/%s", showID, language)
	showEpisodesMutex.Lock()
	defer showEpisodesMutex.Unlock()
	if eps, ok := showEpisodesCache[key]; ok {
		return eps, nil
	}

	agent := initAgent()
	info, err := agent.GetTvInfo(showID, map[string]string{"language": language})
	if err != nil {
		return nil, err
	}

	var eps []tmdbEpisode
	for _, s := range info.Seasons {
		local, err := agent.GetTvSeasonInfo(showID, s.SeasonNumber, map[string]string{"language": language})
		if err != nil {
			log.WithFields(log.Fields{"show": showID, "season": s.SeasonNumber, "error": err}).Warnln("Could not fetch TMDB season")
			continue
		}
		english := map[int]string{}
		if !strings.HasPrefix(strings.ToLower(language), "en") {
			if en, err := agent.GetTvSeasonInfo(showID, s.SeasonNumber, map[string]string{"language": "en-US"}); err == nil {
				for _, e := range en.Episodes {
					english[e.EpisodeNumber] = e.Name
				}
			}
		}
		for _, e := range local.Episodes {
			nameEN, ok := english[e.EpisodeNumber]
			if !ok {
				nameEN = e.Name
			}
			eps = append(eps, newTmdbEpisode(s.SeasonNumber, e.EpisodeNumber, e.Name, nameEN))
		}
	}

	showEpisodesCache[key] = eps
	return eps, nil
}
