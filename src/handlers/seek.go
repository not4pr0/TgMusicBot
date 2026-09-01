package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"strconv"
	"strings"

	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/vc"
	td "github.com/AshokShau/gotdbot"
)

// parseSeekArgument accepts legacy relative seconds and absolute mm:ss or hh:mm:ss.
func parseSeekArgument(input string) (seconds int, absolute bool, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, false, fmt.Errorf("empty seek value")
	}
	if !strings.Contains(input, ":") {
		seconds, err = strconv.Atoi(input)
		if err != nil || seconds < 0 {
			return 0, false, fmt.Errorf("invalid seconds")
		}
		return seconds, false, nil
	}
	parts := strings.Split(input, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, false, fmt.Errorf("use mm:ss or hh:mm:ss")
	}
	values := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return 0, false, fmt.Errorf("empty time component")
		}
		values[i], err = strconv.Atoi(part)
		if err != nil || values[i] < 0 {
			return 0, false, fmt.Errorf("invalid time component")
		}
	}
	if values[len(values)-1] >= 60 || (len(values) == 3 && values[1] >= 60) {
		return 0, false, fmt.Errorf("seconds and minutes must be below 60")
	}
	if len(values) == 2 {
		return values[0]*60 + values[1], true, nil
	}
	return values[0]*3600 + values[1]*60 + values[2], true, nil
}

func seekHandler(c *td.Client, m *td.Message) error {
	if !adminMode(c, m) {
		return td.EndGroups
	}
	chatID := m.ChatId
	if !cache.ChatCache.IsActive(chatID) {
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}
	playingSong := cache.ChatCache.GetPlayingTrack(chatID)
	if playingSong == nil {
		_, err := m.ReplyText(c, "The bot is not streaming in the video chat.", nil)
		return err
	}
	args := Args(m)
	if args == "" {
		_, _ = m.ReplyText(c, "<b>Usage:</b> <code>/seek +seconds</code> or <code>/seek mm:ss</code>\n<b>Examples:</b> <code>/seek 15</code>, <code>/seek 25:00</code>", replyOpts)
		return nil
	}
	value, absolute, err := parseSeekArgument(args)
	if err != nil {
		_, _ = m.ReplyText(c, "Invalid seek value. Use seconds, mm:ss, or hh:mm:ss.", nil)
		return nil
	}
	if !absolute && value < 10 {
		_, _ = m.ReplyText(c, "Minimum relative seek time is 10 seconds.", nil)
		return nil
	}
	currDur, err := vc.Calls.PlayedTime(chatID)
	if err != nil {
		_, _ = m.ReplyText(c, "Failed to fetch the duration of the ongoing stream.", nil)
		return nil
	}
	toSeek := value
	if !absolute {
		toSeek = int(currDur) + value
	}
	if toSeek < 0 || toSeek >= playingSong.Duration {
		_, _ = m.ReplyText(c, fmt.Sprintf("You cannot seek beyond the track duration. Maximum allowed is %s.", utils.SecToMin(playingSong.Duration)), nil)
		return nil
	}
	if err = vc.Calls.SeekStream(c, chatID, playingSong.FilePath, toSeek, playingSong.Duration, playingSong.IsVideo); err != nil {
		_, _ = m.ReplyText(c, fmt.Sprintf("An error occurred while seeking the track: %s", err.Error()), replyOpts)
		return nil
	}
	if absolute {
		_, _ = m.ReplyText(c, fmt.Sprintf("<b>Stream moved to %s by</b> %s", utils.SecToMin(toSeek), firstName(c, m)), replyOpts)
	} else {
		_, _ = m.ReplyText(c, fmt.Sprintf("<b>Stream skipped %s and started from %s seconds by</b> %s", utils.SecToMin(value), utils.SecToMin(toSeek), firstName(c, m)), replyOpts)
	}
	return nil
}
