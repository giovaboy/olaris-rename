package main

import (
	"flag"

	"gitlab.com/olaris/olaris-rename/identify"
)

var recursive = flag.Bool("recursive", true, "Scan folders inside of other folders.")
var logToFile = flag.Bool("log-to-file", false, "Logs are written to stdout as well as a logfile.")
var verbose = flag.Bool("verbose", false, "Show debug log information.")
var dryrun = flag.Bool("dry-run", false, "Don't actually modify any files.")
var action = flag.String("action", "symlink", "How to act on files, valid options are symlink, hardlink, copy or move.")
var filePath = flag.String("filepath", "", "Path to scan (can be a folder or file).")
var movieFolder = flag.String("movie-folder", defaultMovieFolder(), "Folder where movies should be placed.")
var seriesFolder = flag.String("series-folder", defaultSeriesFolder(), "Folder where series should be placed.")
var musicFolder = flag.String("music-folder", defaultMusicFolder(), "Folder where music should be placed.")
var tmdbLookup = flag.Bool("tmdb-lookup", true, "Should the TMDB be used for better look-up and matching.")
var extractPath = flag.String("extract-path", defaultExtractedFolder(), "Path to extract content to.")
var skipExtracting = flag.Bool("skip-extracting", false, "Disable automatic extraction.")
var minFileSize = flag.String("min-file-size", "120", "Minimal file size in MB for olaris-rename to consider a file valid to be processed.")
var seriesFormat = flag.String("series-format", identify.DefaultSeriesFormat, "Format used to rename series.")
var movieFormat = flag.String("movie-format", identify.DefaultMovieFormat, "Format used to rename movies.")
var forceMovie = flag.Bool("force-movie", false, "Forces the supplied path to be identified as a movie.")
var forceSeries = flag.Bool("force-series", false, "Forces the supplied path to be identified as a series.")
