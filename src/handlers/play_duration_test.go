package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"testing"
)

// Direct YouTube URLs initially carry no provider duration. Once their media
// download finishes, the playback state must be populated from the local file
// so UI duration and /seek operate on the actual track length.
func TestHydrateDownloadedTrackDuration(t *testing.T) {
	track := &utils.CachedTrack{FilePath: "/tmp/CVK8i5pFySw.mp4"}
	probeCalled := false

	hydrateDownloadedTrackDuration(track, func(path string) int {
		probeCalled = true
		if path != track.FilePath {
			t.Fatalf("probe received %q, want %q", path, track.FilePath)
		}
		return 3105
	})

	if !probeCalled {
		t.Fatal("the downloaded media must be probed when duration is missing")
	}
	if track.Duration != 3105 {
		t.Fatalf("duration = %d, want 3105", track.Duration)
	}
}
