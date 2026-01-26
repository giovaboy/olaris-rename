package identify

import (
	"fmt"
	"strconv"

	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	log "github.com/sirupsen/logrus"
)

type ParsedFile struct {
	Year         string
	Season       string
	Episode      string
	EpisodeName  string
	ExternalName string
	CleanName    string
	Filepath     string
	Filename     string
	Extension    string
	Quality      string
	Resolution   string
	Group        string
	AnimeGroup   string
	IsSeries     bool
	IsMovie      bool
	IsMusic      bool
	ExternalID   int
	OriginalFile string
	Options      Options

	hasYearAsSeason bool
}

func (p *ParsedFile) String() string {
	return fmt.Sprintf("Year: %s, Season: %s, Episode: %s, EpisodeName: %s, Name: %s, Movie: %v, Series: %v", p.Year, p.Season, p.Episode, p.EpisodeName, p.CleanName, p.IsMovie, p.IsSeries)
}

type Options struct {
	Lookup       bool
	ForceMovie   bool
	ForceSeries  bool
	OriginalFile string
	MovieFormat  string
	SeriesFormat string
	DryRun       bool
	TMDBLanguage string
}

func (p *Options) String() string {
	return fmt.Sprintf("Lookup: %v, ForceMovie: %v, ForceSeries: %v, OriginalFile: %s, MovieFormat: %s, SeriesFormat: %s, DryRun: %v, TMDBLanguage: %s", p.Lookup, p.ForceMovie, p.ForceSeries, p.OriginalFile, p.MovieFormat, p.SeriesFormat, p.DryRun, p.TMDBLanguage)
}

func GetDefaultOptions() Options {
	return getOpts([]Options{})
}

func getOpts(o []Options) (opts Options) {
	if len(o) == 0 {
		opts = Options{}
	} else {
		opts = o[0]
	}
	if opts.MovieFormat == "" {
		opts.MovieFormat = DefaultMovieFormat
	}
	if opts.SeriesFormat == "" {
		opts.SeriesFormat = DefaultSeriesFormat
	}
	if opts.TMDBLanguage == "" {
		opts.TMDBLanguage = DefaultTMDBLanguage
	}
	return opts
}

// EpisodeInfo holds parsed episode number(s)
type EpisodeInfo struct {
	Start   int
	End     int
	IsRange bool
}

// GetFirstEpisodeForLookup returns the first episode number for TMDB lookup
func (ei *EpisodeInfo) GetFirstEpisodeForLookup() int {
	return ei.Start
}

// GetEpisodeRange returns a string representation of the episode range
// Returns "22" for single episodes or "22-23" for ranges
func (ei *EpisodeInfo) GetEpisodeRange() string {
	if ei.IsRange {
		return fmt.Sprintf("%d-%d", ei.Start, ei.End)
	}
	return fmt.Sprintf("%d", ei.Start)
}

// ParseEpisodeString parses episode patterns like:
// - "22" → {Start: 22, End: 22, IsRange: false}
// - "22-23" → {Start: 22, End: 23, IsRange: true}
// - "E22" → {Start: 22, End: 22, IsRange: false}
// - "E22E23" → {Start: 22, End: 23, IsRange: true}
// - "22E23" → {Start: 22, End: 23, IsRange: true}
func ParseEpisodeString(epStr string) (*EpisodeInfo, error) {
	if epStr == "" {
		return nil, fmt.Errorf("empty episode string")
	}

	epStr = strings.TrimSpace(epStr)

	// Regex to match: optional 'E', digits, optional separator (dash/E), and more digits
	// Groups: (1) first episode number, (2) second episode number (if present)
	//pattern := regexp.MustCompile(`^E?(\d+)(?:[E\-]*E?(\d+))?$`)
	pattern := regexp.MustCompile(`^E?(\d+)(?:(?:-E?|E)(\d+))?$`)
	matches := pattern.FindStringSubmatch(epStr)

	if matches == nil {
		return nil, fmt.Errorf("invalid episode format: %s", epStr)
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("could not parse start episode: %s", matches[1])
	}

	info := &EpisodeInfo{
		Start:   start,
		End:     start,
		IsRange: false,
	}

	// If there's a second episode number, it's a range
	if len(matches) > 2 && matches[2] != "" {
		end, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("could not parse end episode: %s", matches[2])
		}

		if start > end {
			return nil, fmt.Errorf("start episode (%d) cannot be greater than end episode (%d)", start, end)
		}

		info.End = end
		info.IsRange = true
	}

	return info, nil
}

func NewParsedFile(filePath string, o ...Options) ParsedFile {
	opts := getOpts(o)
	log.WithField("options", opts.String()).Debugln("Parsing filename with options")
	f := ParsedFile{Filepath: filePath, OriginalFile: opts.OriginalFile, Options: opts}
	f.Extension = filepath.Ext(filePath)
	filename := strings.TrimSuffix(filePath, f.Extension)
	filename = filepath.Base(filename)
	f.Filename = filename
	log.WithFields(log.Fields{"file": f.Filename}).Debugln("Checking file")

	if SupportedVideoExtensions[f.Extension] {
		for _, match := range order {
			res := matchers[match].FindStringSubmatch(filename)
			if len(res) > 0 {
				switch match {
				case "yearAsSeason":
					if len(res) > 1 {
						log.WithField("year", res[2]).Debugln("Found year as season.")
						f.hasYearAsSeason = true
						f.Season = res[2]
					}
				case "year":
					if f.Season == res[2] {
						log.Warnln("We found a year that is the same as the season, to prevent issues with looking up the wrong year we are ignoring the found year.")
					} else {
						f.Year = res[2]
					}
				case "season":
					if f.Season == "" {
						f.Season = fmt.Sprintf("%02s", res[2])
					} else {
						log.Debugln("We already have found a season earlier so skipping the normal season match.")
					}
				case "episode":
		    // Parse the raw episode string (could be "22", "22E23", "E22E23", etc.)
		    episodeInfo, err := ParseEpisodeString(res[0])
		    if err != nil {
		        log.WithFields(log.Fields{"raw": res[0], "error": err}).Debugln("Could not parse episode string")
		        // Fallback: just format the raw string
		        f.Episode = fmt.Sprintf("%02s", res[1])
		    } else {
		        // Format properly: "22" for single or "22-E23" for range (Plex format)
		        if episodeInfo.IsRange {
		            f.Episode = fmt.Sprintf("%02d-E%02d", episodeInfo.Start, episodeInfo.End)  // ← CORRETTO
		        } else {
		            f.Episode = fmt.Sprintf("%02d", episodeInfo.Start)
		        }
		    }
				case "quality":
					f.Quality = res[1]
				case "resolution":
					f.Resolution = res[2]
				case "groupAnime":
					f.AnimeGroup = res[1]
				case "episodeAnime":
					if f.Episode == "" {
						f.Episode = strings.Trim(res[0], " ")
						f.Season = "00"
					}
				}
			}
		}

		cleanName := strings.Replace(f.Filename, ".", " ", -1)

		if !f.IsMusic {
			log.WithFields(log.Fields{"cleanName": cleanName, "year": f.Year, "episode": f.Episode, "season": f.Season}).Debugln("Pre-parsing done, initial result.")
			// CORRECTED AUTO-DETECT LOGIC - Replace the section in your identify/file.go
// Around line 100-130

if opts.ForceMovie {
    f.IsMovie = true
    f.Episode = ""  // Clear false positives
    f.Season = ""
    log.Debugln("Identified file as a movie (forced)")
} else if opts.ForceSeries {
    f.IsSeries = true
    log.Debugln("Identified file as a series (forced)")
} else if f.Season != "" {
    // If season is present, it's definitely a series
    f.IsSeries = true
    log.Debugln("Identified file as an episode (has season)")
} else if f.Year != "" && f.Episode == "" {
    // Year without episode = movie
    f.IsMovie = true
    log.Debugln("Identified file as a movie (has year, no episode)")
} else if f.Year != "" && f.Episode != "" {
    // Year AND episode (without season) = probably movie with false positive episode
    // Log the false positive BEFORE clearing it
    log.WithFields(log.Fields{
        "falsePositiveEpisode": f.Episode,  // Log BEFORE clearing
        "year": f.Year,
    }).Debugln("Identified file as movie, cleared false positive episode")
    f.IsMovie = true
    f.Episode = ""  // Clear false positive AFTER logging
    f.Season = ""
} else if f.Episode != "" && f.Season != "" {
    // Episode AND season (both present) = series
    // This handles S##E## format correctly
    f.IsSeries = true
    log.Debugln("Identified file as an episode (has both season and episode)")
} else {
    // Nothing sensible found, try parent directory
    fileParent := filepath.Base(filepath.Dir(filePath))
    if fileParent != "" && opts.OriginalFile == "" && fileParent != "." {
        log.WithFields(log.Fields{"file": f.Filename, "filePath": filePath, "fileParent": fileParent}).Warnln("Nothing sensible found, trying again with parent.")
        opts.OriginalFile = filePath
        return NewParsedFile(fileParent+f.Extension, opts)
    }
}

			for _, match := range order {
				res := matchers[match].FindStringSubmatch(cleanName)
				if len(res) > 0 {
					// We don't need to remove Episode and Season information from movies, so let's exclude some properties
					if (f.IsMovie && !ignoreMovie[match]) || f.IsSeries {
						if match == "episode" {
							cleanName = strings.Replace(cleanName, res[0], " ", -1)
						} else if match == "season" {
							cleanName = strings.Replace(cleanName, res[1], " ", -1)
						} else if match == "groupAnime" {
							cleanName = strings.Replace(cleanName, res[1], " ", -1)
						} else {
							oldName := cleanName
							cleanName = matchers[match].ReplaceAllString(cleanName, " ")
							if len(strings.TrimRight(cleanName, " ")) < 2 {
								log.WithFields(log.Fields{"matcher": match, "newName": cleanName, "oldName": oldName}).Debugln("The match we just did made the name of the content smaller than two characters, we are going to assume something went wrong and reverting to the previous name.")
								cleanName = oldName
							}
						}
					}
				}
			}

			cleanName = strings.Trim(cleanName, " ")

			// Anime content is really weird, if we do this we might kill the name completely
			if f.AnimeGroup == "" {
				log.WithField("cleanName", cleanName).Debugln("Probably not Anime so cleaning a bit more.")
				cleanName = regexp.MustCompile(`\s{2,}.*`).ReplaceAllString(cleanName, "")
				cleanName = properTitleCase(cleanName)//cases.Title(language.English).String(cleanName)
			}
		}

		cleanName = strings.Replace(cleanName, ":", "", -1)

		f.CleanName = cleanName
	} else if SupportedMusicExtensions[f.Extension] {
		f.IsMusic = true
		return f
	} else {
		return f
	}

	if opts.Lookup {
		queryTmdb(&f)
	}

	if f.ExternalID > 0 && f.hasYearAsSeason {
		// Translate season as year to season number
		agent := initAgent()
		opzioni := make(map[string]string)
		opzioni["language"] = f.Options.TMDBLanguage
		details, err := agent.GetTvInfo(f.ExternalID, opzioni)
		if err != nil {
			log.Errorln("Could not locate TV even though we just found an external ID, this shouldn't be possible. Error:", err)
		}
		couldTranslate := false
		for _, s := range details.Seasons {
			if s.Name == fmt.Sprintf("Season %s", f.Season) {
				log.Debugln("Found a match for the season name, using season number.")
				f.Season = strconv.Itoa(s.SeasonNumber)
				couldTranslate = true
				break
			}
		}
		if !couldTranslate {
			log.Warnln("Could not translate season as year to normal season :-(")
		}
	} else if f.hasYearAsSeason && !f.Options.Lookup {
		log.Warnln("Found an episode that has a year as season but lookup is disabled so not translating season as year to normal season.")
	}

	if addYearToSeries[f.CleanName] && f.Year != "" {
		log.WithFields(log.Fields{"year": f.Year, "name": f.CleanName}).Debugln("Found seriesname that has multiple series with the same name but different years so adding the year into the final name.")
		f.CleanName = fmt.Sprintf("%s (%s)", f.CleanName, f.Year)
	}

	// Windows really hates colons, so lets strip them out.
	f.CleanName = strings.Replace(f.CleanName, ":", "", -1)

	log.WithField("cleanName", f.String()).Infoln("Done parsing filename.")

	return f
}

// SourcePath returns the originfile if one is available otherwise does the most recursed hit.
func (p *ParsedFile) SourcePath() string {
	if p.OriginalFile == "" {
		return p.Filepath
	} else {
		return p.OriginalFile
	}
}

func queryTmdb(p *ParsedFile) error {
	agent := initAgent()

	var options = make(map[string]string)
	options["language"] = p.Options.TMDBLanguage
	if p.Year != "" {
		options["first_air_date_year"] = p.Year
		options["year"] = p.Year
	}

	log.WithFields(log.Fields{"year": p.Year, "title": p.CleanName}).Debugln("Trying to locate data from TMDB")
	if p.IsSeries {
		searchRes, err := agent.SearchTv(p.CleanName, options)
		if err != nil {
			log.WithFields(log.Fields{"name": p.CleanName, "error": err}).Warnln("Got an error from TMDB")
			return err
		}

		if len(searchRes.Results) > 0 {
			tv := searchRes.Results[0] // Take the first result for now
			log.Debugln("TV:", tv)
			p.ExternalID = tv.ID
			p.ExternalName = tv.Name
			p.CleanName = tv.Name
			if tv.FirstAirDate != "" && p.Year == "" {
				p.Year = strings.Split(tv.FirstAirDate, "-")[0]
			}

			// IMPROVED: Retrieve Episode Name(s) for single or multiple episodes
			seasonNum, err := strconv.Atoi(p.Season)
			if err != nil {
				log.WithFields(log.Fields{"season": p.Season, "error": err}).Warnln("Could not convert season to int")
				return err
			}

			// Parse the episode string to handle ranges properly
			episodeInfo, err := ParseEpisodeString(p.Episode)
			if err != nil {
				log.WithFields(log.Fields{"episode": p.Episode, "error": err}).Warnln("Could not parse episode string")
				// Continue anyway, we can still try with just the first part
				episodeInfo = &EpisodeInfo{Start: 1, End: 1, IsRange: false}
			}

			// Fetch episode names for all episodes in the range
			episodeTitles := []string{}
			for episodeNum := episodeInfo.Start; episodeNum <= episodeInfo.End; episodeNum++ {
				log.WithFields(log.Fields{
					"series":  p.CleanName,
					"season":  seasonNum,
					"episode": episodeNum,
				}).Debugln("Fetching episode info from TMDB")

				tvEpisode, err := agent.GetTvEpisodeInfo(tv.ID, seasonNum, episodeNum, options)
				if err != nil {
					log.WithFields(log.Fields{
						"season":  seasonNum,
						"episode": episodeNum,
						"error":   err,
					}).Debugln("Could not fetch episode from TMDB")
					continue
				}

				if len(tvEpisode.Name) > 0 {
					log.WithFields(log.Fields{
						"season":  seasonNum,
						"episode": episodeNum,
						"name":    tvEpisode.Name,
					}).Debugln("Retrieved episode info from TMDB")
					episodeTitles = append(episodeTitles, tvEpisode.Name)
				}
			}

			// Join episode names if multiple episodes (use " & " separator)
			if len(episodeTitles) > 0 {
				p.EpisodeName = strings.Join(episodeTitles, " & ")
			}
		} else {
			log.Debugln("No results found on TMDB")
		}

	} else if p.IsMovie {
		searchRes, err := agent.SearchMovie(p.CleanName, options)
		if err != nil {
			log.WithFields(log.Fields{"name": p.CleanName, "error": err}).Warnln("Got an error from TMDB")
			return err
		}

		if len(searchRes.Results) > 0 {
			mov := searchRes.Results[0] // Take the first result for now
			log.Debugln("Movie:", mov)

			p.ExternalID = mov.ID
			p.ExternalName = mov.Title
			p.CleanName = mov.Title

		} else {
			log.Debugln("No results found on TMDB")
		}
	}

	log.WithFields(log.Fields{"externalID": p.ExternalID, "externalName": p.ExternalName}).Debugln("Received TMDB results.")

	return nil
}

// TargetName is the name the file should be renamed to
func (p *ParsedFile) TargetName() string {
	var newName string

	if p.IsMovie {
		newName = p.Options.MovieFormat
	} else if p.IsSeries {
		newName = p.Options.SeriesFormat
		newName = strings.Replace(newName, "{s}", p.Season, -1)
		newName = strings.Replace(newName, "{e}", p.Episode, -1)
		newName = strings.Replace(newName, "{x}", p.EpisodeName, -1)
	} else {
		newName = p.Filename
	}

	newName = strings.Replace(newName, "{n}", p.CleanName, -1)

	newName = strings.Replace(newName, "{r}", p.Resolution, -1)
	newName = strings.Replace(newName, "{q}", p.Quality, -1)
	newName = strings.Replace(newName, "{y}", p.Year, -1)
	newName = strings.Trim(newName, " ")

	// Sometimes we end up with an extra dot somehow. This should remove the extra dot
	// I'm tired, this can probably be done better...
	if (string(newName[len(newName)-1])) == "." {
		newName = newName[0 : len(newName)-1]
	}

	return newName + p.Extension
}

// FullName is the original name of the file without the ful path
func (p *ParsedFile) FullName() string {
	return p.Filename + p.Extension
}

// EpisodeNum converts Episode to integer, mainly for Olaris-Server support
func (p *ParsedFile) EpisodeNum() (episodeNum int) {
	episodeNum, err := strconv.Atoi(p.Episode)
	if err != nil {
		log.Warnln("Received error when converting episode to int", err)
	}

	return episodeNum
}

// EpisodeNum converts Season to integer, mainly for Olaris-Server support
func (p *ParsedFile) SeasonNum() (seasonNum int) {
	seasonNum, err := strconv.Atoi(p.Season)
	if err != nil {
		log.Warnln("Received error when converting season to int", err)
	}

	return seasonNum
}

// properTitleCase converts a string to title case following English conventions
// Small words (of, the, and, etc.) remain lowercase unless they're the first word
func properTitleCase(s string) string {
	smallWords := map[string]bool{
		"a": true, "an": true, "and": true, "as": true, "at": true,
		"but": true, "by": true, "for": true, "in": true, "of": true,
		"on": true, "or": true, "the": true, "to": true, "via": true,
	}
	
	words := strings.Fields(s)
	for i, word := range words {
		lower := strings.ToLower(word)
		// First word or not a small word: capitalize
		if i == 0 || !smallWords[lower] {
			words[i] = cases.Title(language.English).String(lower)
		} else {
			words[i] = lower
		}
	}
	return strings.Join(words, " ")
}
