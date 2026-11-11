package identify

import (
	"fmt"
	"strings"
	"testing"
)

func TestAnimeFilenames(t *testing.T) {
	tests := []struct {
		file, name, season, episode string
		absolute                    int
	}{
		{"[SubsPlease] Frieren - 12 (1080p) [ABCD1234].mkv", "Frieren", "01", "12", 12},
		{"[Erai-raws] One Piece - 1071 [1080p][Multiple Subtitle].mkv", "One Piece", "01", "1071", 1071},
		{"[Group] Show Name - 05v2 (1080p).mkv", "Show Name", "01", "05", 5},
		{"Frieren - 12 [1080p] x265.mkv", "Frieren", "01", "12", 12},
		{"[Group] Show Name - 05 (1080p) [HEVC x265 10bit].mkv", "Show Name", "01", "05", 5},
		{"[Group] 86 - 05 [1080p].mkv", "86", "01", "05", 5},
		{"[HorribleSubs] Fruits Basket (2019) - 01 [1080p].mkv", "Fruits Basket", "01", "01", 1},
		// Season in the name: the number is relative to it
		{"[SubsPlease] Jujutsu Kaisen S2 - 05 (1080p).mkv", "Jujutsu Kaisen", "02", "05", 0},
		{"[Group] Show Season 3 - 10 [720p].mkv", "Show", "03", "10", 0},
		{"[Group] Show 2nd Season - 07 [720p].mkv", "Show", "02", "07", 0},
		// SxxEyy wins
		{"[Judas] Jujutsu Kaisen - S02E05.mkv", "Jujutsu Kaisen", "02", "05", 0},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			if !p.IsSeries || p.CleanName != tt.name || p.Season != tt.season || p.Episode != tt.episode || p.AbsoluteEpisode != tt.absolute {
				t.Errorf("got series %v %q S%sE%s absolute %d, want %q S%sE%s absolute %d", p.IsSeries, p.CleanName, p.Season, p.Episode, p.AbsoluteEpisode, tt.name, tt.season, tt.episode, tt.absolute)
			}
		})
	}
}

// "x264"/"x265" are codecs, not episodes
func TestCodecIsNotAnEpisode(t *testing.T) {
	tests := []struct {
		file, season, episode string
		series, movie         bool
	}{
		{"Show.S01E05.1080p.x265-GRP.mkv", "01", "05", true, false},
		{"Movie.2010.1080p.x264-GRP.mkv", "", "", false, true},
		{"Downton Abbey 5x06 HDTV x264-FoV.mkv", "05", "06", true, false},
		{"Some Show 1080p x265.mkv", "", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			if p.Season != tt.season || p.Episode != tt.episode || p.IsSeries != tt.series || p.IsMovie != tt.movie {
				t.Errorf("got S%qE%q series %v movie %v", p.Season, p.Episode, p.IsSeries, p.IsMovie)
			}
		})
	}
}

// Not anime: a year means a movie, a year-like number is not an episode
func TestAnimeNotMistaken(t *testing.T) {
	for _, file := range []string{"Fast & Furious - 10 (2023).mkv", "The Matrix Revolutions - 2003.mkv"} {
		p := NewParsedFile(file, Options{Lookup: false})
		if p.IsSeries || p.AbsoluteEpisode != 0 {
			t.Errorf("%s: series %v absolute %d", file, p.IsSeries, p.AbsoluteEpisode)
		}
	}
}

// fakeShow puts a show in the episode cache: seasons[i] episodes in season
// i+1, numbered from 1 or, absolute, continuing from the previous season.
func fakeShow(t *testing.T, id int, absolute bool, seasons ...int) {
	t.Helper()
	eps := []tmdbEpisode{newTmdbEpisode(0, 1, "Speciale", "Special")}
	n := 0
	for i, count := range seasons {
		for e := 1; e <= count; e++ {
			n++
			num := e
			if absolute {
				num = n
			}
			eps = append(eps, newTmdbEpisode(i+1, num, fmt.Sprintf("Ep %d", n), ""))
		}
	}
	key := fmt.Sprintf("%d/%s", id, DefaultTMDBLanguage)
	showEpisodesMutex.Lock()
	showEpisodesCache[key] = eps
	showEpisodesMutex.Unlock()
	t.Cleanup(func() {
		showEpisodesMutex.Lock()
		delete(showEpisodesCache, key)
		showEpisodesMutex.Unlock()
	})
}

func TestMapAnimeEpisode(t *testing.T) {
	fakeShow(t, -1, false, 12, 12, 25) // numbered per season
	fakeShow(t, -2, true, 61, 16, 14)  // numbered absolutely (One Piece)
	tests := []struct {
		show                  int
		file, season, episode string
		review                string
	}{
		{-1, "[G] Show - 05 [1080p].mkv", "01", "05", ""},
		{-1, "[G] Show - 13 [1080p].mkv", "02", "01", ""},
		{-1, "[G] Show - 30 [1080p].mkv", "03", "06", ""},
		{-1, "[G] Show - 49 [1080p].mkv", "03", "25", ""},
		{-1, "[G] Show - 50 [1080p].mkv", "01", "50", "assoluto 50"},
		{-2, "[G] Show - 70 [1080p].mkv", "02", "70", ""},
		{-2, "[G] Show - 91 [1080p].mkv", "03", "91", ""},
		{-1, "[G] Show S3 - 05 [1080p].mkv", "03", "05", ""},
		{-1, "[G] Show S4 - 05 [1080p].mkv", "04", "05", "stagioni 1, 2, 3"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: false})
			p.mapAnimeEpisode(tt.show)
			if p.Season != tt.season || p.Episode != tt.episode {
				t.Errorf("got S%sE%s, want S%sE%s", p.Season, p.Episode, tt.season, tt.episode)
			}
			if p.NeedsReview != (tt.review != "") || !strings.Contains(p.ReviewReason, tt.review) {
				t.Errorf("NeedsReview %v reason %q, want review %q", p.NeedsReview, p.ReviewReason, tt.review)
			}
		})
	}
}

// Live TMDB
func TestAnimeLookup(t *testing.T) {
	skipTMDBInShortMode(t)
	tests := []struct {
		file            string
		id              int
		season, episode string
		review          bool
	}{
		{"[SubsPlease] Frieren - 12 (1080p) x265.mkv", 209867, "01", "12", false},
		{"[SubsPlease] Jujutsu Kaisen - 30 (1080p).mkv", 95479, "01", "30", false},
		{"[Erai-raws] One Piece - 1071 [1080p].mkv", 37854, "21", "1071", false},
		// TMDB has Jujutsu Kaisen in one season: "S2 - 05" cannot be placed
		{"[SubsPlease] Jujutsu Kaisen S2 - 05 (1080p).mkv", 95479, "02", "05", true},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			p := NewParsedFile(tt.file, Options{Lookup: true})
			if p.ExternalID != tt.id || p.Season != tt.season || p.Episode != tt.episode || p.NeedsReview != tt.review {
				t.Errorf("got id %d S%sE%s review %v (%s), want %d S%sE%s review %v", p.ExternalID, p.Season, p.Episode, p.NeedsReview, p.ReviewReason, tt.id, tt.season, tt.episode, tt.review)
			}
		})
	}
}
