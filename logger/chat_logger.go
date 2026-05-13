package logger

import (
	"bufio"
	"encoding/json"
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

	if _, err := os.Stat(l.LogPath); os.IsNotExist(err) {
		_ = os.WriteFile(l.LogPath, []byte(""), 0644)
		l.nextID = 1
		return nil
	}

	maxID := 0
	file, err := os.Open(l.LogPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record ChatRecord
		if err := json.Unmarshal([]byte(scanner.Text()), &record); err == nil {
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

	recordID := l.nextID
	l.nextID++

	if len(prompt) > 600 {
		prompt = prompt[:600]
	}

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

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(l.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
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

	var allRecords []ChatRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record ChatRecord
		if err := json.Unmarshal([]byte(scanner.Text()), &record); err == nil {
			if guildID == 0 || record.GuildID == guildID {
				allRecords = append(allRecords, record)
			}
		}
	}

	if len(allRecords) > limit {
		allRecords = allRecords[len(allRecords)-limit:]
	}

	// Reverse to show newest first
	for i, j := 0, len(allRecords)-1; i < j; i, j = i+1, j-1 {
		allRecords[i], allRecords[j] = allRecords[j], allRecords[i]
	}

	return allRecords, nil
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
		var record ChatRecord
		if err := json.Unmarshal([]byte(scanner.Text()), &record); err == nil {
			if record.ID == recordID {
				if guildID == 0 || record.GuildID == guildID {
					return &record, nil
				}
			}
		}
	}

	return nil, nil
}


