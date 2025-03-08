package discord

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"memo-bot/internal/db"
	"memo-bot/internal/service"
	"memo-bot/internal/timeutil"

	"github.com/bwmarrin/discordgo"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "memo",
		Description: "Create a new memo",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "content",
				Description: "What to remind you about",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "when",
				Description: "When to remind you ('in 2 hours', 'tomorrow at 3pm', 'next monday at 15:00', or '2024-03-07 15:30')",
				Required:    true,
			},
		},
	},
	{
		Name:        "list",
		Description: "Show all your pending memos",
	},
	{
		Name:        "delete",
		Description: "Delete a specific memo",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "id",
				Description: "The ID of the memo to delete",
				Required:    true,
			},
		},
	},
}

// Client represents a Discord client that handles all Discord-related operations
type Client struct {
	session  *discordgo.Session
	service  *service.MemoService
	timezone string
}

// NewClient creates a new Discord client
func NewClient(botToken string, service *service.MemoService, timezone string) (*Client, error) {
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	client := &Client{
		session:  session,
		service:  service,
		timezone: timezone,
	}

	// Set up command handlers
	session.AddHandler(client.handleInteraction)

	return client, nil
}

// Connect establishes a connection to Discord and registers slash commands
func (c *Client) Connect() error {
	if err := c.session.Open(); err != nil {
		return fmt.Errorf("failed to connect to Discord: %w", err)
	}

	// Register slash commands
	for _, cmd := range commands {
		_, err := c.session.ApplicationCommandCreate(c.session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("failed to create slash command %q: %w", cmd.Name, err)
		}
	}

	return nil
}

func (c *Client) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()

	var response string
	var err error
	var isEphemeral bool = true // Set default to ephemeral for all commands

	switch data.Name {
	case "memo":
		response, err = c.handleMemoCommand(s, i)
	case "list":
		response, err = c.handleListCommand(s, i)
	case "delete":
		response, err = c.handleDeleteCommand(s, i)
	}

	if err != nil {
		response = fmt.Sprintf("❌ %s", err)
	}

	var flags discordgo.MessageFlags
	if isEphemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
			Flags:   flags,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (c *Client) handleMemoCommand(s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	options := i.ApplicationCommandData().Options
	content := options[0].StringValue()
	timeStr := options[1].StringValue()

	// Parse relative and absolute time formats using timeutil package
	remindAt, err := timeutil.ParseTime(timeStr, c.timezone)
	if err != nil {
		return "", fmt.Errorf("invalid time format (case-insensitive). Examples:\n- today at 3pm\n- tomorrow at 3pm\n- in 2 hours\n- next monday at 15:00\n- 2024-03-07 15:30")
	}

	// Check if the time is in the past
	if remindAt.Before(time.Now()) {
		return "", fmt.Errorf("memo time must be in the future")
	}

	ctx := context.Background()
	err = c.service.CreateMemo(ctx, i.Member.User.ID, i.ChannelID, content, remindAt)
	if err != nil {
		return "", err
	}

	// Shorten content if it's too long
	displayContent := content
	if len(content) > 50 {
		displayContent = content[:47] + "..."
	}

	// Load configured timezone
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		log.Printf("Error loading timezone: %v, falling back to Local", err)
		loc = time.Local
	}

	return fmt.Sprintf("✅ <@%s> created a memo: %s\n⏰ %s",
		i.Member.User.ID,
		displayContent,
		remindAt.In(loc).Format("Monday, January 2, 2006 at 15:04 MST")), nil
}

func (c *Client) handleListCommand(s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	ctx := context.Background()

	// Get personal memos for current channel
	personalMemos, err := c.service.ListPendingMemos(ctx, i.Member.User.ID, i.ChannelID)
	if err != nil {
		return "", err
	}

	// Get all memos in current channel
	allChannelMemos, err := c.service.ListAllPendingMemosInChannel(ctx, i.ChannelID)
	if err != nil {
		return "", err
	}

	// Get counts across all channels for the user
	counts, err := c.service.GetReminderCounts(ctx, i.Member.User.ID)
	if err != nil {
		return "", err
	}

	// Load configured timezone
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		log.Printf("Error loading timezone: %v, falling back to Local", err)
		loc = time.Local
	}

	var response strings.Builder

	response.WriteString(fmt.Sprintf("**Current channel** · %d memo(s) from all users\n", len(allChannelMemos)))

	// Show personal memos in current channel
	response.WriteString("## Your memos in this channel\n")
	if len(personalMemos) == 0 {
		response.WriteString("You have no memos in this channel.\n")
	} else {
		for _, memo := range personalMemos {
			response.WriteString(fmt.Sprintf("\n🔸 **Memo #%d**\n", memo.ID))
			response.WriteString(fmt.Sprintf("⏰ %s\n", memo.RemindAt.In(loc).Format("Monday, January 2, 2006 at 15:04 MST")))
			response.WriteString(fmt.Sprintf("📌 %s\n", memo.Content))
			response.WriteString("───────────────────\n")
		}
	}

	// Show other users' memos in current channel
	otherMemos := filterOtherUserMemos(allChannelMemos, i.Member.User.ID)
	if len(otherMemos) > 0 {
		response.WriteString("## Other's memos in this channel\n")
		for _, memo := range otherMemos {
			user, err := s.User(memo.DiscordUserID)
			username := fmt.Sprintf("<@%s>", memo.DiscordUserID)
			if err != nil || user == nil {
				username = "Unknown User"
			}
			response.WriteString(fmt.Sprintf("\n🔹 **Memo #%d** by %s\n", memo.ID, username))
			response.WriteString(fmt.Sprintf("⏰ %s\n", memo.RemindAt.In(loc).Format("Monday, January 2, 2006 at 15:04 MST")))
			response.WriteString(fmt.Sprintf("📌 %s\n", memo.Content))
			response.WriteString("───────────────────\n")
		}
	}

	// Build summary of other channels
	totalMemos := 0
	otherChannels := make(map[string][]channelInfo)
	for _, count := range counts {
		totalMemos += int(count.Count)
		if count.DiscordChannelID != i.ChannelID {
			channel, err := s.Channel(count.DiscordChannelID)
			if err != nil || channel == nil {
				continue
			}
			otherChannels[channel.GuildID] = append(otherChannels[channel.GuildID], channelInfo{
				channel: channel,
				count:   int(count.Count),
			})
		}
	}

	// Add summary section if there are memos in other channels
	if len(otherChannels) > 0 {
		response.WriteString("## Your memos in other channels / servers\n")
		for guildID, channels := range otherChannels {
			guild, err := s.Guild(guildID)
			if err != nil || guild == nil {
				continue
			}
			for _, ch := range channels {
				response.WriteString(fmt.Sprintf("• <#%s>: %d memo(s)\n", ch.channel.ID, ch.count))
			}
		}
	}

	// Add total count at the bottom
	response.WriteString(fmt.Sprintf("\n📈 Total personal memos across all channels: %d", totalMemos))

	return response.String(), nil
}

// filterOtherUserMemos returns memos that don't belong to the specified user
func filterOtherUserMemos(memos []db.Memo, userID string) []db.Memo {
	var otherMemos []db.Memo
	for _, memo := range memos {
		if memo.DiscordUserID != userID {
			otherMemos = append(otherMemos, memo)
		}
	}
	return otherMemos
}

type channelInfo struct {
	channel *discordgo.Channel
	count   int
}

func (c *Client) handleDeleteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) (string, error) {
	memoID := i.ApplicationCommandData().Options[0].IntValue()

	ctx := context.Background()

	// First check if the memo exists and belongs to the user
	memo, err := c.service.GetMemo(ctx, int32(memoID))
	if err != nil {
		return "", err
	}

	if memo.DiscordUserID != i.Member.User.ID {
		return "", fmt.Errorf("memo #%d belongs to <@%s>. You can only delete your own memos", memoID, memo.DiscordUserID)
	}

	err = c.service.DeleteMemo(ctx, int32(memoID), i.Member.User.ID)
	if err != nil {
		return "", fmt.Errorf("failed to delete memo: %v", err)
	}

	return "✅ Memo deleted successfully!", nil
}

// Close closes the Discord connection
func (c *Client) Close() error {
	return c.session.Close()
}

// SendReminder sends a reminder message to Discord
func (c *Client) SendReminder(memo db.Memo) error {
	// Load configured timezone
	loc, err := time.LoadLocation(c.timezone)
	if err != nil {
		log.Printf("Error loading timezone: %v, falling back to Local", err)
		loc = time.Local
	}

	// This message is public since it's the actual reminder
	messageContent := fmt.Sprintf("🔔 **Memo** (scheduled for %s)\n```\n%s\n```",
		memo.RemindAt.In(loc).Format("Monday, January 2, 2006 at 15:04 MST"),
		memo.Content)
	_, err = c.session.ChannelMessageSend(memo.DiscordChannelID, messageContent)
	if err != nil {
		return fmt.Errorf("failed to send Discord message: %w", err)
	}
	return nil
}

// IsConnected checks if the Discord client is connected
func (c *Client) IsConnected() bool {
	return c.session != nil && c.session.State != nil && c.session.State.SessionID != ""
}

// AddMessageHandler adds a handler for incoming Discord messages
func (c *Client) AddMessageHandler(handler func(*discordgo.Session, *discordgo.MessageCreate)) {
	c.session.AddHandler(handler)
}

// GetChannelName retrieves the channel name for a given channel ID
func (c *Client) GetChannelName(channelID string) (string, error) {
	channel, err := c.session.Channel(channelID)
	if err != nil {
		return "", fmt.Errorf("failed to get channel info: %w", err)
	}
	return channel.Name, nil
}
