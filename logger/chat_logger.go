package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ChatRecord struct {
	ID           int       `json:"id"`
	Type         string    `json:"type"`
	TSUTC        string    `json:"ts_utc"`
	GuildID      int64     `json:"guild_id,omitempty"`
	GuildName    string    `json:"guild_name,omitempty"`
	ChannelID    int64     `json:"channel_id"`
	ChannelName  string    `json:"channel_name,omitempty"`
	UserID       int64     `json:"user_id"`
	UserName     string    `json:"user_name"`
	UserDisplay  string    `json:"user_display"`
	Trigger      string    `json:"trigger"`
	Prompt       string    `json:"prompt"`
	ReplyLength  int       `json:"reply_length"`
}

type ChatReplayLogger struct {
	LogPath string
	mu      sync.Mutex
	nextID  int
}

func NewChatReplayLogger(logPath string) *ChatReplayLogger {
	return &ChatReplayLogger{
		LogPath: logPath,
	}
}

func (l *ChatReplayLogger) Initialize() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(l.LogPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}

	// If file doesn't exist, create it
	if _, err := os.Stat(l.LogPath); os.IsNotExist(err) {
		_ = os.WriteFile(l.LogPath, []byte(""), 0644)
		l.nextID = 1
		return nil
	}

	// Find max ID from existing file
	maxID := 0
	file, err := os.Open(l.LogPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip separators and empty lines
		if line == "" || line[0] == '-' || line[0] == '[' {
			continue
		}
		
		// Try to parse JSON from lines that look like records
		var record ChatRecord
		if err := json.Unmarshal([]byte(line), &record); err == nil {
			if record.ID > maxID {
				maxID = record.ID
			}
		}
	}

	l.nextID = maxID + 1
	return nil
}

func (l *ChatReplayLogger) LogChat(guildID int64, guildName string, channelID int64, channelName string, userID int64, userName, userDisplay, trigger, prompt string, replyLength int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(prompt) > 600 {
		prompt = prompt[:600]
	}

	recordID := l.nextID
	l.nextID++

	// Create JSON record
	record := ChatRecord{
		ID:           recordID,
		Type:         "chat",
		TSUTC:        time.Now().UTC().Format(time.RFC3339),
		GuildID:      guildID,
		GuildName:    guildName,
		ChannelID:    channelID,
		ChannelName:  channelName,
		UserID:       userID,
		UserName:     userName,
		UserDisplay:  userDisplay,
		Trigger:      trigger,
		Prompt:       prompt,
		ReplyLength:  replyLength,
	}

	// Marshal to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// Format readable entry
	channelDisplay := fmt.Sprintf("#%s", channelName)
	if channelName == "" {
		channelDisplay = fmt.Sprintf("(DM %d)", channelID)
	}

	guildDisplay := guildName
	if guildName == "" {
		guildDisplay = fmt.Sprintf("Guild %d", guildID)
	}

	entry := fmt.Sprintf(
		"%s\n[%s] %s | %s | @%s (%s)\nPrompt: %s\nReply Length: %d chars\n%s\n\n",
		string(data),
		record.TSUTC,
		guildDisplay,
		channelDisplay,
		userName,
		userDisplay,
		prompt,
		replyLength,
		"─────────────────────────────────",
	)

	// Append to file
	f, err := os.OpenFile(l.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}

func (l *ChatReplayLogger) ReadRecentIndexed(limit int, guildID int64) ([]ChatRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.LogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []ChatRecord
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse JSON records
		var record ChatRecord
		if err := json.Unmarshal([]byte(line), &record); err == nil {
			if guildID == 0 || record.GuildID == guildID {
				records = append(records, record)
			}
		}
	}

	// Keep only last 'limit' records
	if len(records) > limit {
		records = records[len(records)-limit:]
	}

	// Reverse to show newest first
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	return records, nil
}

func (l *ChatReplayLogger) GetByIndex(recordID int, guildID int64) (*ChatRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.LogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		
		var record ChatRecord
		if err := json.Unmarshal([]byte(line), &record); err == nil {
			if record.ID == recordID {
				if guildID == 0 || record.GuildID == guildID {
					return &record, nil
				}
			}
		}
	}

	return nil, nil
}

func (l *ChatReplayLogger) Close() error {
	// No database to close
	return nil
}


