package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"memo-bot/internal/db"
)

type MemoService struct {
	queries db.Querier
}

func NewMemoService(dbConn *sql.DB) *MemoService {
	return &MemoService{
		queries: db.New(dbConn),
	}
}

func (s *MemoService) CreateUser(ctx context.Context, userID, username string) error {
	_, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		UserID:   userID,
		Username: username,
	})
	return err
}

func (s *MemoService) UpdateUserDiscordChannel(ctx context.Context, userID, channelID string) error {
	return s.queries.UpdateUserDiscordChannel(ctx, db.UpdateUserDiscordChannelParams{
		UserID: userID,
		DiscordChannelID: sql.NullString{
			String: channelID,
			Valid:  true,
		},
	})
}

func (s *MemoService) CreateMemo(ctx context.Context, discordUserID, discordChannelID, content string, remindAt time.Time) error {
	// Check if reminder time is in the past
	if remindAt.Before(time.Now()) {
		return fmt.Errorf("reminder time must be in the future")
	}

	_, err := s.queries.CreateMemo(ctx, db.CreateMemoParams{
		DiscordUserID:    discordUserID,
		DiscordChannelID: discordChannelID,
		Content:          content,
		RemindAt:         remindAt,
	})

	if err != nil {
		// Check for specific database errors and convert them to user-friendly messages
		if strings.Contains(err.Error(), "remind_at_check") {
			return fmt.Errorf("reminder time must be in the future")
		}
		// Add other specific error cases here if needed
		return fmt.Errorf("failed to create reminder: %v", err)
	}

	return nil
}

func (s *MemoService) ListPendingMemos(ctx context.Context, discordUserID, discordChannelID string) ([]db.Memo, error) {
	memos, err := s.queries.ListPendingMemos(ctx, db.ListPendingMemosParams{
		DiscordUserID:    discordUserID,
		DiscordChannelID: discordChannelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch your reminders: %v", err)
	}
	return memos, nil
}

func (s *MemoService) GetReminderCounts(ctx context.Context, discordUserID string) ([]db.GetReminderCountsRow, error) {
	counts, err := s.queries.GetReminderCounts(ctx, discordUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reminder counts: %v", err)
	}
	return counts, nil
}

func (s *MemoService) DeleteMemo(ctx context.Context, memoID int32, discordUserID string) error {
	err := s.queries.DeleteMemo(ctx, db.DeleteMemoParams{
		ID:            memoID,
		DiscordUserID: discordUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("reminder #%d not found or you don't have permission to delete it", memoID)
		}
		return fmt.Errorf("failed to delete reminder: %v", err)
	}
	return nil
}

func (s *MemoService) GetPendingReminders(ctx context.Context, now time.Time) ([]db.Memo, error) {
	return s.queries.GetPendingReminders(ctx, now)
}

func (s *MemoService) MarkMemoAsSent(ctx context.Context, memoID int32) error {
	return s.queries.MarkMemoAsSent(ctx, memoID)
}

// ListAllPendingMemosInChannel returns all pending memos in a specific channel
func (s *MemoService) ListAllPendingMemosInChannel(ctx context.Context, discordChannelID string) ([]db.Memo, error) {
	memos, err := s.queries.ListAllPendingMemosInChannel(ctx, discordChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending memos in channel: %w", err)
	}
	return memos, nil
}

// GetMemo returns a specific memo by ID
func (s *MemoService) GetMemo(ctx context.Context, memoID int32) (*db.Memo, error) {
	memo, err := s.queries.GetMemo(ctx, memoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reminder #%d not found", memoID)
		}
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}
	return &memo, nil
}
