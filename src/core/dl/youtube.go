/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import (
	"time"

	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// youTubeData provides an interface for fetching track and playlist information from YouTube.
type youTubeData struct {
	Query    string
	ApiUrl   string
	APIKey   string
	Patterns map[string]*regexp.Regexp
}

var youtubePatterns = map[string]*regexp.Regexp{
	"youtube":   regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/.*`),
	"youtu_be":  regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtu\.be/.*`),
	"yt_music":  regexp.MustCompile(`(?i)^(?:https?://)?music\.youtube\.com/.*`),
	"yt_shorts": regexp.MustCompile(`(?i)^(?:https?://)?(?:www\.)?youtube\.com/shorts/.*`),
}

// newYouTubeData initializes a youTubeData instance with pre-compiled regex patterns and a cleaned query.
func newYouTubeData(query string) *youTubeData {
	return &youTubeData{
		Query:    strings.TrimSpace(query),
		ApiUrl:   strings.TrimRight(config.ApiUrl, "/"),
		APIKey:   config.ApiKey,
		Patterns: youtubePatterns,
	}
}

func (y *youTubeData) isValid() bool {
	if y.Query == "" {
		slog.Info("The query or patterns are empty.")
		return false
	}

	for _, pattern := range y.Patterns {
		if pattern.MatchString(y.Query) {
			return true
		}
	}
	return false
}

func (y *youTubeData) getInfo() (utils.PlatformTracks, error) {
	if !y.isValid() {
		return utils.PlatformTracks{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	y.Query = normalizeYouTubeURL(y.Query)
	videoID := extractVideoID(y.Query)
	playlistID := extractPlaylistID(y.Query)

	switch {
	case playlistID != "":
		var tracks utils.PlatformTracks
		var err error
		if strings.HasPrefix(playlistID, "RD") {
			tracks, err = GetYouTubeMixPlaylist(ctx, playlistID)
		} else {
			tracks, err = getYouTubePlaylist(ctx, playlistID)
		}
		if err == nil && len(tracks.Results) > 0 {
			return tracks, nil
		}
		if err != nil {
			slog.Warn("YouTube playlist InnerTube failed, falling back to yt-dlp", "playlist", playlistID, "error", err)
		} else {
			slog.Warn("YouTube playlist InnerTube returned no tracks, falling back to yt-dlp", "playlist", playlistID)
		}
		return y.getInfoWithYtDlp(y.Query, true)

	case videoID != "":
		for _, query := range []string{videoID, y.Query} {
			tracks, err := searchYouTube(query, 10)
			if err != nil {
				continue
			}

			for _, track := range tracks {
				if track.Id == videoID {
					return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
				}
			}
		}

		if title, err := getYouTubeTitleFromOEmbed(videoID); err == nil && title != "" {
			tracks, err := searchYouTube(title, 10)
			if err == nil {
				for _, track := range tracks {
					if track.Id == videoID {
						return utils.PlatformTracks{Results: []utils.MusicTrack{track}}, nil
					}
				}
			}
		}

		slog.Warn("Video ID was extracted but no matching track was found in search results", "video_id", videoID)
		tracks, err := getYouTubeVideo(ctx, videoID)
		if err == nil && len(tracks.Results) > 0 {
			return tracks, nil
		}
		if err != nil {
			slog.Warn("YouTube video InnerTube failed, using minimal metadata to avoid a yt-dlp probe", "video", videoID, "error", err)
		} else {
			slog.Warn("YouTube video InnerTube returned no tracks, using minimal metadata to avoid a yt-dlp probe", "video", videoID)
		}
		return utils.PlatformTracks{Results: []utils.MusicTrack{{
			Id:       videoID,
			Title:    "YouTube video " + videoID,
			Url:      ytWatchURL + videoID,
			Platform: utils.YouTube,
		}}}, nil
	}

	return utils.PlatformTracks{}, errors.New("no video or playlist results were found")
}

func (y *youTubeData) search() (utils.PlatformTracks, error) {
	tracks, err := searchYouTube(y.Query, 5)
	if err != nil {
		return utils.PlatformTracks{}, err
	}

	if len(tracks) == 0 {
		return utils.PlatformTracks{}, errors.New("no video results were found")
	}

	return utils.PlatformTracks{Results: tracks}, nil
}

func (y *youTubeData) getTrack() (utils.TrackInfo, error) {
	if y.Query == "" {
		return utils.TrackInfo{}, errors.New("the query is empty")
	}

	if !y.isValid() {
		return utils.TrackInfo{}, errors.New("the provided URL is invalid or the platform is not supported")
	}

	if y.ApiUrl != "" && y.APIKey != "" {
		if trackInfo, err := newApiData(y.Query).getTrack(); err == nil {
			return trackInfo, nil
		}
	}

	getInfo, err := y.getInfo()
	if err != nil {
		return utils.TrackInfo{}, err
	}
	if len(getInfo.Results) == 0 {
		return utils.TrackInfo{}, errors.New("no video results were found")
	}

	track := getInfo.Results[0]
	trackInfo := utils.TrackInfo{
		Id:       track.Id,
		URL:      track.Url,
		Platform: utils.YouTube,
	}

	return trackInfo, nil
}

// downloadTrack handles the download of a track from YouTube.
func (y *youTubeData) downloadTrack(info utils.TrackInfo, video bool) (string, error) {
	if !video && info.CdnURL != "" {
		return info.CdnURL, nil
	}

	if !video && y.ApiUrl != "" && y.APIKey != "" {
		if filePath, err := y.downloadWithApi(info.Id, video); err == nil {
			return filePath, nil
		}
	}

	return y.downloadWithYtDlp(info.Id, video)
}

// buildYtdlpParams constructs the command-line parameters for yt-dlp to download media.
func (y *youTubeData) buildYtdlpParams(videoID string, video bool, cookieFile string) []string {
	outputTemplate := filepath.Join(config.DownloadsDir, "%(id)s.%(ext)s")

	params := []string{
		"yt-dlp",
		"--no-warnings",
		"--quiet",
		"--geo-bypass",
		"--retries", "2",
		"--continue",
		"--no-part",
		"--concurrent-fragments", "3",
		"--socket-timeout", "10",
		"--throttled-rate", "100K",
		"--retry-sleep", "1",
		"--no-write-thumbnail",
		"--no-write-info-json",
		"--no-embed-metadata",
		"--no-embed-chapters",
		"--no-embed-subs",
		"--extractor-args", "youtube:player_js_version=actual",
		"-o", outputTemplate,
	}

	if video {
		formatSelector := "bestvideo[height<=720]+bestaudio/best[height<=720]"
		params = append(params, "-f", formatSelector, "--merge-output-format", "mp4")
	} else {
		params = append(params, "-f", "bestaudio[ext=m4a]/bestaudio")
	}

	if cookieFile != "" {
		params = append(params, "--cookies", cookieFile)
	}
	// The proxy (home connection via Tailscale) hides the VPS IP from YouTube,
	// so it is used together with cookies, not as a substitute.
	if config.Proxy != "" {
		params = append(params, "--proxy", config.Proxy)
	}

	videoURL := "https://www.youtube.com/watch?v=" + videoID
	params = append(params, videoURL, "--print", "after_move:filepath")

	return params
}

// downloadWithYtDlp downloads media from YouTube using the yt-dlp command-line tool.
// Every configured cookie file is tried in order before a last cookieless
// attempt, so one expired export does not take the bot down.
func (y *youTubeData) downloadWithYtDlp(videoID string, video bool) (string, error) {
	if videoID == "" {
		return "", errors.New("videoID is empty")
	}

	cookieFiles := config.GetCookiesPath()
	if len(cookieFiles) == 0 {
		return y.runYtDlp(videoID, video, "")
	}

	var attempts []string
	for _, cookieFile := range cookieFiles {
		filePath, err := y.runYtDlp(videoID, video, cookieFile)
		if err == nil {
			return filePath, nil
		}
		if isProxyConnectionError(err.Error()) {
			return "", fmt.Errorf("yt-dlp proxy failed with %s: %w", filepath.Base(cookieFile), err)
		}
		attempts = append(attempts, fmt.Sprintf("%s: %s", filepath.Base(cookieFile), err.Error()))
		slog.Warn("yt-dlp failed with cookie file, trying next one", "cookie", filepath.Base(cookieFile), "error", err)
	}

	if filePath, err := y.runYtDlp(videoID, video, ""); err == nil {
		slog.Warn("yt-dlp succeeded without cookies after all configured cookie files failed")
		return filePath, nil
	} else {
		if isProxyConnectionError(err.Error()) {
			return "", fmt.Errorf("yt-dlp proxy failed without cookies: %w", err)
		}
		attempts = append(attempts, "without cookies: "+err.Error())
	}

	return "", fmt.Errorf("yt-dlp failed with all configured cookie files: %s", strings.Join(attempts, " | "))
}

func (y *youTubeData) runYtDlp(videoID string, video bool, cookieFile string) (string, error) {
	ytdlpParams := y.buildYtdlpParams(videoID, video, cookieFile)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlpParams[0], ytdlpParams[1:]...)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			// Only delete cookie files the bot itself downloaded from COOKIES_URL;
			// files mounted via COOKIES_PATH/COOKIES_DIR belong to the operator.
			if cookieFile != "" && config.IsManagedCookiePath(cookieFile) && strings.Contains(stderr, "Sign in to confirm you're not a bot") {
				_ = os.Remove(cookieFile)
			}
			return "", fmt.Errorf("yt-dlp failed with exit code %d: %s", exitErr.ExitCode(), stderr)
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("yt-dlp timed out for video ID: %s", videoID)
		}

		return "", fmt.Errorf("an unexpected error occurred while downloading %s: %w", videoID, err)
	}

	downloadedPathStr := strings.TrimSpace(string(output))
	if downloadedPathStr == "" {
		return "", fmt.Errorf("no output path was returned for %s", videoID)
	}

	if _, err := os.Stat(downloadedPathStr); os.IsNotExist(err) {
		return "", fmt.Errorf("the file was not found at the reported path: %s", downloadedPathStr)
	}

	return downloadedPathStr, nil
}

// downloadWithApi downloads a track using the external API.
func (y *youTubeData) downloadWithApi(videoID string, _ bool) (string, error) {
	videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	api := newApiData(videoUrl)
	track, err := api.getTrack()
	if err != nil {
		return "", err
	}

	down, err := newDownload(track)
	if err != nil {
		slog.Info("Error creating download: " + err.Error())
		return "", err
	}

	return down.Process()
}
