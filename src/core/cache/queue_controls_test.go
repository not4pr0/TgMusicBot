package cache

import (
	"ashokshau/tgmusic/src/utils"
	"testing"
)

func TestClearUpcomingKeepsCurrentTrack(t *testing.T) {
	c := newCache()
	c.AddSongs(1, []*utils.CachedTrack{makeTrack("current", "Current"), makeTrack("next", "Next"), makeTrack("later", "Later")})

	if removed := c.ClearUpcoming(1); removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	queue := c.GetQueue(1)
	if len(queue) != 1 || queue[0].TrackID != "current" {
		t.Fatalf("unexpected queue after clear: %#v", queue)
	}
}

func TestRemoveQueuedTrackByIDNeverRemovesCurrent(t *testing.T) {
	c := newCache()
	c.AddSongs(1, []*utils.CachedTrack{makeTrack("current", "Current"), makeTrack("next", "Next")})

	if c.RemoveQueuedTrackByID(1, "current") {
		t.Fatal("current track must not be removable through queue controls")
	}
	if !c.RemoveQueuedTrackByID(1, "next") {
		t.Fatal("problem")
	}
}
