package main

import (
	"bufio"
	"log"
	"github.com/akikohatsune/teto/bot"
	"github.com/akikohatsune/teto/config"
	"github.com/akikohatsune/teto/utils"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// Sync .env
	utils.AutoMergeDotenv()

	// Load settings
	settings := config.LoadSettings()

	// Initialize Bot
	Teto, err := bot.NewTetoBot(settings)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Start Bot
	if err := Teto.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	log.Println("Teto (Go Edition) is now running. Press CTRL-C to exit.")
	log.Println("Terminal commands: 'reload', 'annihilate'")

	// Terminal command listener
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd := strings.TrimSpace(scanner.Text())
			switch cmd {
			case "reload":
				Teto.ReloadRules()
			case "annihilate":
				log.Println("!!! ANNIHILATING ALL DATA !!!")
				Teto.Close()
				// Delete files
				files := []string{
					settings.ChatMemoryDBPath,
					settings.BanDBPath,
					settings.CallnamesDBPath,
					settings.ChatReplayLogPath,
				}
				for _, f := range files {
					if err := os.Remove(f); err == nil {
						log.Printf("Deleted: %s", f)
					} else if !os.IsNotExist(err) {
						log.Printf("Error deleting %s: %v", f, err)
					}
				}
				log.Println("Data annihilation complete. Exiting...")
				os.Exit(0)
			}
		}
	}()

	// Wait for interrupt signal to gracefully shut down the bot.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session and memory store.
	Teto.Close()
	log.Println("Shutting down...")
}

