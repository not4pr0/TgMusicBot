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
	"os/exec"
	"path/filepath"
	"strings"
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

func (y *YouTubeData) getInfoWithYtDlp(ctx context.Context, rawURL string, playlist bool) (utils.PlatformTracks, error) {
	if strings.TrimSpace(rawURL) == "" {
		return utils.PlatformTracks{}, errors.New("empty YouTube URL")
	}

	cookieFiles := y.getCookieFiles()
	if len(cookieFiles) == 0 {
		return y.runInfoYtDlp(ctx, rawURL, playlist, "")
	}

	var attempts []string
	for _, cookieFile := range cookieFiles {
		tracks, err := y.runInfoYtDlp(ctx, rawURL, playlist, cookieFile)
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

	tracks, err := y.runInfoYtDlp(ctx, rawURL, playlist, "")
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

func (y *YouTubeData) runInfoYtDlp(ctx context.Context, rawURL string, playlist bool, cookieFile string) (utils.PlatformTracks, error) {
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
	if config.Conf.Proxy != "" {
		args = append(args, "--proxy", config.Conf.Proxy)
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
			return utils.PlatformTracks{}, fmt.Errorf("yt-dlp info timed out")
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
			if track := entry.toMusicTrackForPlatform(utils.YouTube); track.Id != "" {
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
	}.toMusicTrackForPlatform(utils.YouTube)
	if track.Id == "" {
		return utils.PlatformTracks{}, errors.New("yt-dlp returned no video id")
	}
	return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
}

func (e ytdlpEntry) toMusicTrack() utils.MusicTrack {
	return e.toMusicTrackForPlatform(utils.YouTube)
}

func (e ytdlpEntry) toMusicTrackForPlatform(platform string) utils.MusicTrack {
	id := strings.TrimSpace(e.ID)
	if id == "" {
		id = extractVideoID(e.URL)
	}
	if id == "" {
		id = extractVideoID(e.WebpageURL)
	}
	if id == "" {
		return utils.MusicTrack{}
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

	views := ""
	if e.ViewCount > 0 {
		views = fmt.Sprintf("%d", e.ViewCount)
	}

	return utils.MusicTrack{
		Id:        id,
		Title:     e.Title,
		Url:       url,
		Thumbnail: e.Thumbnail,
		Duration:  int(e.Duration),
		Channel:   channel,
		Views:     views,
		Platform:  platform,
	}
}
