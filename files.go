package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/giovaboy/olaris-rename/identify"
	log "github.com/sirupsen/logrus"
)

// actOnFile performs one action (symlink, hardlink, copy, move) from source to target
func actOnFile(source, target, action string) error {
	switch action {
	case "symlink":
		real, err := filepath.EvalSymlinks(source)
		if err != nil {
			log.WithFields(log.Fields{"source": source, "err": err}).Debugln("error during symlink evaluation")
			return err
		}
		rel, err := filepath.Rel(filepath.Dir(target), real)
		if err != nil {
			log.WithFields(log.Fields{"targetLocation": filepath.Dir(target), "source": real, "err": err}).Debugln("error during Rel call")
			return err
		}
		log.WithFields(log.Fields{"target": target, "source": rel}).Debugln("Using relative path for symlinks.")
		return os.Symlink(rel, target)
	case "hardlink":
		return os.Link(source, target)
	case "copy":
		return copyFileContents(source, target)
	case "move":
		return os.Rename(source, target)
	}
	return fmt.Errorf("unknown action %q", action)
}

// SubtitleResult is a subtitle file acted on together with its video
type SubtitleResult struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

var subtitleExtensions = map[string]bool{
	".srt": true,
	".ass": true,
	".ssa": true,
	".sub": true,
	".idx": true,
	".vtt": true,
	".sup": true,
	".smi": true,
}

// sidecarSubtitles finds the subtitles of a video in its folder: the files
// named like the video, optionally followed by language and flags, e.g.
//
//	Movie.srt  Movie.en.srt  Movie.it.forced.srt  Movie.eng.sdh.srt
//
// Their target is the target of the video with the same suffix, which is
// the naming Plex reads (Movie (2009).en.forced.srt). When several videos
// share the start of the name (Show.S01E01, Show.S01E01.part2) a subtitle
// belongs to the longest one.
func sidecarSubtitles(video, target string) []SubtitleResult {
	dir := filepath.Dir(video)
	base := strings.TrimSuffix(filepath.Base(video), filepath.Ext(video))
	targetBase := strings.TrimSuffix(target, filepath.Ext(target))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var videos []string
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if !e.IsDir() && identify.SupportedVideoExtensions[strings.ToLower(ext)] {
			videos = append(videos, strings.TrimSuffix(e.Name(), ext))
		}
	}
	belongsTo := func(stem, videoBase string) bool {
		return stem == videoBase || strings.HasPrefix(stem, videoBase+".")
	}

	var subs []SubtitleResult
	for _, e := range entries {
		name := e.Name()
		ext := filepath.Ext(name)
		if e.IsDir() || !subtitleExtensions[strings.ToLower(ext)] {
			continue
		}
		stem := strings.TrimSuffix(name, ext)
		if !belongsTo(stem, base) {
			continue
		}
		owner := base
		for _, v := range videos {
			if len(v) > len(owner) && belongsTo(stem, v) {
				owner = v
			}
		}
		if owner != base {
			continue
		}
		subs = append(subs, SubtitleResult{
			Source: filepath.Join(dir, name),
			Target: targetBase + stem[len(base):] + strings.ToLower(ext),
		})
	}
	return subs
}
