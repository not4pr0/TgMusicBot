package dl

import (
	"ashokshau/tgmusic/config"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var removableDownloadExtensions = map[string]struct{}{
	".aac": {}, ".flac": {}, ".m4a": {}, ".mp3": {}, ".mp4": {},
	".ogg": {}, ".opus": {}, ".wav": {}, ".webm": {}, ".part": {},
}

// CleanupDownloads removes stale media files while leaving Telegram session and
// other application files in the shared database directory untouched.
func CleanupDownloads() {
	if config.DownloadTTLHours <= 0 {
		return
	}
	entries, err := os.ReadDir(config.DownloadsDir)
	if err != nil {
		slog.Warn("failed to inspect downloads directory", "directory", config.DownloadsDir, "error", err)
		return
	}
	cutoff := time.Now().Add(-time.Duration(config.DownloadTTLHours) * time.Hour)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !isRemovableDownload(entry.Name()) {
			continue
		}
		path := filepath.Join(config.DownloadsDir, entry.Name())
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil {
			slog.Warn("failed to remove stale download", "file", entry.Name(), "error", err)
			continue
		}
		removed++
	}
	if removed > 0 {
		slog.Info("removed stale downloads", "count", removed, "ttl_hours", config.DownloadTTLHours)
	}
}

func isRemovableDownload(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".part" {
		return true
	}
	_, ok := removableDownloadExtensions[ext]
	return ok
}
