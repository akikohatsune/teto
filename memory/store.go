package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

type ChatMessage struct {
	Role    string      `json:"role"`
	Content string      `json:"content"`
	Images  interface{} `json:"images,omitempty"`
}

type ShortTermMemoryStore struct {
	DBPath     string
	MaxHistory int
	db         *sql.DB
	mu         sync.Mutex
}

func NewShortTermMemoryStore(dbPath string, maxHistory int) *ShortTermMemoryStore {
	return &ShortTermMemoryStore{
		DBPath:     dbPath,
		MaxHistory: maxHistory,
	}
}

func (s *ShortTermMemoryStore) Initialize() error {
	dir := filepath.Dir(s.DBPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}

	db, err := sql.Open("sqlite", s.DBPath)
	if err != nil {
		return err
	}
	s.db = db

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	schemas := []string{
		`CREATE TABLE IF NOT EXISTS chat_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 0,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			images_json TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bot_banned_users (
			guild_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			banned_by INTEGER,
			reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (guild_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_call_preferences (
			guild_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			user_calls_Teto TEXT,
			Teto_calls_user TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (guild_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS privacy_preferences (
			user_id INTEGER PRIMARY KEY,
			opt_out_logging BOOLEAN NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_memory_user_id_id ON chat_memory (user_id, id)`,
		`CREATE INDEX IF NOT EXISTS idx_bot_banned_users_guild_user ON bot_banned_users (guild_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_call_preferences_guild_user ON user_call_preferences (guild_id, user_id)`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}

func (s *ShortTermMemoryStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *ShortTermMemoryStore) AppendMessage(channelID int64, userID int64, role, content string, images interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var imagesJSON sql.NullString
	if images != nil {
		data, err := json.Marshal(images)
		if err == nil {
			imagesJSON = sql.NullString{String: string(data), Valid: true}
		}
	}

	_, err := s.db.Exec(
		"INSERT INTO chat_memory (channel_id, user_id, role, content, images_json) VALUES (?, ?, ?, ?, ?)",
		channelID, userID, role, content, imagesJSON,
	)
	if err != nil {
		return err
	}

	return s.trimUserHistory(userID)
}

func (s *ShortTermMemoryStore) GetHistory(userID int64) ([]ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(
		`SELECT role, content, images_json 
		 FROM chat_memory 
		 WHERE user_id = ? 
		 ORDER BY id DESC 
		 LIMIT ?`,
		userID, s.MaxHistory*2,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var imagesJSON sql.NullString
		if err := rows.Scan(&msg.Role, &msg.Content, &imagesJSON); err != nil {
			continue
		}
		if imagesJSON.Valid {
			_ = json.Unmarshal([]byte(imagesJSON.String), &msg.Images)
		}
		history = append(history, msg)
	}

	// Reverse history to be in chronological order
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	return history, nil
}

func (s *ShortTermMemoryStore) ClearAllHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("DELETE FROM chat_memory")
	return err
}

func (s *ShortTermMemoryStore) BanUser(guildID int64, userID int64, bannedBy int64, reason string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var exists bool
	_ = s.db.QueryRow("SELECT 1 FROM bot_banned_users WHERE guild_id = ? AND user_id = ?", guildID, userID).Scan(&exists)

	_, err := s.db.Exec(
		`INSERT INTO bot_banned_users (guild_id, user_id, banned_by, reason)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(guild_id, user_id) DO UPDATE SET
		 	banned_by = excluded.banned_by,
			reason = excluded.reason,
			updated_at = CURRENT_TIMESTAMP`,
		guildID, userID, bannedBy, reason,
	)
	return !exists, err
}

func (s *ShortTermMemoryStore) UnbanUser(guildID int64, userID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec("DELETE FROM bot_banned_users WHERE guild_id = ? AND user_id = ?", guildID, userID)
	if err != nil {
		return false, err
	}
	count, _ := res.RowsAffected()
	return count > 0, nil
}

func (s *ShortTermMemoryStore) IsUserBanned(guildID int64, userID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM bot_banned_users WHERE guild_id = ? AND user_id = ?", guildID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

func (s *ShortTermMemoryStore) SetUserCallsTeto(guildID int64, userID int64, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO user_call_preferences (guild_id, user_id, user_calls_Teto)
		 VALUES (?, ?, ?)
		 ON CONFLICT(guild_id, user_id) DO UPDATE SET
		 	user_calls_Teto = excluded.user_calls_Teto,
			updated_at = CURRENT_TIMESTAMP`,
		guildID, userID, name,
	)
	return err
}

func (s *ShortTermMemoryStore) SetTetoCallsUser(guildID int64, userID int64, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO user_call_preferences (guild_id, user_id, Teto_calls_user)
		 VALUES (?, ?, ?)
		 ON CONFLICT(guild_id, user_id) DO UPDATE SET
		 	Teto_calls_user = excluded.Teto_calls_user,
			updated_at = CURRENT_TIMESTAMP`,
		guildID, userID, name,
	)
	return err
}

func (s *ShortTermMemoryStore) GetUserCallPreferences(guildID int64, userID int64) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var userCallsTeto, TetoCallsUser sql.NullString
	err := s.db.QueryRow(
		"SELECT user_calls_Teto, Teto_calls_user FROM user_call_preferences WHERE guild_id = ? AND user_id = ?",
		guildID, userID,
	).Scan(&userCallsTeto, &TetoCallsUser)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return userCallsTeto.String, TetoCallsUser.String, err
}

func (s *ShortTermMemoryStore) SetLoggingOptOut(userID int64, optOut bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("INSERT OR REPLACE INTO privacy_preferences (user_id, opt_out_logging) VALUES (?, ?)", userID, optOut)
	return err
}

func (s *ShortTermMemoryStore) IsLoggingOptedOut(userID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var optOut bool
	err := s.db.QueryRow("SELECT opt_out_logging FROM privacy_preferences WHERE user_id = ?", userID).Scan(&optOut)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return optOut, err
}

func (s *ShortTermMemoryStore) PruneInactiveUsers(idleSeconds int) error {
	if idleSeconds <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`DELETE FROM chat_memory
		 WHERE user_id IN (
			 SELECT user_id
			 FROM chat_memory
			 GROUP BY user_id
			 HAVING MAX(created_at) < datetime('now', ?)
		 )`,
		fmt.Sprintf("-%d seconds", idleSeconds),
	)
	return err
}

func (s *ShortTermMemoryStore) PruneOldImages(expirySeconds int) error {
	if expirySeconds <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		"UPDATE chat_memory SET images_json = NULL WHERE created_at < datetime('now', ?)",
		fmt.Sprintf("-%d seconds", expirySeconds),
	)
	return err
}

func (s *ShortTermMemoryStore) trimUserHistory(userID int64) error {
	var cutoffID int64
	err := s.db.QueryRow(
		`SELECT id FROM chat_memory WHERE user_id = ? ORDER BY id DESC LIMIT 1 OFFSET ?`,
		userID, s.MaxHistory*2-1,
	).Scan(&cutoffID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM chat_memory WHERE user_id = ? AND id < ?", userID, cutoffID)
	return err
}

