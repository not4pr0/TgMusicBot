/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type cookieFileCandidate struct {
	path    string
	modUnix int64
	rank    int
}

// discoverCookieFiles collects cookie files from the COOKIES_PATH (files) and
// COOKIES_DIR (directories) environment variables, both comma-separated.
// Results are ordered by rank then by most recently modified.
func discoverCookieFiles(pathsRaw, dirsRaw string) []string {
	seen := make(map[string]bool)
	var candidates []cookieFileCandidate

	addPath := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}

		st, err := os.Stat(path)
		if err != nil {
			slog.Info("Cookie path ignored", "path", path, "error", err)
			return
		}
		if st.IsDir() {
			addCookieDir(&candidates, seen, path)
			return
		}
		addCookieFile(&candidates, seen, path, st)
	}

	for _, raw := range strings.Split(pathsRaw, ",") {
		addPath(raw)
	}
	for _, raw := range strings.Split(dirsRaw, ",") {
		addPath(raw)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		if candidates[i].modUnix == candidates[j].modUnix {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].modUnix > candidates[j].modUnix
	})

	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.path)
	}
	if len(paths) > 0 {
		slog.Info("Loaded YouTube cookie files", "count", len(paths), "first", filepath.Base(paths[0]))
	}
	return paths
}

func addCookieDir(candidates *[]cookieFileCandidate, seen map[string]bool, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Info("Cookie directory ignored", "path", dir, "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)
		if !strings.HasSuffix(lowerName, ".txt") && !strings.Contains(lowerName, "cookie") {
			continue
		}
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err != nil {
			slog.Info("Cookie file ignored", "path", path, "error", err)
			continue
		}
		addCookieFile(candidates, seen, path, st)
	}
}

func addCookieFile(candidates *[]cookieFileCandidate, seen map[string]bool, path string, st os.FileInfo) {
	cleanPath := filepath.Clean(path)
	if seen[cleanPath] {
		return
	}
	if st.Size() <= 0 {
		slog.Info("Cookie file ignored because it is empty", "path", cleanPath)
		return
	}
	if !looksLikeCookieFile(cleanPath) {
		slog.Info("Cookie file ignored because it does not look like a Netscape cookie file", "path", cleanPath)
		return
	}

	rank := cookieFileRank(cleanPath)
	if !cookieFileHasLogin(cleanPath) {
		// Anonymous exports (no LOGIN_INFO cookie) make YouTube return empty
		// search results and "Sign in to confirm you're not a bot" walls, so
		// they are only tried once every logged-in export has failed.
		rank += 10
	}

	seen[cleanPath] = true
	*candidates = append(*candidates, cookieFileCandidate{
		path:    cleanPath,
		modUnix: st.ModTime().Unix(),
		rank:    rank,
	})
}

// cookieFileHasLogin reports whether a Netscape cookie export carries a
// LOGIN_INFO cookie, which only a genuinely logged-in YouTube session has.
func cookieFileHasLogin(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte("LOGIN_INFO"))
}

func cookieFileRank(path string) int {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasPrefix(name, "cookies-"):
		return 0
	case name == "cookies.txt":
		return 1
	case strings.Contains(name, "bkp") || strings.Contains(name, "bak") || strings.Contains(name, "old"):
		return 3
	default:
		return 2
	}
}

func looksLikeCookieFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info("Cookie file ignored because it cannot be read", "path", path, "error", err)
		return false
	}
	if len(data) > 8192 {
		data = data[:8192]
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "netscape http cookie file") ||
		strings.Contains(content, "youtube.com") ||
		strings.Contains(content, "google.com")
}
