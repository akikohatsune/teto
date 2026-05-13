package bot

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/akikohatsune/teto/client"

	"github.com/bwmarrin/discordgo"
)

// generateChatReply handles the core logic of processing a prompt, fetching history,
// calling the LLM, and saving the results. It returns the generated reply string.
func (b *TetoBot) generateChatReply(
	guildID int64,
	channelID int64,
	userID int64,
	userName string,
	prompt string,
	trigger string,
	attachments []*discordgo.MessageAttachment,
) (string, error) {

	if prompt == "" {
		prompt = "hi"
	}

	// 1. Filter User Prompt
	decision := b.Filter.InspectUserPrompt(prompt)
	if decision.Blocked {
		return b.Filter.UserBlockMessage(decision), nil
	}

	// 2. Process Attachments
	var images []client.ImageInput
	for _, att := range attachments {
		if strings.HasPrefix(att.ContentType, "image/") {
			if att.Size > b.Settings.ImageMaxBytes {
				return fmt.Sprintf("Image too large: %s", att.Filename), nil
			}
			data, err := downloadFile(att.URL)
			if err == nil {
				images = append(images, client.ImageInput{
					MimeType: att.ContentType,
					DataB64:  base64.StdEncoding.EncodeToString(data),
				})
			}
		}
	}

	// 3. Load History
	historyRaw, _ := b.Memory.GetHistory(userID)
	var history []client.ChatMessage
	for _, h := range historyRaw {
		var imgs []client.ImageInput
		history = append(history, client.ChatMessage{
			Role:    h.Role,
			Content: h.Content,
			Images:  imgs,
		})
	}

	// 4. Apply Call Preferences
	userCalls, tetoCalls, _ := b.Memory.GetUserCallPreferences(guildID, userID)
	llmPrompt := prompt
	if userCalls != "" || tetoCalls != "" {
		parts := []string{"[call_profile_context]"}
		if userCalls != "" {
			parts = append(parts, "user calls Teto: "+userCalls)
		}
		if tetoCalls != "" {
			parts = append(parts, "Teto calls user: "+tetoCalls)
		}
		parts = append(parts, "[message_content]", prompt)
		llmPrompt = strings.Join(parts, "\n")
	}

	userMsg := client.ChatMessage{Role: "user", Content: llmPrompt, Images: images}
	messages := append(history, userMsg)

	// 5. Call LLM
	reply, err := b.Client.Generate(context.Background(), messages)
	if err != nil {
		log.Printf("LLM Error: %v", err)
		return "i overload!", nil
	}

	// 6. Sanitize and Filter Reply
	reply = b.SanitizeOutput(reply)
	replyDecision := b.Filter.InspectModelReply(reply)
	if replyDecision.Blocked {
		reply = b.Filter.ReplyBlockMessage()
	}

	// 7. Save Memory
	memoryPrompt := prompt
	if len(images) > 0 {
		memoryPrompt += fmt.Sprintf("\n[attached_images=%d]", len(images))
	}
	_ = b.Memory.AppendMessage(channelID, userID, "user", memoryPrompt, images)
	_ = b.Memory.AppendMessage(channelID, userID, "assistant", reply, nil)

	// 8. Log (if not opted out)
	optOut, _ := b.Memory.IsLoggingOptedOut(userID)
	if !optOut {
		_ = b.ReplayLogger.LogChat(guildID, "", channelID, "", userID, userName, userName, trigger, prompt, len(reply))
	}

	return reply, nil
}

func (b *TetoBot) SanitizeOutput(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	text = regexp.MustCompile(`@everyone`).ReplaceAllString(text, "@\u200beveryone")
	text = regexp.MustCompile(`@here`).ReplaceAllString(text, "@\u200bhere")
	text = regexp.MustCompile(` {2,}`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (b *TetoBot) RunChatAndReplyInteraction(i *discordgo.InteractionCreate, prompt string) {
	channelID, _ := strconv.ParseInt(i.ChannelID, 10, 64)
	var guildID int64
	if i.GuildID != "" {
		guildID, _ = strconv.ParseInt(i.GuildID, 10, 64)
	}
	
	userID, _ := strconv.ParseInt(i.Member.User.ID, 10, 64)
	userName := i.Member.User.Username
	if i.Member == nil {
		userID, _ = strconv.ParseInt(i.User.ID, 10, 64)
		userName = i.User.Username
	}

	reply, _ := b.generateChatReply(guildID, channelID, userID, userName, prompt, "interaction", nil)
	b.SendLongFollowup(i, reply)
}

func (b *TetoBot) SendLongFollowup(i *discordgo.InteractionCreate, text string) {
	maxLen := 1900
	if len(text) <= maxLen {
		b.Session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: text,
		})
		return
	}

	for iIdx := 0; iIdx < len(text); iIdx += maxLen {
		end := iIdx + maxLen
		if end > len(text) {
			end = len(text)
		}
		chunk := text[iIdx:end]
		b.Session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: chunk,
		})
	}
}

func (b *TetoBot) RunChatAndReply(m *discordgo.MessageCreate, prompt string, trigger string) {
	channelID, _ := strconv.ParseInt(m.ChannelID, 10, 64)
	var guildID int64
	if m.GuildID != "" {
		guildID, _ = strconv.ParseInt(m.GuildID, 10, 64)
	}
	userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
	userName := m.Author.Username

	_ = b.Session.ChannelTyping(m.ChannelID)

	reply, _ := b.generateChatReply(guildID, channelID, userID, userName, prompt, trigger, m.Attachments)
	b.SendLongMessage(m.ChannelID, reply, m.Reference())
}

func (b *TetoBot) SendLongMessage(channelID string, text string, ref *discordgo.MessageReference) {
	maxLen := 1900
	if len(text) <= maxLen {
		b.Session.ChannelMessageSendReply(channelID, text, ref)
		return
	}

	for i := 0; i < len(text); i += maxLen {
		end := i + maxLen
		if end > len(text) {
			end = len(text)
		}
		chunk := text[i:end]
		if i == 0 {
			b.Session.ChannelMessageSendReply(channelID, chunk, ref)
		} else {
			b.Session.ChannelMessageSend(channelID, chunk)
		}
	}
}

