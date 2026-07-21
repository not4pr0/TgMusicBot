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
const youtubeProxyHelp = "❌ YouTube proxy is unreachable. Check Tailscale/tinyproxy on the Proxmox node, then retry."

// UserFacingError converts internal downloader errors into short, actionable
// messages for chat, instead of dumping raw yt-dlp stderr on users.
func UserFacingError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	if isProxyConnectionError(msg) {
		return youtubeProxyHelp
	}
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

// isProxyConnectionError detects failures to reach the PROXY endpoint itself
// (Tailscale 100.x range), as opposed to YouTube rejecting the request.
func isProxyConnectionError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "unable to connect to proxy") ||
		strings.Contains(lower, "proxyerror") ||
		strings.Contains(lower, "connecttimeout") ||
		strings.Contains(lower, "connection to 100.") && strings.Contains(lower, "timed out")
}
