package identify

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Anime style names
//
// Fansub releases number episodes without SxxEyy:
//
//	[SubsPlease] Frieren - 12 (1080p) [ABCD1234].mkv     absolute episode 12
//	[Erai-raws] One Piece - 1071 [1080p].mkv             absolute episode 1071
//	[Group] Show S2 - 05 [1080p].mkv                     episode 5 of season 2
//
// An absolute number becomes the N-th regular episode of the show on TMDB
// (specials excluded), which works both for shows numbered per season and
// for shows numbered absolutely within their seasons (One Piece). Without
// TMDB the file goes to season 1. When TMDB does not have the episode (not
// listed yet, or a season TMDB merges with another) the file needs a review.

var (
	// " - 12", " - 1071", " - 05v2" followed by a separator or the end
	animeEpisodeRegex = regexp.MustCompile(`\s-\s([0-9]{2,4})(?:v[0-9]+)?(?:[\s\[\(._-]|$)`)
	// Season written in the name before the number: "Show S2", "Show Season 2", "Show 2nd Season"
	animeSeasonRegex = regexp.MustCompile(`(?i)\s+(?:s([0-9]{1,2})|season\s+([0-9]{1,2})|([0-9]{1,2})(?:st|nd|rd|th)\s+season)$`)
	yearLikeRegex    = regexp.MustCompile(`^(?:19|20)[0-9]{2}$`)
)

// parseAnimeEpisode reads " - NN" when the name has no SxxEyy. Only for
// fansub names ([Group] at the start) or names without a year, so that
// "Fast & Furious - 10 (2023)" stays a movie.
func (f *ParsedFile) parseAnimeEpisode(name string) {
	if f.Episode != "" || (f.AnimeGroup == "" && f.Year != "") {
		return
	}
	for _, loc := range animeEpisodeRegex.FindAllStringSubmatchIndex(name, -1) {
		num := name[loc[2]:loc[3]]
		if yearLikeRegex.MatchString(num) {
			continue
		}
		n, _ := strconv.Atoi(num)
		if n == 0 {
			continue
		}
		before := name[:loc[0]]
		f.animeName = animeSeriesName(before)
		if m := animeSeasonRegex.FindStringSubmatch(before); m != nil {
			season, _ := strconv.Atoi(m[1] + m[2] + m[3])
			f.Season = fmt.Sprintf("%02d", season)
			f.Episode = fmt.Sprintf("%02d", n)
			f.animeSeason = true
			log.WithFields(log.Fields{"season": season, "episode": n}).Debugln("Found an anime style episode with the season in the name.")
		} else {
			f.Season = "01"
			f.Episode = fmt.Sprintf("%02d", n)
			f.AbsoluteEpisode = n
			log.WithField("absolute", n).Debugln("Found an anime style absolute episode number.")
		}
		return
	}
}

// animeSeriesName: the text before " - NN" without the [Group], the year and
// the season ("[HorribleSubs] Fruits Basket (2019)" → "Fruits Basket")
func animeSeriesName(before string) string {
	if m := matchers["groupAnime"].FindStringIndex(before); m != nil {
		before = before[m[1]:]
	}
	if !strings.Contains(before, " ") {
		before = strings.NewReplacer(".", " ", "_", " ").Replace(before)
	}
	before = animeSeasonRegex.ReplaceAllString(before, "")
	before = matchers["year"].ReplaceAllString(before, " ")
	return strings.Join(strings.Fields(before), " ")
}

// mapAnimeEpisode turns the anime numbering into the TMDB one, or marks the
// file for review when TMDB does not have that episode.
func (p *ParsedFile) mapAnimeEpisode(showID int) {
	if p.AbsoluteEpisode == 0 && !p.animeSeason {
		return
	}
	episodes, err := showEpisodes(showID, p.Options.TMDBLanguage)
	if err != nil {
		log.WithFields(log.Fields{"show": showID, "error": err}).Warnln("Could not fetch the TMDB episode list, keeping the anime numbering as is")
		return
	}
	var regular []tmdbEpisode
	for _, e := range episodes {
		if e.Season > 0 {
			regular = append(regular, e)
		}
	}
	sort.Slice(regular, func(i, j int) bool {
		if regular[i].Season != regular[j].Season {
			return regular[i].Season < regular[j].Season
		}
		return regular[i].Episode < regular[j].Episode
	})

	if p.AbsoluteEpisode > 0 {
		n := p.AbsoluteEpisode
		if n > len(regular) {
			p.NeedsReview = true
			p.ReviewReason = fmt.Sprintf("episodio assoluto %d, TMDB elenca solo %d episodi", n, len(regular))
			log.WithFields(log.Fields{"absolute": n, "known": len(regular)}).Warnln("The absolute episode is not on TMDB yet, the file needs a review.")
			return
		}
		e := regular[n-1]
		p.Season = fmt.Sprintf("%02d", e.Season)
		p.Episode = fmt.Sprintf("%02d", e.Episode)
		log.WithFields(log.Fields{"absolute": n, "tmdb": e.code()}).Infoln("Mapped the absolute episode number to TMDB.")
		return
	}

	season, _ := strconv.Atoi(p.Season)
	episode, _ := strconv.Atoi(p.Episode)
	for _, e := range regular {
		if e.Season == season && e.Episode == episode {
			return
		}
	}
	var seasons []string
	counts := map[int]int{}
	for _, e := range regular {
		if counts[e.Season] == 0 {
			seasons = append(seasons, strconv.Itoa(e.Season))
		}
		counts[e.Season]++
	}
	p.NeedsReview = true
	p.ReviewReason = fmt.Sprintf("%s nel nome ma TMDB non ha quell'episodio (stagioni %s)", rangeCode(season, episode, episode), strings.Join(seasons, ", "))
	log.WithFields(log.Fields{"file": rangeCode(season, episode, episode)}).Warnln("The anime season in the name does not exist on TMDB, the file needs a review.")
}

// findEpisodeMatch is the first match of the "episode" matcher that is not a
// codec: "x264", "x265" are not episodes; an X episode needs the season
// digits before it (1x05).
func findEpisodeMatch(name string) []string {
	for _, loc := range matchers["episode"].FindAllStringSubmatchIndex(name, -1) {
		if (name[loc[0]] == 'x' || name[loc[0]] == 'X') && (loc[0] == 0 || name[loc[0]-1] < '0' || name[loc[0]-1] > '9') {
			continue
		}
		res := make([]string, len(loc)/2)
		for i := range res {
			if loc[2*i] >= 0 {
				res[i] = name[loc[2*i]:loc[2*i+1]]
			}
		}
		return res
	}
	return nil
}

// findMatch runs one of the matchers on a name
func findMatch(match, name string) []string {
	if match == "episode" {
		return findEpisodeMatch(name)
	}
	return matchers[match].FindStringSubmatch(name)
}
