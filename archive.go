package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/giovaboy/olaris-rename/identify"
	"github.com/mholt/archives"
	log "github.com/sirupsen/logrus"
)

// extractVideos extracts only the video files contained in an archive into e.extractPath
// and returns the paths of the extracted files.
//
// Every entry is written through an os.Root bound to extractPath, so entries that try to
// escape it ("../", absolute paths, symlinked directories) are rejected by the OS layer
// itself, regardless of what the archive library does. Requires Go >= 1.25.
func (e *App) extractVideos(archivePath string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx := context.Background()

	format, stream, err := archives.Identify(ctx, archivePath, f)
	if err != nil {
		return nil, fmt.Errorf("could not identify archive format: %w", err)
	}

	// Plain compressed streams (.gz, .bz2 without tar) are not archives and have no entries
	extractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, fmt.Errorf("%s is not an extractable archive", archivePath)
	}

	if err := ensurePath(e.extractPath); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(e.extractPath)
	if err != nil {
		return nil, fmt.Errorf("could not open extract path: %w", err)
	}
	defer root.Close()

	var extracted []string

	err = extractor.Extract(ctx, stream, func(ctx context.Context, fi archives.FileInfo) error {
		name := filepath.FromSlash(fi.NameInArchive)

		// Only regular files: skip directories, symlinks and hardlinks
		if !fi.Mode().IsRegular() {
			return nil
		}

		// We only care about video files, everything else (nfo, samples' txt, etc.) is ignored
		if !identify.SupportedVideoExtensions[strings.ToLower(filepath.Ext(name))] {
			return nil
		}

		// Explicit traversal check for a clear log line; os.Root would block it anyway
		if !filepath.IsLocal(name) {
			log.WithFields(log.Fields{"archive": archivePath, "entry": fi.NameInArchive}).Warnln("Skipping archive entry with unsafe path")
			return nil
		}

		// Keep the folder structure inside the archive: the parser can fall back
		// to the parent folder name when the file name itself is meaningless
		if dir := filepath.Dir(name); dir != "." {
			if err := root.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("could not create %s: %w", dir, err)
			}
		}

		src, err := fi.Open()
		if err != nil {
			return fmt.Errorf("could not open %s in archive: %w", fi.NameInArchive, err)
		}
		defer src.Close()

		dst, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("could not create %s: %w", name, err)
		}

		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			return fmt.Errorf("could not extract %s: %w", name, err)
		}
		if err := dst.Close(); err != nil {
			return err
		}

		target := filepath.Join(e.extractPath, name)
		log.WithFields(log.Fields{"archive": archivePath, "file": target}).Infoln("Extracted video from archive")
		extracted = append(extracted, target)

		return nil
	})

	return extracted, err
}
