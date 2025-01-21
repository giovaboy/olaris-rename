package main

import (
	"flag"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	flag.Parse()

	if *logToFile {
		lp := configFolderPath("olaris-rename.log")
		f, err := os.OpenFile(lp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
		defer f.Close()
	}

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *filePath == "" {
		log.Errorln("--filepath is a required argument.")
		flag.PrintDefaults()
		return
	}

	if !actions[*action] {
		log.Errorf("Unknown --action '%s'", *action)
		flag.PrintDefaults()
		return
	}

	if *dryrun {
		log.Warnln("--dry-run is enabled, not touching files")
	}

	e := NewApp(*recursive, *action, *movieFolder, *extractPath, *seriesFolder, *dryrun, *tmdbLookup, *skipExtracting, *minFileSize, *forceMovie, *forceSeries)
	e.StartRun(*filePath)
}
