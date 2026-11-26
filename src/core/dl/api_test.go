package dl

import (
	"ashokshau/tgmusic/src/utils"
	"testing"
)

func TestSoundCloudWorksWithoutExternalAPI(t *testing.T) {
	api := &apiData{
		Query:    "https://soundcloud.com/example/track",
		Patterns: apiPatterns,
	}

	if !api.isValid() {
		t.Fatal("SoundCloud URL should be supported without an external API configuration")
	}

	track, err := api.getTrack()
	if err != nil {
		t.Fatalf("getTrack returned an error: %v", err)
	}
	if track.URL != api.Query || track.Platform != utils.SoundCloud || track.CdnURL != "" {
		t.Fatalf("unexpected SoundCloud fallback track: %#v", track)
	}
}

func TestOtherAPIServicesStillRequireConfiguration(t *testing.T) {
	api := &apiData{
		Query:    "https://open.spotify.com/track/1234567890",
		Patterns: apiPatterns,
	}

	if api.isValid() {
		t.Fatal("Spotify URL should still require an external API configuration")
	}
}
