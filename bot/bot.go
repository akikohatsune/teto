package bot

import (
	"github.com/akikohatsune/teto/client"
	"github.com/akikohatsune/teto/config"
	"github.com/akikohatsune/teto/filter"
	"github.com/akikohatsune/teto/logger"
	"github.com/akikohatsune/teto/memory"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type TetoBot struct {
	Session         *discordgo.Session
	Settings        *config.Settings
	Client          *client.LLMClient
	Memory          *memory.ShortTermMemoryStore
	Filter          *filter.KomiFilter
	ReplayLogger    *logger.ChatReplayLogger
	IsTerminated    bool
	DeletedMessages sync.Map
}

func NewTetoBot(settings *config.Settings) (*TetoBot, error) {
	dg, err := discordgo.New("Bot " + settings.DiscordToken)
	if err != nil {
		return nil, err
	}

	mem := memory.NewShortTermMemoryStore(settings.ChatMemoryDBPath, settings.MaxHistory)
	if err := mem.Initialize(); err != nil {
		return nil, err
	}

	kf := filter.NewKomiFilter(settings.KomiFilterEnabled, settings.KomiFilterMaxCheckChars, settings.KomiFilterBlockResponseOnLeak)
	rl := logger.NewChatReplayLogger(settings.ChatReplayLogPath)
	if err := rl.Initialize(); err != nil {
		return nil, err
	}

	llm := client.NewLLMClient(settings)

	return &TetoBot{
		Session:      dg,
		Settings:     settings,
		Client:       llm,
		Memory:       mem,
		Filter:       kf,
		ReplayLogger: rl,
	}, nil
}

func (b *TetoBot) Start() error {
	b.Session.AddHandler(b.OnReady)
	b.Session.AddHandler(b.OnMessageCreate)
	b.Session.AddHandler(b.OnMessageDelete)
	b.Session.AddHandler(b.OnInteractionCreate)

	b.Session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	if err := b.Session.Open(); err != nil {
		return err
	}

	// Register Application Commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "Chat with Teto",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "What do you want to say to Teto?",
					Required:    true,
				},
			},
		},
		{
			Name:        "dontsendmydata",
			Description: "Opt-out of chat replay logging",
		},
	}


	for _, v := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", v)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", v.Name, err)
		}
	}

	b.StartBackgroundTasks()

	return nil
}

func (b *TetoBot) Close() {
	_ = b.Session.Close()
	_ = b.Memory.Close()
}

func (b *TetoBot) ReloadRules() {
	newPrompt := config.LoadSystemRulesPrompt(b.Settings.SystemRulesMD)
	if newPrompt != "" {
		b.Settings.SystemPrompt = newPrompt
		log.Println("[RELOAD] System rules reloaded successfully.")
	} else {
		log.Println("[RELOAD] Failed to reload rules or rules file is empty.")
	}
}

