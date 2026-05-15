package bot

import (
	"log"
	"os"
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
			var isEphemeral bool
			for _, opt := range data.Options {
				if opt.Name == "ephemeral" {
					isEphemeral = opt.BoolValue()
				}
			}

			// Defer response to avoid timeout and set ephemeral flag if needed
			var flags discordgo.MessageFlags
			if isEphemeral {
				flags = discordgo.MessageFlagsEphemeral
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: flags,
				},
			})

			// Proceed with chat directly
			b.RunChatAndReplyInteraction(i, prompt, isEphemeral)
			return
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
		} else if data.Name == "system_md" {
			// Read system_rules.md
			content, err := os.ReadFile(b.Settings.SystemRulesMD)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Error reading file: %v", err),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Respond with the content (ephemeral)
			// Using SendLongFollowup to handle potential character limits
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})

			b.SendLongFollowup(i, "📜 **Current System Rules:**\n\n"+string(content), true)
		}
		return
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
		
		b.RunChatAndReply(m, strings.TrimSpace(prompt), "mention")
	}
}

