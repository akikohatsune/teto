package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Settings struct {
	DiscordToken                string
	CommandPrefix               string
	RPCEnabled                  bool
	RPCStatus                   string
	RPCActivityType             string
	RPCActivityName             string
	RPCActivityURL              string
	NvidiaAPIKey                string
	NvidiaModel                 string
	SystemRulesMD               string
	SystemPrompt                string
	ChatReplayLogPath           string
	ChatMemoryDBPath            string
	BanDBPath                   string
	CallnamesDBPath             string
	MemoryIdleTTLSeconds        int
	ImageMaxBytes               int
	MaxReplyChars               int
	Temperature                 float64
	MaxHistory                  int
	KomiFilterEnabled           bool
	KomiFilterMaxCheckChars      int
	KomiFilterBlockResponseOnLeak bool
	OwnerID                     string
	RestartIntervalHours        int
}

func GetEnvStr(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

func GetEnvInt(name string, defaultValue int) int {
	raw := GetEnvStr(name, "")
	if raw == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("Warning: %s must be an integer, got %v. Using default: %d", name, raw, defaultValue)
		return defaultValue
	}
	return val
}

func GetEnvFloat(name string, defaultValue float64) float64 {
	raw := GetEnvStr(name, "")
	if raw == "" {
		return defaultValue
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Printf("Warning: %s must be a float, got %v. Using default: %f", name, raw, defaultValue)
		return defaultValue
	}
	return val
}

func GetEnvBool(name string, defaultValue bool) bool {
	raw := strings.ToLower(GetEnvStr(name, ""))
	if raw == "" {
		return defaultValue
	}
	return raw == "true" || raw == "1" || raw == "yes" || raw == "on"
}

func LoadSystemRulesPrompt(pathValue string) string {
	if pathValue == "" {
		return ""
	}
	content, err := os.ReadFile(pathValue)
	if err != nil {
		return ""
	}
	rules := strings.TrimSpace(string(content))
	if rules == "" {
		return ""
	}
	// Return rules directly as the primary prompt
	return rules
}

func LoadSettings() *Settings {
	_ = godotenv.Load()

	nvidiaAPIKey := GetEnvStr("NVIDIA_API_KEY", "")
	if nvidiaAPIKey == "" {
		log.Fatal("Missing NVIDIA_API_KEY in environment variables.")
	}

	discordToken := GetEnvStr("DISCORD_TOKEN", "")
	if discordToken == "" {
		log.Fatal("Missing DISCORD_TOKEN in environment variables.")
	}

	systemRulesMD := GetEnvStr("SYSTEM_RULES_MD", "system_rules.md")
	systemPrompt := LoadSystemRulesPrompt(systemRulesMD)

	return &Settings{
		DiscordToken:                discordToken,
		CommandPrefix:               GetEnvStr("COMMAND_PREFIX", "!"),
		RPCEnabled:                  GetEnvBool("RPC_ENABLED", true),
		RPCStatus:                   strings.ToLower(GetEnvStr("RPC_STATUS", "online")),
		RPCActivityType:             strings.ToLower(GetEnvStr("RPC_ACTIVITY_TYPE", "playing")),
		RPCActivityName:             GetEnvStr("RPC_ACTIVITY_NAME", "with AI chats"),
		RPCActivityURL:              GetEnvStr("RPC_ACTIVITY_URL", ""),
		NvidiaAPIKey:                nvidiaAPIKey,
		NvidiaModel:                 GetEnvStr("NVIDIA_MODEL", "google/gemma-3n-e4b-it"),
		SystemRulesMD:               systemRulesMD,
		SystemPrompt:                systemPrompt,
		ChatReplayLogPath:           GetEnvStr("CHAT_REPLAY_LOG_PATH", filepath.Join("logger", "chat_replay.jsonl")),
		ChatMemoryDBPath:            GetEnvStr("CHAT_MEMORY_DB_PATH", "chat_memory.db"),
		BanDBPath:                   GetEnvStr("BAN_DB_PATH", "ban_control.db"),
		CallnamesDBPath:             GetEnvStr("CALLNAMES_DB_PATH", "callnames.db"),
		MemoryIdleTTLSeconds:        GetEnvInt("MEMORY_IDLE_TTL_SECONDS", 300),
		ImageMaxBytes:               GetEnvInt("IMAGE_MAX_BYTES", 5*1024*1024),
		MaxReplyChars:               GetEnvInt("MAX_REPLY_CHARS", 1800),
		Temperature:                 GetEnvFloat("TEMPERATURE", 0.7),
		MaxHistory:                  GetEnvInt("MAX_HISTORY", 10),
		KomiFilterEnabled:           GetEnvBool("KOMIFILTER_ENABLED", true),
		KomiFilterMaxCheckChars:      GetEnvInt("KOMIFILTER_MAX_CHECK_CHARS", 6000),
		KomiFilterBlockResponseOnLeak: GetEnvBool("KOMIFILTER_BLOCK_RESPONSE_ON_LEAK", true),
		OwnerID:                     GetEnvStr("OWNER_USER_ID", ""),
		RestartIntervalHours:        GetEnvInt("RESTART_INTERVAL_HOURS", 12),
	}
}

