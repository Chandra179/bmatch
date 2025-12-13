package user

import (
	"context"
	"fmt"

	"bmatch/pkg/logger"
)

type Service struct {
	repo   Repository
	logger logger.Logger
}

func NewService(repo Repository, logger logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to get user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Error(ctx, "failed to get user by email",
			logger.Field{Key: "email", Value: email},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user's profile
func (s *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if len(req.Tags) > 0 {
		user.Tags = req.Tags
	}
	if req.SkillLevel != "" {
		user.SkillLevel = req.SkillLevel
	}
	if len(req.Availability) > 0 {
		user.Availability = req.Availability
	}
	if req.Intent != "" {
		user.Intent = req.Intent
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error(ctx, "failed to update user",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	s.logger.Info(ctx, "user updated", logger.Field{Key: "user_id", Value: userID})
	return user, nil
}

// IncrementGroupsJoined increments the groups joined counter
func (s *Service) IncrementGroupsJoined(ctx context.Context, userID string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Stats.GroupsJoined++
	return s.repo.UpdateUserStats(ctx, userID, user.Stats)
}

// IncrementGroupsCreated increments the groups created counter
func (s *Service) IncrementGroupsCreated(ctx context.Context, userID string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Stats.GroupsCreated++
	return s.repo.UpdateUserStats(ctx, userID, user.Stats)
}

// IncrementGroupsCompleted increments the groups completed counter
func (s *Service) IncrementGroupsCompleted(ctx context.Context, userID string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	user.Stats.GroupsCompleted++
	return s.repo.UpdateUserStats(ctx, userID, user.Stats)
}
