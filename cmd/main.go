package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"memo-bot/internal/config"
	"memo-bot/internal/discord"
	"memo-bot/internal/service"

	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Get local timezone
	localLoc, err := time.LoadLocation("Local")
	if err != nil {
		log.Fatalf("Failed to load local timezone: %v", err)
	}

	memoService := service.NewMemoService(db)

	// Set up Discord client
	discordClient, err := discord.NewClient(cfg.Discord.BotToken, memoService)
	if err != nil {
		log.Fatalf("Failed to create Discord client: %v", err)
	}

	// Connect to Discord
	if err := discordClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to Discord: %v", err)
	}
	defer discordClient.Close()

	scanInterval, err := time.ParseDuration(cfg.App.ScanInterval)
	if err != nil {
		log.Fatalf("Failed to parse scan interval: %v", err)
	}

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	log.Printf("Backend started in timezone: %s", localLoc.String())
	log.Printf("Checking for reminders every %v", scanInterval)

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Check for missed reminders on startup
	log.Printf("Performing initial scan for missed reminders...")
	checkReminders(memoService, discordClient)
	log.Printf("Initial scan completed")

	running := true
	for running {
		select {
		case <-ticker.C:
			log.Printf("Scanning for reminders at %s...", time.Now().In(localLoc).Format("2006-01-02 15:04:05 MST"))
			checkReminders(memoService, discordClient)
			log.Printf("Scan completed at %s", time.Now().In(localLoc).Format("2006-01-02 15:04:05 MST"))
		case <-stop:
			log.Println("Shutting down gracefully...")
			running = false
		}
	}
}

func handleSetChannel(s *discordgo.Session, m *discordgo.MessageCreate, service *service.MemoService) {
	parts := strings.Fields(m.Content)
	if len(parts) != 2 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !setchannel <user_id>")
		return
	}

	userID := parts[1]
	ctx := context.Background()

	// Update the user's Discord channel
	err := service.UpdateUserDiscordChannel(ctx, userID, m.ChannelID)
	if err != nil {
		log.Printf("Error updating user's Discord channel: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to set channel. Make sure your user ID is correct.")
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Successfully set this channel for reminders for user ID: %s", userID))
}

func handleHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	helpText := `**MemoBot Commands**
!setchannel <user_id> - Set the current channel to receive your reminders
!help - Show this help message

Note: The user_id is the same ID you use in the CLI application.`

	s.ChannelMessageSend(m.ChannelID, helpText)
}

func checkReminders(service *service.MemoService, discordClient *discord.Client) {
	ctx := context.Background()
	now := time.Now().UTC()

	reminders, err := service.GetPendingReminders(ctx, now)
	if err != nil {
		log.Printf("Error getting pending reminders: %v", err)
		return
	}

	if len(reminders) > 0 {
		log.Printf("Found %d reminder(s) to process", len(reminders))
	}

	for _, reminder := range reminders {
		if err := discordClient.SendReminder(reminder); err != nil {
			log.Printf("Error sending reminder: %v", err)
			continue
		}

		if err := service.MarkMemoAsSent(ctx, reminder.ID); err != nil {
			log.Printf("Error marking memo as sent: %v", err)
		}
	}
}
