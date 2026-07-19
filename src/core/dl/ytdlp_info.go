/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ytdlpInfoTimeout     = 90 * time.Second
	ytdlpDownloadTimeout = 5 * time.Minute
)

type ytdlpInfo struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	WebpageURL string       `json:"webpage_url"`
	URL        string       `json:"url"`
	Thumbnail  string       `json:"thumbnail"`
	Duration   float64      `json:"duration"`
	Channel    string       `json:"channel"`
	Uploader   string       `json:"uploader"`
	ViewCount  int64        `json:"view_count"`
	Entries    []ytdlpEntry `json:"entries"`
}

type ytdlpEntry struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	WebpageURL string  `json:"webpage_url"`
	Thumbnail  string  `json:"thumbnail"`
	Duration   float64 `json:"duration"`
	Channel    string  `json:"channel"`
	Uploader   string  `json:"uploader"`
	ViewCount  int64   `json:"view_count"`
}

// getInfoWithYtDlp fetches YouTube metadata with yt-dlp, trying each configured
// cookie file before falling back to a cookieless request. Used when InnerTube
// fails (VPS IP flagged by YouTube).
func (y *youTubeData) getInfoWithYtDlp(rawURL string, playlist bool) (utils.PlatformTracks, error) {
	if strings.TrimSpace(rawURL) == "" {
		return utils.PlatformTracks{}, errors.New("empty YouTube URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpInfoTimeout)
	defer cancel()

	cookieFiles := config.GetCookiesPath()
	if len(cookieFiles) == 0 {
		return runInfoYtDlp(ctx, rawURL, utils.YouTube, playlist, "", config.Proxy)
	}

	var attempts []string
	for _, cookieFile := range cookieFiles {
		tracks, err := runInfoYtDlp(ctx, rawURL, utils.YouTube, playlist, cookieFile, config.Proxy)
		if err == nil && len(tracks.Results) > 0 {
			return tracks, nil
		}
		if err == nil {
			err = errors.New("no tracks returned")
		}
		if isProxyConnectionError(err.Error()) {
			return utils.PlatformTracks{}, fmt.Errorf("yt-dlp info proxy failed with %s: %w", filepath.Base(cookieFile), err)
		}
		attempts = append(attempts, fmt.Sprintf("%s: %s", filepath.Base(cookieFile), err.Error()))
		slog.Warn("yt-dlp info failed with cookie file, trying next one", "cookie", filepath.Base(cookieFile), "error", err)
	}

	tracks, err := runInfoYtDlp(ctx, rawURL, utils.YouTube, playlist, "", config.Proxy)
	if err == nil && len(tracks.Results) > 0 {
		return tracks, nil
	}
	if err != nil {
		if isProxyConnectionError(err.Error()) {
			return utils.PlatformTracks{}, fmt.Errorf("yt-dlp info proxy failed without cookies: %w", err)
		}
		attempts = append(attempts, "without cookies: "+err.Error())
	}

	return utils.PlatformTracks{}, fmt.Errorf("yt-dlp info failed with all configured cookie files: %s", strings.Join(attempts, " | "))
}

// getGenericInfoWithYtDlp fetches metadata for non-YouTube platforms
// (SoundCloud, ...) directly with yt-dlp, without cookies or proxy.
func getGenericInfoWithYtDlp(rawURL, platform string, playlist bool) (utils.PlatformTracks, error) {
	if strings.TrimSpace(rawURL) == "" {
		return utils.PlatformTracks{}, errors.New("empty media URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpInfoTimeout)
	defer cancel()

	return runInfoYtDlp(ctx, rawURL, platform, playlist, "", "")
}

func runInfoYtDlp(ctx context.Context, rawURL, platform string, playlist bool, cookieFile, proxy string) (utils.PlatformTracks, error) {
	args := []string{
		"--dump-single-json",
		"--no-warnings",
		"--quiet",
	}
	if playlist {
		args = append(args, "--flat-playlist")
	}
	if cookieFile != "" {
		args = append(args, "--cookies", cookieFile)
	}
	if proxy != "" {
		args = append(args, "--proxy", proxy)
	}
	args = append(args, rawURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return utils.PlatformTracks{}, fmt.Errorf("yt-dlp info failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return utils.PlatformTracks{}, errors.New("yt-dlp info timed out")
		}
		return utils.PlatformTracks{}, err
	}

	var info ytdlpInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("decode yt-dlp info: %w", err)
	}

	if len(info.Entries) > 0 {
		tracks := make([]utils.MusicTrack, 0, len(info.Entries))
		for _, entry := range info.Entries {
			if track := entry.toMusicTrackForPlatform(platform); track.Id != "" {
				tracks = append(tracks, track)
			}
		}
		return utils.PlatformTracks{Results: tracks}, nil
	}

	track := ytdlpEntry{
		ID:         info.ID,
		Title:      info.Title,
		URL:        info.URL,
		WebpageURL: info.WebpageURL,
		Thumbnail:  info.Thumbnail,
		Duration:   info.Duration,
		Channel:    info.Channel,
		Uploader:   info.Uploader,
		ViewCount:  info.ViewCount,
	}.toMusicTrackForPlatform(platform)
	if track.Id == "" {
		return utils.PlatformTracks{}, errors.New("yt-dlp returned no media id")
	}
	return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
}

func (e ytdlpEntry) toMusicTrackForPlatform(platform string) utils.MusicTrack {
	id := strings.TrimSpace(e.ID)
	if id == "" {
		id = extractVideoID(e.URL)
	}

	title := strings.TrimSpace(e.Title)
	if title == "" {
		title = id
	}

	url := strings.TrimSpace(e.WebpageURL)
	if url == "" || !strings.HasPrefix(url, "http") {
		if platform == utils.YouTube {
			url = ytWatchURL + id
		} else if strings.HasPrefix(e.URL, "http") {
			url = e.URL
		} else {
			url = id
		}
	}

	channel := e.Channel
	if channel == "" {
		channel = e.Uploader
	}

	var views string
	if e.ViewCount > 0 {
		views = strconv.FormatInt(e.ViewCount, 10)
	}

	return utils.MusicTrack{
		Id:        id,
		Title:     title,
		Url:       url,
		Thumbnail: e.Thumbnail,
		Duration:  int(e.Duration),
		Channel:   channel,
		Views:     views,
		Platform:  platform,
	}
}

// downloadGenericWithYtDlp downloads audio for non-YouTube platforms directly
// with yt-dlp (SoundCloud fallback when the external API is down).
func downloadGenericWithYtDlp(rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", errors.New("empty media URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpDownloadTimeout)
	defer cancel()

	outputTemplate := filepath.Join(config.DownloadsDir, "%(id)s.%(ext)s")
	args := []string{
		"--no-warnings",
		"--quiet",
		"--no-playlist",
		"--retries", "2",
		"--socket-timeout", "10",
		"-f", "bestaudio/best",
		"-o", outputTemplate,
		rawURL,
		"--print", "after_move:filepath",
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("yt-dlp failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("yt-dlp timed out")
		}
		return "", err
	}

	downloadedPath := strings.TrimSpace(string(output))
	if downloadedPath == "" {
		return "", errors.New("yt-dlp returned no output path")
	}
	if _, err := os.Stat(downloadedPath); err != nil {
		return "", fmt.Errorf("downloaded file not found: %w", err)
	}
	return downloadedPath, nil
}
