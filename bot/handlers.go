package bot

import (
	"log"
	"strconv"
	"strings"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (b *TetoBot) OnReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s", s.State.User.String())
	b.ApplyPresence()
}

func (b *TetoBot) OnMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	b.DeletedMessages.Store(m.ID, true)
}

func (b *TetoBot) OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle Application Commands
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		if data.Name == "chat" {
			prompt := data.Options[0].StringValue()
			userID, _ := strconv.ParseInt(i.Member.User.ID, 10, 64)
			if i.Member == nil { // DM
				userID, _ = strconv.ParseInt(i.User.ID, 10, 64)
			}

			// Check if user already agreed to ToS
			agreed, _ := b.Memory.HasAgreedToToS(userID)
			
			if agreed {
				// User already agreed, proceed with chat directly
				b.RunChatAndReplyInteraction(i, prompt)
				return
			}

			// Store the pending prompt and channel info
			pendingInfo := map[string]interface{}{
				"prompt":     prompt,
				"channelID":  i.ChannelID,
				"guildID":    i.GuildID,
			}
			b.PendingChats.Store(userID, pendingInfo)

			// Send ToS Ephemeral with Yes/No buttons
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "📜 Terms of Service Agreement",
							Description: "By chatting with Teto, you acknowledge and agree to the following terms:",
							Color:       0x39C5BB, // Teto/Miku themed teal
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:  "Discord Terms of Service",
									Value: "You must comply with [Discord's ToS](https://discord.com/terms) at all times.",
								},
								{
									Name:  "Bot Owner's ToS",
									Value: "• Be respectful and follow community guidelines.\n• No illegal or malicious use of the bot.\n• Content generated is for entertainment purposes.",
								},
								{
									Name:  "📝 Privacy Notice",
									Value: "Chat messages are logged for replay purposes. If you don't want your messages logged, use `/dontsendmydata` to opt-out.",
								},
							},
							Footer: &discordgo.MessageEmbedFooter{
								Text: "Do you agree to these terms?",
							},
						},
					},
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "✅ Yes, I Agree",
									Style:    discordgo.SuccessButton,
									CustomID: "tos_agree_" + strconv.FormatInt(userID, 10),
								},
								discordgo.Button{
									Label:    "❌ Disagree",
									Style:    discordgo.DangerButton,
									CustomID: "tos_disagree_" + strconv.FormatInt(userID, 10),
								},
							},
						},
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("InteractionRespond error: %v", err)
			}
		} else if data.Name == "dontsendmydata" {
			userID, _ := strconv.ParseInt(i.Member.User.ID, 10, 64)
			if i.Member == nil { // DM
				userID, _ = strconv.ParseInt(i.User.ID, 10, 64)
			}

			_ = b.Memory.SetLoggingOptOut(userID, true)

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "⚠️ Privacy & Data Warning",
							Description: "You have successfully opted out of **Teto's Replay Logging**.",
							Color:       0xFF5555, // Red warning color
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:  "Discord's Native Logging",
									Value: "Please remember that **Discord still logs all messages** in accordance with their ToS.",
								},
								{
									Name:  "Bot Safety & Context",
									Value: "Anything you say can still affect the bot's temporary short-term memory or trigger automated safety filters.\n\nPlease chat responsibly.",
								},
							},
						},
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("InteractionRespond error: %v", err)
			}
		}
		return
	}

	// Handle Button Clicks
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID
		
		if strings.HasPrefix(customID, "tos_agree_") {
			userIDStr := strings.TrimPrefix(customID, "tos_agree_")
			userID, _ := strconv.ParseInt(userIDStr, 10, 64)
			
			// Record ToS Agreement
			_ = b.Memory.RecordToSAgreement(userID)
			
			// Get pending chat info
			pendingInfo, ok := b.PendingChats.Load(userID)
			b.PendingChats.Delete(userID) // Clean up
			
			// Respond to button interaction
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "✅ Thank you for agreeing! Starting chat...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("InteractionRespond error: %v", err)
			}
			
			if ok {
				info := pendingInfo.(map[string]interface{})
				prompt := info["prompt"].(string)
				
				// Create a MessageCreate wrapper for the chat
				msg := &discordgo.MessageCreate{
					Message: &discordgo.Message{
						ChannelID: i.ChannelID,
						GuildID:   i.GuildID,
						Author:    i.Member.User,
						Content:   prompt,
					},
				}
				if i.Member == nil {
					msg.Message.Author = i.User
				}
				
				// Process the chat
				b.RunChatAndReply(msg, strings.TrimSpace(prompt), "slash_command")
			}
			return
		}
		
		if strings.HasPrefix(customID, "tos_disagree_") {
			userIDStr := strings.TrimPrefix(customID, "tos_disagree_")
			userID, _ := strconv.ParseInt(userIDStr, 10, 64)
			
			// Don't record agreement - they can try again next time
			b.PendingChats.Delete(userID) // Clean up
			
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ You have declined the ToS. You can agree anytime by using `/chat` again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("InteractionRespond error: %v", err)
			}
			return
		}
	}
}

func (b *TetoBot) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if _, deleted := b.DeletedMessages.Load(m.ID); deleted {
		return
	}

	// Check if banned
	var guildID int64
	if m.GuildID != "" {
		id, _ := strconv.ParseInt(m.GuildID, 10, 64)
		guildID = id
	}

	userID, _ := strconv.ParseInt(m.Author.ID, 10, 64)
	banned, _ := b.Memory.IsUserBanned(guildID, userID)
	if banned {
		return
	}

	if b.IsTerminated {
		return
	}

	// DM Support: If it's a DM, respond if not a command
	isDM := m.GuildID == ""

	// Handle Replay ID inline (e.g., !replayTeto123)
	if strings.HasPrefix(m.Content, b.Settings.CommandPrefix+"replayTeto") {
		b.HandleReplayCommand(m)
		return
	}

	// Check for commands
	if strings.HasPrefix(m.Content, b.Settings.CommandPrefix) {
		b.HandlePrefixCommands(m)
		return
	}

	// Auto-reply on mention or if it's a DM
	mentioned := false
	for _, u := range m.Mentions {
		if u.ID == s.State.User.ID {
			mentioned = true
			break
		}
	}

	if mentioned || isDM {
		prompt := m.Content
		if mentioned {
			prompt = strings.ReplaceAll(prompt, fmt.Sprintf("<@%s>", s.State.User.ID), "")
			prompt = strings.ReplaceAll(prompt, fmt.Sprintf("<@!%s>", s.State.User.ID), "")
		}
		
		// Check ToS for message-based chat
		agreed, _ := b.Memory.HasAgreedToToS(userID)
		if !agreed {
			b.Session.ChannelMessageSendReply(m.ChannelID, "Note: By chatting with Teto, you agree to Discord's ToS and the Bot Owner's ToS.\n💡 If you don't want your messages logged, use `/dontsendmydata` to opt-out. (This notification is shown only once).", m.Reference())
			_ = b.Memory.RecordToSAgreement(userID)
		}

		b.RunChatAndReply(m, strings.TrimSpace(prompt), "mention")
	}
}

