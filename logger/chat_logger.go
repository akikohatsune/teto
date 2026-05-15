package logger

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
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
	DBPath     string
	TextLogPath string
	db         *sql.DB
	mu         sync.Mutex
	nextID     int
}

func NewChatReplayLogger(dbPath string) *ChatReplayLogger {
	// Generate text log path from db path
	dir := filepath.Dir(dbPath)
	base := filepath.Base(dbPath)
	nameWithoutExt := base[:len(base)-len(filepath.Ext(base))]
	textLogPath := filepath.Join(dir, nameWithoutExt+"_readable.txt")
	
	return &ChatReplayLogger{
		DBPath:      dbPath,
		TextLogPath: textLogPath,
	}
}

func (l *ChatReplayLogger) Initialize() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(l.DBPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}

	db, err := sql.Open("sqlite", l.DBPath)
	if err != nil {
		return err
	}
	l.db = db

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	schema := `CREATE TABLE IF NOT EXISTS chat_replay (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL DEFAULT 'chat',
		ts_utc TEXT NOT NULL,
		guild_id INTEGER,
		guild_name TEXT,
		channel_id INTEGER NOT NULL,
		channel_name TEXT,
		user_id INTEGER NOT NULL,
		user_name TEXT NOT NULL,
		user_display TEXT NOT NULL,
		trigger TEXT NOT NULL,
		prompt TEXT NOT NULL,
		reply_length INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Create index for faster queries
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_replay_id ON chat_replay(id DESC)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_replay_guild ON chat_replay(guild_id)`)

	// Get max ID from database (O(1) instead of O(n))
	maxID := 0
	row := db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM chat_replay")
	if err := row.Scan(&maxID); err != nil && err != sql.ErrNoRows {
		return err
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

	// Insert into database
	query := `INSERT INTO chat_replay 
		(type, ts_utc, guild_id, guild_name, channel_id, channel_name, user_id, user_name, user_display, trigger, prompt, reply_length)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := l.db.Exec(query,
		"chat",
		time.Now().UTC().Format(time.RFC3339),
		guildID,
		guildName,
		channelID,
		channelName,
		userID,
		userName,
		userDisplay,
		trigger,
		prompt,
		replyLength,
	)

	if err == nil {
		l.nextID++
		
		// Also write to readable text log
		l.writeToTextLog(l.nextID-1, time.Now().UTC().Format(time.RFC3339), guildID, guildName, channelID, channelName, userID, userName, userDisplay, trigger, prompt, replyLength)
	}

	return err
}

func (l *ChatReplayLogger) writeToTextLog(id int, tsUTC string, guildID int64, guildName string, channelID int64, channelName string, userID int64, userName, userDisplay, trigger, prompt string, replyLength int) {
	// Format channel display
	channelDisplay := fmt.Sprintf("#%s", channelName)
	if channelName == "" {
		channelDisplay = fmt.Sprintf("(DM %d)", channelID)
	}
	
	// Format guild display
	guildDisplay := guildName
	if guildName == "" {
		guildDisplay = fmt.Sprintf("Guild %d", guildID)
	}
	
	// Build text entry
	entry := fmt.Sprintf(
		"[%s] %s | %s | @%s (%s)\nPrompt: %s\nReply Length: %d chars\n%s\n\n",
		tsUTC,
		guildDisplay,
		channelDisplay,
		userName,
		userDisplay,
		prompt,
		replyLength,
		"─────────────────────────────────",
	)
	
	// Append to text log file
	f, err := os.OpenFile(l.TextLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	_, _ = f.WriteString(entry)
}

func (l *ChatReplayLogger) ReadRecentIndexed(limit int, guildID int64) ([]ChatRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	query := `SELECT id, type, ts_utc, guild_id, guild_name, channel_id, channel_name, user_id, user_name, user_display, trigger, prompt, reply_length 
	          FROM chat_replay`

	if guildID > 0 {
		query += ` WHERE guild_id = ?`
	}

	query += ` ORDER BY id DESC LIMIT ?`

	var rows *sql.Rows
	var err error

	if guildID > 0 {
		rows, err = l.db.Query(query, guildID, limit)
	} else {
		rows, err = l.db.Query(query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ChatRecord
	for rows.Next() {
		var r ChatRecord
		err := rows.Scan(&r.ID, &r.Type, &r.TSUTC, &r.GuildID, &r.GuildName, &r.ChannelID, &r.ChannelName, &r.UserID, &r.UserName, &r.UserDisplay, &r.Trigger, &r.Prompt, &r.ReplyLength)
		if err != nil {
			continue
		}
		records = append(records, r)
	}

	return records, nil
}

func (l *ChatReplayLogger) GetByIndex(recordID int, guildID int64) (*ChatRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	query := `SELECT id, type, ts_utc, guild_id, guild_name, channel_id, channel_name, user_id, user_name, user_display, trigger, prompt, reply_length 
	          FROM chat_replay WHERE id = ?`

	if guildID > 0 {
		query += ` AND guild_id = ?`
	}

	row := l.db.QueryRow(query, recordID)

	var r ChatRecord
	var err error

	if guildID > 0 {
		err = row.Scan(&r.ID, &r.Type, &r.TSUTC, &r.GuildID, &r.GuildName, &r.ChannelID, &r.ChannelName, &r.UserID, &r.UserName, &r.UserDisplay, &r.Trigger, &r.Prompt, &r.ReplyLength)
	} else {
		err = row.Scan(&r.ID, &r.Type, &r.TSUTC, &r.GuildID, &r.GuildName, &r.ChannelID, &r.ChannelName, &r.UserID, &r.UserName, &r.UserDisplay, &r.Trigger, &r.Prompt, &r.ReplyLength)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &r, nil
}

func (l *ChatReplayLogger) Close() error {
	if l.db != nil {
		return l.db.Close()
	}
	return nil
}


