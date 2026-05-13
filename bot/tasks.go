package bot

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *TetoBot) StartBackgroundTasks() {
	go b.memoryCleanupLoop()
	if b.Settings.RestartIntervalHours > 0 {
		go b.scheduledRestartLoop()
	}
}

func (b *TetoBot) memoryCleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := b.Memory.PruneInactiveUsers(b.Settings.MemoryIdleTTLSeconds); err != nil {
			log.Printf("[memory-cleanup] Error pruning inactive users: %v", err)
		}
		if err := b.Memory.PruneOldImages(b.Settings.MemoryIdleTTLSeconds); err != nil {
			log.Printf("[memory-cleanup] Error pruning old images: %v", err)
		}
	}
}

func (b *TetoBot) scheduledRestartLoop() {
	ticker := time.NewTicker(time.Duration(b.Settings.RestartIntervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Printf("Scheduled restart triggered after %d hours...", b.Settings.RestartIntervalHours)
		// In Go, we might want to just exit and let a process manager restart it,
		// or handle it gracefully. For now, we'll just exit.
		if b.Settings.OwnerID != "" {
			b.Session.ChannelMessageSend(b.Settings.OwnerID, "Performing scheduled restart...")
		}
		b.Close()
		log.Fatal("Scheduled restart exit.")
	}
}

func (b *TetoBot) ApplyPresence() {
	if !b.Settings.RPCEnabled {
		return
	}

	status := discordgo.StatusOnline
	switch b.Settings.RPCStatus {
	case "idle":
		status = discordgo.StatusIdle
	case "dnd":
		status = discordgo.StatusDoNotDisturb
	case "invisible":
		status = discordgo.StatusInvisible
	}

	var activityType discordgo.ActivityType
	switch b.Settings.RPCActivityType {
	case "listening":
		activityType = discordgo.ActivityTypeListening
	case "watching":
		activityType = discordgo.ActivityTypeWatching
	case "competing":
		activityType = discordgo.ActivityTypeCompeting
	case "streaming":
		activityType = discordgo.ActivityTypeStreaming
	default:
		activityType = discordgo.ActivityTypeGame
	}

	activity := &discordgo.Activity{
		Name: b.Settings.RPCActivityName,
		Type: activityType,
		URL:  b.Settings.RPCActivityURL,
	}

	_ = b.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status:     string(status),
		Activities: []*discordgo.Activity{activity},
	})

	log.Printf("Discord RPC presence applied: status=%s, type=%s, name=%s", b.Settings.RPCStatus, b.Settings.RPCActivityType, b.Settings.RPCActivityName)
}


