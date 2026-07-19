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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getGenericInfoWithYtDlp(ctx context.Context, rawURL, platform string, playlist bool) (utils.PlatformTracks, error) {
	if strings.TrimSpace(rawURL) == "" {
		return utils.PlatformTracks{}, errors.New("empty media URL")
	}

	args := []string{
		"--dump-single-json",
		"--no-warnings",
		"--quiet",
	}
	if playlist {
		args = append(args, "--flat-playlist")
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

func downloadGenericWithYtDlp(ctx context.Context, rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", errors.New("empty media URL")
	}

	outputTemplate := filepath.Join(config.Conf.DownloadsDir, "%(id)s.%(ext)s")
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
