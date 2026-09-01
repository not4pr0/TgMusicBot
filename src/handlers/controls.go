package handlers

import (
	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/utils"
	"ashokshau/tgmusic/src/vc"
	"fmt"
	td "github.com/AshokShau/gotdbot"
	"html"
	"strconv"
	"strings"
)

const queuePageSize = 5

func handleAdvancedPlayControl(c *td.Client, cb *td.UpdateNewCallbackQuery, data string) (bool, error) {
	chatID := cb.ChatId
	switch {
	case data == "play_seek_back":
		return true, seekByOffset(c, cb, -10)
	case data == "play_seek_forward":
		return true, seekByOffset(c, cb, 10)
	case data == "play_queue":
		return true, showQueuePanel(c, cb, 0)
	case data == "play_controls":
		return true, showPlayerControls(c, cb, "▶ Now Playing")
	case data == "play_loop":
		_, err := cb.EditMessageText(c, "<b>Loop current track</b>", &td.EditTextMessageOpts{ReplyMarkup: core.LoopMarkup(), ParseMode: "HTML"})
		return true, err
	case strings.HasPrefix(data, "play_loop_"):
		loop, err := strconv.Atoi(strings.TrimPrefix(data, "play_loop_"))
		if err != nil || loop < 0 || loop > 10 {
			_ = cb.Answer(c, 0, false, "Invalid loop count.", "")
			return true, nil
		}
		if !cache.ChatCache.SetLoopCount(chatID, loop) {
			_ = cb.Answer(c, 0, false, "No active track.", "")
			return true, nil
		}
		_ = cb.Answer(c, 0, false, fmt.Sprintf("Loop set to %d.", loop), "")
		return true, showPlayerControls(c, cb, fmt.Sprintf("🔁 Loop: %d", loop))
	case strings.HasPrefix(data, "play_qpage_"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "play_qpage_"))
		if err != nil {
			page = 0
		}
		return true, showQueuePanel(c, cb, page)
	case data == "play_qshuffle":
		if cache.ChatCache.ShuffleUpcoming(chatID) < 2 {
			_ = cb.Answer(c, 0, false, "Need at least two queued tracks.", "")
		} else {
			_ = cb.Answer(c, 0, false, "Queue shuffled.", "")
		}
		return true, showQueuePanel(c, cb, 0)
	case data == "play_qclear":
		removed := cache.ChatCache.ClearUpcoming(chatID)
		_ = cb.Answer(c, 0, false, fmt.Sprintf("Removed %d queued tracks.", removed), "")
		return true, showQueuePanel(c, cb, 0)
	case strings.HasPrefix(data, "play_qnow_"):
		id := strings.TrimPrefix(data, "play_qnow_")
		if !cache.ChatCache.MoveTrackToFront(chatID, id) {
			_ = cb.Answer(c, 0, false, "Track not found in queue.", "")
			return true, nil
		}
		if err := vc.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to play the track.", "")
			return true, err
		}
		_ = cb.Answer(c, 0, false, "Playing now.", "")
		return true, showPlayerControls(c, cb, "▶ Now Playing")
	case strings.HasPrefix(data, "play_qrm_"):
		id := strings.TrimPrefix(data, "play_qrm_")
		if !cache.ChatCache.RemoveQueuedTrackByID(chatID, id) {
			_ = cb.Answer(c, 0, false, "Only queued tracks can be removed.", "")
			return true, nil
		}
		_ = cb.Answer(c, 0, false, "Track removed from queue.", "")
		return true, showQueuePanel(c, cb, 0)
	}
	return false, nil
}

func seekByOffset(c *td.Client, cb *td.UpdateNewCallbackQuery, offset int) error {
	track := cache.ChatCache.GetPlayingTrack(cb.ChatId)
	if track == nil || track.Duration <= 0 {
		_ = cb.Answer(c, 0, false, "No seekable track.", "")
		return nil
	}
	played, err := vc.Calls.PlayedTime(cb.ChatId)
	if err != nil {
		_ = cb.Answer(c, 0, false, "Unable to read playback position.", "")
		return nil
	}
	target := int(played) + offset
	if target < 0 {
		target = 0
	}
	if target >= track.Duration {
		target = track.Duration - 1
	}
	if err = vc.Calls.SeekStream(c, cb.ChatId, track.FilePath, target, track.Duration, track.IsVideo); err != nil {
		_ = cb.Answer(c, 0, false, "Unable to seek.", "")
		return err
	}
	_ = cb.Answer(c, 0, false, fmt.Sprintf("Seeked to %s.", utils.SecToMin(target)), "")
	return showPlayerControls(c, cb, "▶ Now Playing")
}

func showPlayerControls(c *td.Client, cb *td.UpdateNewCallbackQuery, status string) error {
	track := cache.ChatCache.GetPlayingTrack(cb.ChatId)
	if track == nil {
		_, err := cb.EditMessageText(c, "There is no active playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
		return err
	}
	text := fmt.Sprintf("%s\n\n<b>Track:</b> <a href='%s'>%s</a>\n<b>Duration:</b> %s\n<b>Requested by:</b> %s", status, html.EscapeString(track.URL), html.EscapeString(track.Name), utils.SecToMin(track.Duration), html.EscapeString(track.User))
	_, err := cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons("play"), ParseMode: "HTML", DisableWebPagePreview: true})
	return err
}

func showQueuePanel(c *td.Client, cb *td.UpdateNewCallbackQuery, page int) error {
	queue := cache.ChatCache.GetQueue(cb.ChatId)
	if len(queue) == 0 {
		_, err := cb.EditMessageText(c, "The queue is empty.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML"})
		return err
	}
	upcoming := len(queue) - 1
	pages := (upcoming + queuePageSize - 1) / queuePageSize
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	current := queue[0]
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>📋 Queue</b>\n\n▶ <code>%s</code> — %s\n", html.EscapeString(truncate(current.Name, 40)), utils.SecToMin(current.Duration)))
	if upcoming == 0 {
		b.WriteString("\nNo upcoming tracks.")
	} else {
		start := 1 + page*queuePageSize
		end := start + queuePageSize
		if end > len(queue) {
			end = len(queue)
		}
		b.WriteString(fmt.Sprintf("\n<b>Next up — page %d/%d:</b>\n", page+1, pages))
		for i := start; i < end; i++ {
			b.WriteString(fmt.Sprintf("%d. <code>%s</code> — %s\n", i, html.EscapeString(truncate(queue[i].Name, 38)), utils.SecToMin(queue[i].Duration)))
		}
	}
	b.WriteString(fmt.Sprintf("\n<b>Total:</b> %d tracks", len(queue)))
	_, err := cb.EditMessageText(c, b.String(), &td.EditTextMessageOpts{ReplyMarkup: core.QueueControlMarkup(queue, page, queuePageSize), ParseMode: "HTML", DisableWebPagePreview: true})
	return err
}
