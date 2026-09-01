/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"fmt"

	"github.com/AshokShau/gotdbot"
)

func cb(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
		Style: style,
	}
}

func userId(text string, userId int64, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text:  text,
		Type:  &gotdbot.InlineKeyboardButtonTypeUser{UserId: userId},
		Style: style,
	}
}

func url(text, link string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
		Style: style,
	}
}

var CloseBtn = cb("Close", "vcplay_close", gotdbot.ButtonStyleDanger{})
var HomeBtn = cb("Home", "help_back", gotdbot.ButtonStylePrimary{})
var HelpBtn = cb("Help", "help_all", gotdbot.ButtonStyleDefault{})
var UserBtn = cb("Users", "help_user", gotdbot.ButtonStyleDefault{})
var AdminBtn = cb("Admins", "help_admin", gotdbot.ButtonStyleDefault{})
var OwnerBtn = cb("Owner", "help_owner", gotdbot.ButtonStyleDefault{})
var DevsBtn = cb("Devs", "help_devs", gotdbot.ButtonStyleDefault{})
var PlaylistBtn = cb("Playlist", "help_playlist", gotdbot.ButtonStyleDefault{})
var AutoplayBtn = cb("Autoplay", "help_autoplay", gotdbot.ButtonStyleDefault{})

var SourceCodeBtn = url("Source Code", "https://github.com/AshokShau/TgMusicBot", gotdbot.ButtonStylePrimary{})
var channelBtn = url("Updates", config.SupportChannel, gotdbot.ButtonStyleDefault{})
var groupBtn = url("Group", config.SupportGroup, gotdbot.ButtonStyleDefault{})

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string, cmdDelete bool, language string) *gotdbot.ReplyMarkupInlineKeyboard {
	playText := "Everyone"
	if playMode == utils.Admins {
		playText = "Admins"
	}

	deleteText := "False"
	if cmdDelete {
		deleteText = "True"
	}

	adminText := "Everyone"
	if adminMode == utils.Admins {
		adminText = "Admins"
	}

	langText := "English"
	if language != "en" && language != "" {
		langText = language
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{
				cb("Play Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(playText, "settings_play", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Command Delete ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(deleteText, "settings_delete", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Admin Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(adminText, "settings_admin", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Language ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(langText, "settings_lang", gotdbot.ButtonStyleDefault{}),
			},
			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{UserBtn, AdminBtn, OwnerBtn},
			{PlaylistBtn, DevsBtn, AutoplayBtn},
			{HomeBtn, CloseBtn},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{HelpBtn, HomeBtn},
			{CloseBtn, SourceCodeBtn},
		},
	}
}

func ControlButtons(mode string) *gotdbot.ReplyMarkupInlineKeyboard {
	back := cb("⏪ 10s", "play_seek_back", gotdbot.ButtonStyleDefault{})
	forward := cb("10s ⏩", "play_seek_forward", gotdbot.ButtonStyleDefault{})
	skip := cb("⏭", "play_skip", gotdbot.ButtonStyleDefault{})
	stop := cb("⏹", "play_stop", gotdbot.ButtonStyleDefault{})
	pause := cb("⏸", "play_pause", gotdbot.ButtonStyleDefault{})
	resume := cb("▶", "play_resume", gotdbot.ButtonStyleDefault{})
	mute := cb("🔇", "play_mute", gotdbot.ButtonStyleDefault{})
	unmute := cb("🔊", "play_unmute", gotdbot.ButtonStyleDefault{})
	queue := cb("📋 Queue", "play_queue", gotdbot.ButtonStylePrimary{})
	loop := cb("🔁 Loop", "play_loop", gotdbot.ButtonStyleDefault{})
	playlist := cb("➕", "play_add_to_list", gotdbot.ButtonStylePrimary{})
	main := [][]gotdbot.InlineKeyboardButton{{back, pause, forward}, {skip, stop, mute}, {queue, loop, playlist}, {CloseBtn}}
	if mode == "pause" {
		main[0][1] = resume
	}
	if mode == "mute" {
		main[1][2] = unmute
	}
	return &gotdbot.ReplyMarkupInlineKeyboard{Rows: main}
}

func LoopMarkup() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{Rows: [][]gotdbot.InlineKeyboardButton{{cb("Off", "play_loop_0", gotdbot.ButtonStyleDefault{}), cb("1", "play_loop_1", gotdbot.ButtonStyleDefault{}), cb("2", "play_loop_2", gotdbot.ButtonStyleDefault{}), cb("5", "play_loop_5", gotdbot.ButtonStyleDefault{}), cb("10", "play_loop_10", gotdbot.ButtonStyleDefault{})}, {cb("↩ Player", "play_controls", gotdbot.ButtonStylePrimary{}), CloseBtn}}}
}

func QueueControlMarkup(queue []*utils.CachedTrack, page, pageSize int) *gotdbot.ReplyMarkupInlineKeyboard {
	rows := make([][]gotdbot.InlineKeyboardButton, 0)
	start := 1 + page*pageSize
	end := start + pageSize
	if end > len(queue) {
		end = len(queue)
	}
	for i := start; i < end; i++ {
		rows = append(rows, []gotdbot.InlineKeyboardButton{cb(fmt.Sprintf("▶ %d", i), "play_qnow_"+queue[i].TrackID, gotdbot.ButtonStylePrimary{}), cb(fmt.Sprintf("✖ %d", i), "play_qrm_"+queue[i].TrackID, gotdbot.ButtonStyleDanger{})})
	}
	pages := (len(queue) - 1 + pageSize - 1) / pageSize
	if pages > 1 {
		prev := page - 1
		if prev < 0 {
			prev = 0
		}
		next := page + 1
		if next >= pages {
			next = pages - 1
		}
		rows = append(rows, []gotdbot.InlineKeyboardButton{cb("‹", fmt.Sprintf("play_qpage_%d", prev), gotdbot.ButtonStyleDefault{}), cb(fmt.Sprintf("%d/%d", page+1, pages), fmt.Sprintf("play_qpage_%d", page), gotdbot.ButtonStyleDefault{}), cb("›", fmt.Sprintf("play_qpage_%d", next), gotdbot.ButtonStyleDefault{})})
	}
	rows = append(rows, []gotdbot.InlineKeyboardButton{cb("🔀", "play_qshuffle", gotdbot.ButtonStyleDefault{}), cb("🗑 Clear", "play_qclear", gotdbot.ButtonStyleDanger{}), cb("↩ Player", "play_controls", gotdbot.ButtonStylePrimary{})})
	return &gotdbot.ReplyMarkupInlineKeyboard{Rows: rows}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {

	addMeBtn := url(
		"Aᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ɢʀᴏᴜᴘ",
		fmt.Sprintf("https://t.me/%s?startgroup=true", username),
		gotdbot.ButtonStylePrimary{},
	)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{HelpBtn},
			{channelBtn, groupBtn},
			{SourceCodeBtn},
		},
	}
}

func PlayNowButton(trackID string) gotdbot.InlineKeyboardButton {
	return cb("Play Now", fmt.Sprintf("play_now_%s", trackID), gotdbot.ButtonStyleDanger{})
}

func QueueMarkup(trackID string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{PlayNowButton(trackID), CloseBtn},
		},
	}
}
