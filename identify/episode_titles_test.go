package identify

import (
	"bufio"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func skipTMDBInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test that calls the live TMDB API in short mode")
	}
}

// Taz-Mania (TMDB 10126), every episode in Italian and English, as of 2026-11
func loadTazMania(t *testing.T) []tmdbEpisode {
	t.Helper()
	f, err := os.Open("testdata/tmdb-10126.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var eps []tmdbEpisode
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		s, _ := strconv.Atoi(c[0])
		e, _ := strconv.Atoi(c[1])
		eps = append(eps, newTmdbEpisode(s, e, c[2], c[3]))
	}
	return eps
}

func TestEpisodeTitlesFromFilename(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"Taz-Mania - S01E02 - Like Father, Like Son + Frights of Passage HDTV-720p", []string{"Like Father, Like Son", "Frights of Passage"}},
		{"Taz-Mania - S02E05 - Tazmanian Lullaby + Deer Taz + A Taz-Manian Moment HDTV-720p", []string{"Tazmanian Lullaby", "Deer Taz", "A Taz-Manian Moment"}},
		{"Taz-Mania - S04E09 - We'll Always Have Taz-Mania + Moments You've Missed HDTV-720p", []string{"We'll Always Have Taz-Mania", "Moments You've Missed"}},
		{"Taz-Mania - S04E12 - HDTV-720p", nil},
		{"Serie - S01E05 - Titolo giusto", []string{"Titolo giusto"}},
		{"Serie - S01E05-E06 - Uno + Due [WEBDL-1080p]", []string{"Uno", "Due"}},
		{"Show.S01E02.Like.Father.Like.Son.720p.HDTV.x264-GRP", []string{"Like Father Like Son"}},
		{"Show.S01E02.720p.HDTV.x264-GRP", nil},
		{"Show S01E02 iTA ENG 1080p", nil},
		{"Show 1x02 - Il titolo - DVDRip", []string{"Il titolo"}},
		{"Show - S01E04 - War & Pieces Bluray-1080p", []string{"War & Pieces"}},
		{"Movie (2009) 1080p", nil},
		{"Show - S03E05 - TBA WEBDL-1080p", nil},
		{"Show - S03E05 - Episode 5 WEBDL-1080p", nil},
		{"Show - S03E05-E06 - Episodio 5 + Il vero titolo", []string{"Il vero titolo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := episodeTitlesFromFilename(tt.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("episodeTitlesFromFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := map[string]string{
		"We'll Always Have Taz-Mania": "we ll always have taz mania",
		"Ci sarà sempre Tazmania":     "ci sara sempre tazmania",
		"  Perché?  Così! ":           "perche cosi",
		"Èlite & Ñandú":               "elite nandu",
	}
	for in, want := range tests {
		if got := normalizeTitle(in); got != want {
			t.Errorf("normalizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTitleScore(t *testing.T) {
	if s := titleScore("deer taz", "deer taz"); s != scoreExact {
		t.Errorf("exact = %v", s)
	}
	if s := titleScore("here kitty kitty kitty", "here kitty kitty kitty part 2"); s != scoreContains {
		t.Errorf("contains = %v", s)
	}
	// shared words of 3+ letters: {son, father, day} vs {like, father, son} → 2/4
	if s := titleScore("son father day", "like father like son"); s < 0.4 || s > 0.5 {
		t.Errorf("jaccard = %v", s)
	}
	if s := titleScore("", "anything"); s != 0 {
		t.Errorf("empty = %v", s)
	}
}

func TestResolveTitlesTazMania(t *testing.T) {
	eps := loadTazMania(t)
	tests := []struct {
		file                      string
		status                    TitleStatus
		season, start, end        int
		names                     []string
		fileSeason, fileEp, fileE int
	}{
		{"Taz-Mania - S01E02 - Like Father, Like Son + Frights of Passage HDTV-720p", TitlesResolved, 1, 2, 3, []string{"Tale padre, tale figlio", "Una sfida impossibile"}, 1, 2, 2},
		{"Taz-Mania - S02E05 - Tazmanian Lullaby + Deer Taz + A Taz-Manian Moment HDTV-720p", TitlesResolved, 2, 8, 10, []string{"Ninnananna tazmaniana", "Cerbiatto caro", "Episodio 10"}, 2, 5, 5},
		{"Taz-Mania - S04E09 - We'll Always Have Taz-Mania + Moments You've Missed HDTV-720p", TitlesResolved, 4, 14, 15, []string{"Ci sarà sempre Tazmania", "Tagliati dalla fretta"}, 4, 9, 9},
		// Italian titles work as well
		{"Tazmania - S04E09 - Ci sarà sempre Tazmania + Tagliati dalla fretta", TitlesResolved, 4, 14, 15, []string{"Ci sarà sempre Tazmania", "Tagliati dalla fretta"}, 4, 9, 9},
		// " & " inside a title is not a separator
		{"Taz-Mania - S01E04 - War & Pieces HDTV-720p", TitlesResolved, 1, 4, 4, []string{"Guerra e pace"}, 1, 4, 4},
		// ... but between two titles it is
		{"Taz-Mania - S01E02 - Like Father, Like Son & Frights of Passage", TitlesResolved, 1, 2, 3, []string{"Tale padre, tale figlio", "Una sfida impossibile"}, 1, 2, 2},
		// Certain but not consecutive
		{"Taz-Mania - S01E02 - Like Father, Like Son + Deer Taz", TitlesReview, 1, 2, 2, nil, 1, 2, 2},
		// One title certain elsewhere, the other unknown
		{"Taz-Mania - S01E05 - Tazmanian Lullaby + Something Nobody Wrote", TitlesReview, 1, 5, 5, nil, 1, 5, 5},
		// Two unknown titles for one episode
		{"Taz-Mania - S01E05 - Something Nobody Wrote + Another Mystery", TitlesReview, 1, 5, 5, nil, 1, 5, 5},
		// One unknown title: nothing to say, keep the number
		{"Taz-Mania - S01E05 - Something Nobody Wrote", TitlesKeep, 1, 5, 5, nil, 1, 5, 5},
		{"Taz-Mania - S04E12 - HDTV-720p", TitlesNone, 4, 12, 12, nil, 4, 12, 12},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := resolveTitles(episodeTitlesFromFilename(tt.file), eps, tt.fileSeason, tt.fileEp, tt.fileE)
			if got.Status != tt.status {
				t.Fatalf("status = %v, want %v (reason %q, matches %+v)", got.Status, tt.status, got.Reason, got.Matches)
			}
			if got.Season != tt.season || got.Start != tt.start || got.End != tt.end {
				t.Errorf("range = S%02dE%02d-E%02d, want S%02dE%02d-E%02d", got.Season, got.Start, got.End, tt.season, tt.start, tt.end)
			}
			if tt.names != nil && !reflect.DeepEqual(got.Names, tt.names) {
				t.Errorf("names = %q, want %q", got.Names, tt.names)
			}
			if got.Status == TitlesReview && got.Reason == "" {
				t.Errorf("review without a reason")
			}
		})
	}
}

// Live TMDB: the Taz-Mania files are renumbered from their titles
func TestTitlesRenumberTazMania(t *testing.T) {
	skipTMDBInShortMode(t)
	tests := []struct {
		file, season, episode, name, target string
	}{
		{"Taz-Mania - S01E02 - Like Father, Like Son + Frights of Passage HDTV-720p.mkv", "01", "02-03", "Tale padre, tale figlio & Una sfida impossibile",
			"Tazmania (1991) {tmdb-10126}/Season 01/Tazmania - S01E02-E03 - Tale padre, tale figlio & Una sfida impossibile.mkv"},
		{"Taz-Mania - S02E05 - Tazmanian Lullaby + Deer Taz + A Taz-Manian Moment HDTV-720p.mkv", "02", "08-10", "Ninnananna tazmaniana & Cerbiatto caro & Episodio 10", ""},
		{"Taz-Mania - S04E09 - We'll Always Have Taz-Mania + Moments You've Missed HDTV-720p.mkv", "04", "14-15", "Ci sarà sempre Tazmania & Tagliati dalla fretta", ""},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: true, ForceSeries: true})
			if p.ExternalID != 10126 {
				t.Fatalf("ExternalID = %d, want 10126", p.ExternalID)
			}
			if p.Season != tt.season || p.Episode != tt.episode || p.EpisodeName != tt.name {
				t.Errorf("got S%sE%s %q, want S%sE%s %q", p.Season, p.Episode, p.EpisodeName, tt.season, tt.episode, tt.name)
			}
			if p.NeedsReview || p.TitleCheck != "resolved" {
				t.Errorf("TitleCheck = %q, NeedsReview = %v", p.TitleCheck, p.NeedsReview)
			}
			if p.FileSeason == "" || p.FileEpisode == "" {
				t.Errorf("file numbering not kept: %q %q", p.FileSeason, p.FileEpisode)
			}
			if tt.target != "" && p.TargetName() != tt.target {
				t.Errorf("TargetName() = %q, want %q", p.TargetName(), tt.target)
			}
		})
	}

	// No titles: the numbering of the file name, as before
	p := NewParsedFile("Taz-Mania - S04E12 - HDTV-720p.mkv", Options{Lookup: true, ForceSeries: true})
	if p.Season != "04" || p.Episode != "12" || p.NeedsReview || p.TitleCheck != "" {
		t.Errorf("no titles: got S%sE%s, TitleCheck %q, NeedsReview %v", p.Season, p.Episode, p.TitleCheck, p.NeedsReview)
	}

	// Titles that contradict the number and cannot be resolved
	p = NewParsedFile("Taz-Mania - S01E05 - Tazmanian Lullaby + Something Nobody Wrote HDTV-720p.mkv", Options{Lookup: true, ForceSeries: true})
	if !p.NeedsReview || p.ReviewReason == "" || p.Season != "01" || p.Episode != "05" {
		t.Errorf("review: got S%sE%s, NeedsReview %v, reason %q", p.Season, p.Episode, p.NeedsReview, p.ReviewReason)
	}
}

// Live TMDB: a normal file whose title matches its number does not change
func TestTitlesMatchingNumber(t *testing.T) {
	skipTMDBInShortMode(t)
	p := NewParsedFile("Breaking Bad - S01E05 - Gray Matter HDTV-720p.mkv", Options{Lookup: true})
	if p.ExternalID != 1396 {
		t.Fatalf("ExternalID = %d, want 1396", p.ExternalID)
	}
	if p.Season != "01" || p.Episode != "05" || p.NeedsReview || p.FileEpisode != "" {
		t.Errorf("got S%sE%s, NeedsReview %v, FileEpisode %q", p.Season, p.Episode, p.NeedsReview, p.FileEpisode)
	}
	if p.TitleCheck != "resolved" {
		t.Errorf("TitleCheck = %q, want resolved", p.TitleCheck)
	}
}
