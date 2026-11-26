package dl

import (
	"ashokshau/tgmusic/src/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type spotifyOEmbed struct {
	Title string `json:"title"`
}

var nonWord = regexp.MustCompile(`[^[:alnum:]]+`)

// getSpotifyFallback resolves a Spotify URL to a close YouTube equivalent.
// Spotify streams are DRM-protected, so yt-dlp cannot download them directly.
func getSpotifyFallback(rawURL string) (utils.PlatformTracks, error) {
	endpoint := "https://open.spotify.com/oembed?" + url.Values{"url": {rawURL}}.Encode()
	resp, err := sendRequestWithOptions(http.MethodGet, endpoint, nil, nil, apiClient, apiRequestTimeout, 1)
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("Spotify metadata unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return utils.PlatformTracks{}, fmt.Errorf("Spotify metadata returned %s", resp.Status)
	}

	var metadata spotifyOEmbed
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("invalid Spotify metadata: %w", err)
	}
	query := strings.TrimSpace(metadata.Title)
	if query == "" {
		return utils.PlatformTracks{}, errors.New("Spotify metadata has no title")
	}

	candidates, err := searchYouTube(query, 5)
	if err != nil {
		return utils.PlatformTracks{}, fmt.Errorf("YouTube fallback search failed: %w", err)
	}
	best, score := bestYouTubeMatch(query, candidates)
	if best.Id == "" || score < 0.55 {
		return utils.PlatformTracks{}, fmt.Errorf("no close YouTube match for %q (score %.2f)", query, score)
	}
	best.Platform = utils.YouTube
	return utils.PlatformTracks{Results: []utils.MusicTrack{best}}, nil
}

func bestYouTubeMatch(query string, candidates []utils.MusicTrack) (utils.MusicTrack, float64) {
	wanted := normalizedTokens(query)
	var best utils.MusicTrack
	bestScore := 0.0
	for _, candidate := range candidates {
		if candidate.Id == "" {
			continue
		}
		score := tokenOverlap(wanted, normalizedTokens(candidate.Title))
		if score > bestScore {
			best, bestScore = candidate, score
		}
	}
	return best, bestScore
}

func normalizedTokens(value string) map[string]struct{} {
	value = strings.ToLower(nonWord.ReplaceAllString(value, " "))
	result := make(map[string]struct{})
	for _, token := range strings.Fields(value) {
		if len(token) > 1 {
			result[token] = struct{}{}
		}
	}
	return result
}

func tokenOverlap(wanted, candidate map[string]struct{}) float64 {
	if len(wanted) == 0 || len(candidate) == 0 {
		return 0
	}
	matched := 0
	for token := range wanted {
		if _, ok := candidate[token]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(wanted))
}
