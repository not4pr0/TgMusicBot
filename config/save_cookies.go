/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	cookiesDr             = "src/cookies"
	cookieFetchTimeout    = 30 * time.Second
	cookieRefreshInterval = time.Hour
)

var (
	cookieHTTPClient = &http.Client{Timeout: cookieFetchTimeout}

	cookiesMu   sync.RWMutex
	cookiesPath []string
)

// GetCookiesPath returns a snapshot of the current cookie file paths.
func GetCookiesPath() []string {
	cookiesMu.RLock()
	defer cookiesMu.RUnlock()
	if len(cookiesPath) == 0 {
		return nil
	}
	out := make([]string, len(cookiesPath))
	copy(out, cookiesPath)
	return out
}

// setCookiesPath atomically replaces the cookie file paths.
func setCookiesPath(paths []string) {
	cookiesMu.Lock()
	cookiesPath = paths
	cookiesMu.Unlock()
}

// IsManagedCookiePath reports whether the path points into the directory where
// cookies fetched from COOKIES_URL are stored. Only those files may be deleted
// when YouTube rejects them; files discovered via COOKIES_PATH/COOKIES_DIR
// belong to the operator.
func IsManagedCookiePath(path string) bool {
	return strings.HasPrefix(filepath.Clean(path), cookiesDr+string(os.PathSeparator))
}

// fetchContent downloads content from Pastebin or Batbin.
func fetchContent(url string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	id := parts[len(parts)-1]

	var rawURL string
	if strings.Contains(url, "pastebin.com") {
		rawURL = fmt.Sprintf("https://pastebin.com/raw/%s", id)
	} else if strings.Contains(url, "batbin.me") {
		rawURL = fmt.Sprintf("https://batbin.me/raw/%s", id)
	} else {
		rawURL = url
	}

	resp, err := cookieHTTPClient.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to GET %s: %w", rawURL, err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body from %s: %w", rawURL, err)
	}

	return string(body), nil
}

// saveContent saves content to a file in the cookies directory and returns the file path.
func saveContent(url, content string) (string, error) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "file_" + strings.ReplaceAll(strings.Split(strings.ReplaceAll(url, "/", "_"), "?")[0], "#", "")
	}
	filename += ".txt"

	filePath := filepath.Join(cookiesDr, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return filePath, nil
}

// downloadAllCookies downloads all URLs and returns the saved file paths.
func downloadAllCookies(urls []string) []string {
	paths := make([]string, 0, len(urls))
	for _, url := range urls {
		content, err := fetchContent(url)
		if err != nil {
			slog.Info("Error fetching cookies from", "url", url, "error", err)
			continue
		}

		path, err := saveContent(url, content)
		if err != nil {
			slog.Info("Error saving cookies for", "url", url, "error", err)
			continue
		}

		paths = append(paths, path)
	}
	return paths
}

// refreshCookies rebuilds the full cookie path list (local files first, then
// downloaded ones) and swaps it atomically, so stale entries never accumulate.
func refreshCookies(urls []string) {
	paths := discoverCookieFiles(os.Getenv("COOKIES_PATH"), os.Getenv("COOKIES_DIR"))
	paths = append(paths, downloadAllCookies(urls)...)

	setCookiesPath(paths)
	slog.Info("Refreshed cookie files", "count", len(paths), "urls", len(urls))
}

// startCookieRefresher runs an initial fetch and then refreshes on a fixed
// interval, so expired cookie exports are replaced without restarting the bot.
func startCookieRefresher(urls []string) {
	if len(urls) == 0 && os.Getenv("COOKIES_PATH") == "" && os.Getenv("COOKIES_DIR") == "" {
		return
	}
	refreshCookies(urls)

	ticker := time.NewTicker(cookieRefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		refreshCookies(urls)
	}
}
