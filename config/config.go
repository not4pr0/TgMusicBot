/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Conf is the global configuration for the bot.
var Conf *BotConfig

// LoadConfig loads the configuration from environment variables and sets the global Conf.
func LoadConfig() error {
	envFiles := []string{".env.local", ".env"}
	if err := loadEnvFiles(envFiles...); err != nil {
		slog.Info("Warning loading env files", "error", err)
	}

	Conf = &BotConfig{
		ApiId:             getEnvInt32("API_ID", 0),
		ApiHash:           os.Getenv("API_HASH"),
		Token:             os.Getenv("TOKEN"),
		SessionStrings:    getSessionStrings("STRING", 10),
		SessionType:       getEnvStr("SESSION_TYPE", "pyrogram"),
		MongoUri:          os.Getenv("MONGO_URI"),
		DbName:            getEnvStr("DB_NAME", "Anon"),
		ApiUrl:            getEnvStr("API_URL", "https://beta.fallenapi.fun"),
		ApiKey:            os.Getenv("API_KEY"),
		OwnerId:           getEnvInt64("OWNER_ID"),
		LoggerId:          getEnvInt64("LOGGER_ID"),
		Proxy:             os.Getenv("PROXY"),
		DefaultService:    strings.ToLower(getEnvStr("DEFAULT_SERVICE", "youtube")),
		MaxFileSize:       getEnvInt64("MAX_FILE_SIZE"),
		SongDurationLimit: getEnvInt64("SONG_DURATION_LIMIT"),
		DownloadsDir:      getEnvStr("DOWNLOADS_DIR", "downloads"),
		SupportGroup:      getEnvStr("SUPPORT_GROUP", "https://t.me/FallenSupport"),
		SupportChannel:    getEnvStr("SUPPORT_CHANNEL", "https://t.me/FallenProjects"),
		cookiesUrl:        processCookieURLs(os.Getenv("COOKIES_URL")),
		Port:              getEnvStr("PORT", "6060"),
	}

	devsEnv := os.Getenv("DEVS")
	if devsEnv != "" {
		devsEnv = strings.ReplaceAll(devsEnv, "\n", " ")
		devsEnv = strings.ReplaceAll(devsEnv, ",", " ")

		for _, idStr := range strings.Fields(devsEnv) {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				Conf.DEVS = append(Conf.DEVS, id)
			} else {
				slog.Info("Invalid DEV ID ''", "id", idStr, "error", err)
			}
		}
	}

	if Conf.OwnerId != 0 && !containsInt(Conf.DEVS, Conf.OwnerId) {
		Conf.DEVS = append(Conf.DEVS, Conf.OwnerId)
	}

	if err := Conf.validate(); err != nil {
		return err
	}

	Conf.CookiesPath = discoverCookieFiles(
		os.Getenv("COOKIES_PATH"),
		os.Getenv("COOKIES_DIR"),
	)

	if err := os.MkdirAll(Conf.DownloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	if len(Conf.cookiesUrl) > 0 {
		if err := os.MkdirAll(cookiesDr, 0750); err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		go saveAllCookies(Conf.cookiesUrl)
	}

	return nil
}

type cookieFileCandidate struct {
	path    string
	modUnix int64
	rank    int
}

func discoverCookieFiles(pathsRaw, dirsRaw string) []string {
	seen := make(map[string]bool)
	var candidates []cookieFileCandidate

	addPath := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}

		st, err := os.Stat(path)
		if err != nil {
			slog.Info("Cookie path ignored", "path", path, "error", err)
			return
		}
		if st.IsDir() {
			addCookieDir(&candidates, seen, path)
			return
		}
		addCookieFile(&candidates, seen, path, st)
	}

	for _, raw := range strings.Split(pathsRaw, ",") {
		addPath(raw)
	}
	for _, raw := range strings.Split(dirsRaw, ",") {
		addPath(raw)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		if candidates[i].modUnix == candidates[j].modUnix {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].modUnix > candidates[j].modUnix
	})

	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.path)
	}
	if len(paths) > 0 {
		slog.Info("Loaded YouTube cookie files", "count", len(paths), "first", filepath.Base(paths[0]))
	}
	return paths
}

func addCookieDir(candidates *[]cookieFileCandidate, seen map[string]bool, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Info("Cookie directory ignored", "path", dir, "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)
		if !strings.HasSuffix(lowerName, ".txt") && !strings.Contains(lowerName, "cookie") {
			continue
		}
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err != nil {
			slog.Info("Cookie file ignored", "path", path, "error", err)
			continue
		}
		addCookieFile(candidates, seen, path, st)
	}
}

func addCookieFile(candidates *[]cookieFileCandidate, seen map[string]bool, path string, st os.FileInfo) {
	cleanPath := filepath.Clean(path)
	if seen[cleanPath] {
		return
	}
	if st.Size() <= 0 {
		slog.Info("Cookie file ignored because it is empty", "path", cleanPath)
		return
	}
	if !looksLikeCookieFile(cleanPath) {
		slog.Info("Cookie file ignored because it does not look like a Netscape cookie file", "path", cleanPath)
		return
	}

	seen[cleanPath] = true
	*candidates = append(*candidates, cookieFileCandidate{
		path:    cleanPath,
		modUnix: st.ModTime().Unix(),
		rank:    cookieFileRank(cleanPath),
	})
}

func cookieFileRank(path string) int {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasPrefix(name, "cookies-"):
		return 0
	case name == "cookies.txt":
		return 1
	case strings.Contains(name, "bkp") || strings.Contains(name, "bak") || strings.Contains(name, "old"):
		return 3
	default:
		return 2
	}
}

func looksLikeCookieFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Info("Cookie file ignored because it cannot be read", "path", path, "error", err)
		return false
	}
	if len(data) > 8192 {
		data = data[:8192]
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "netscape http cookie file") ||
		strings.Contains(content, "youtube.com") ||
		strings.Contains(content, "google.com")
}
