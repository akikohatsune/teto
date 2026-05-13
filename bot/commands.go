package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *TetoBot) IsOwner(userID string) bool {
	return b.Settings.OwnerID == "" || b.Settings.OwnerID == userID
}

func (b *TetoBot) HandleReplayCommand(m *discordgo.MessageCreate) {
	if !b.IsOwner(m.Author.ID) {
		b.Session.ChannelMessageSendReply(m.ChannelID, "Only the bot owner can use this command.", m.Reference())
		return
	}

	rawID := strings.TrimPrefix(m.Content, b.Settings.CommandPrefix+"replayTeto")
	if rawID == "" {
		b.HandlePrefixCommands(m)
		return
	}

	id, err := strconv.Atoi(rawID)
	if err != nil {
		return
	}

	var guildID int64
	if m.GuildID != "" {
		guildID, _ = strconv.ParseInt(m.GuildID, 10, 64)
	}

	record, _ := b.ReplayLogger.GetByIndex(id, guildID)
	if record == nil {
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Replay ID %d not found.", id), m.Reference())
		return
	}

	payload := fmt.Sprintf("Replay #%d\nTime: %s\nUser: %s (%d)\nTrigger: %s\nPrompt:\n%s", record.ID, record.TSUTC, record.UserDisplay, record.UserID, record.Trigger, record.Prompt)
	b.SendLongMessage(m.ChannelID, payload, m.Reference())
}

func (b *TetoBot) HandlePrefixCommands(m *discordgo.MessageCreate) {
	parts := strings.Fields(m.Content)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(strings.TrimPrefix(parts[0], b.Settings.CommandPrefix))
	args := parts[1:]

	var guildID int64
	if m.GuildID != "" {
		guildID, _ = strconv.ParseInt(m.GuildID, 10, 64)
	}

	switch cmd {
	case "chat", "ask":
		prompt := strings.Join(args, " ")
		b.RunChatAndReply(m, prompt, "command")

	case "clearmemo", "resetchat":
		if !b.IsOwner(m.Author.ID) {
			b.Session.ChannelMessageSendReply(m.ChannelID, "Only owner can clear memory.", m.Reference())
			return
		}
		_ = b.Memory.ClearAllHistory()
		b.Session.ChannelMessageSendReply(m.ChannelID, "Cleared all memory.", m.Reference())

	case "ban":
		if !b.IsOwner(m.Author.ID) {
			return
		}
		if len(m.Mentions) == 0 {
			return
		}
		target := m.Mentions[0]
		targetID, _ := strconv.ParseInt(target.ID, 10, 64)
		ownerID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
		reason := strings.Join(args, " ")
		_, _ = b.Memory.BanUser(guildID, targetID, ownerID, reason)
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Banned %s.", target.Username), m.Reference())

	case "removeban", "unban":
		if !b.IsOwner(m.Author.ID) {
			return
		}
		if len(m.Mentions) == 0 {
			return
		}
		target := m.Mentions[0]
		targetID, _ := strconv.ParseInt(target.ID, 10, 64)
		_, _ = b.Memory.UnbanUser(guildID, targetID)
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Unbanned %s.", target.Username), m.Reference())

	case "ucallTeto", "callTeto":
		if len(args) == 0 {
			return
		}
		name := strings.Join(args, " ")
		userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
		_ = b.Memory.SetUserCallsTeto(guildID, userID, name)
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Saved: you call Teto `%s`.", name), m.Reference())

	case "Tetocallu", "callme":
		if len(args) == 0 {
			return
		}
		name := strings.Join(args, " ")
		userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
		_ = b.Memory.SetTetoCallsUser(guildID, userID, name)
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Saved: Teto will call you `%s`.", name), m.Reference())

	case "Tetomention", "callprofile":
		userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
		u, Teto, _ := b.Memory.GetUserCallPreferences(guildID, userID)
		if u == "" {
			u = "Teto"
		}
		if Teto == "" {
			Teto = m.Author.Username
		}
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Current profile | You call Teto: `%s` | Teto calls you: `%s`", u, Teto), m.Reference())

	case "dontsendmydata":
		userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
		_ = b.Memory.SetLoggingOptOut(userID, true)
		b.Session.ChannelMessageSendReply(m.ChannelID, "⚠️ **Privacy Warning:** You have opted out of chat replay logging. However, please remember that **Discord still logs all messages** per their ToS. Also, anything you say can still affect the bot's temporary context or trigger safety filters. Please chat responsibly.", m.Reference())

	case "provider":

		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Provider: NVIDIA NIM | Model: %s | DB: %s", b.Settings.NvidiaModel, b.Settings.ChatMemoryDBPath), m.Reference())

	case "terminated":
		if !b.IsOwner(m.Author.ID) {
			return
		}
		if len(args) == 0 {
			b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Terminated status: %v", b.IsTerminated), m.Reference())
			return
		}
		action := strings.ToLower(args[0])
		if action == "on" {
			b.IsTerminated = true
		} else if action == "off" {
			b.IsTerminated = false
		}
		b.Session.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Terminated set to %v", b.IsTerminated), m.Reference())

	case "replayTeto":
		if !b.IsOwner(m.Author.ID) {
			return
		}
		if len(args) == 0 || args[0] == "ls" {
			records, _ := b.ReplayLogger.ReadRecentIndexed(20, guildID)
			lines := []string{"Recent replay logs:"}
			for _, r := range records {
				lines = append(lines, fmt.Sprintf("[%d] %s | %s | %s", r.ID, r.TSUTC, r.UserDisplay, r.Trigger))
			}
			b.SendLongMessage(m.ChannelID, strings.Join(lines, "\n"), m.Reference())
		} else {
			id, _ := strconv.Atoi(args[0])
			record, _ := b.ReplayLogger.GetByIndex(id, guildID)
			if record != nil {
				payload := fmt.Sprintf("Replay #%d\nUser: %s\nPrompt: %s", record.ID, record.UserDisplay, record.Prompt)
				b.SendLongMessage(m.ChannelID, payload, m.Reference())
			}
		}
	}
}


