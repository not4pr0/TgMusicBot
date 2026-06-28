/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package dl

import "strings"

const youtubeCookieHelp = "❌ YouTube cookies expired or invalid. Export fresh cookies from a logged-in browser, install them on the VPS, then retry."

func UserFacingError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "sign in to confirm") ||
		strings.Contains(lower, "not a bot") ||
		strings.Contains(lower, "configured cookie files") {
		return youtubeCookieHelp
	}

	if len(msg) > 500 {
		return msg[:500] + "..."
	}
	return msg
}
