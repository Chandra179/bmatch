package group

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bmatch/pkg/cache"
	"bmatch/pkg/logger"

	"github.com/google/uuid"
)

type Service struct {
	repo    Repository
	matcher GroupMatcher
	cache   cache.Cache
	logger  logger.Logger
}

func NewService(repo Repository, matcher GroupMatcher, cache cache.Cache, logger logger.Logger) *Service {
	return &Service{
		repo:    repo,
		matcher: matcher,
		cache:   cache,
		logger:  logger,
	}
}

// CreateGroup creates a new group and automatically adds the owner as the first member
func (s *Service) CreateGroup(ctx context.Context, ownerID string, req CreateGroupRequest) (*Group, error) {
	// Validate join type
	if req.JoinType != JoinTypeOpen && req.JoinType != JoinTypeApplication {
		return nil, ErrInvalidJoinType
	}

	// Set default capacity if not provided
	if req.Capacity == 0 {
		req.Capacity = 5
	}

	group := &Group{
		ID:           uuid.New().String(),
		OwnerID:      ownerID,
		Title:        req.Title,
		Description:  req.Description,
		Proposal:     req.Proposal,
		Tags:         req.Tags,
		Capacity:     req.Capacity,
		CurrentCount: 1, // Owner is auto-member
		JoinType:     req.JoinType,
		Status:       StatusOpen,
		Applications: []Application{},
	}

	// Create group and add owner as member in transaction
	err := s.repo.WithTransaction(ctx, sql.LevelReadCommitted, func(ctx context.Context, tx *sql.Tx) error {
		if err := s.repo.CreateGroup(ctx, tx, group); err != nil {
			return fmt.Errorf("create group: %w", err)
		}

		// Add owner as LEADER
		member := &GroupMember{
			GroupID: group.ID,
			UserID:  ownerID,
			Role:    RoleLeader,
		}

		if err := s.repo.AddMember(ctx, tx, member); err != nil {
			return fmt.Errorf("add owner as member: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to create group", logger.Field{Key: "error", Value: err})
		return nil, err
	}

	s.logger.Info(ctx, "group created", logger.Field{Key: "group_id", Value: group.ID})
	return group, nil
}

// JoinGroup allows a user to join an OPEN group with ACID guarantees
func (s *Service) JoinGroup(ctx context.Context, groupID, userID string) error {
	// Execute join operation within serializable transaction
	err := s.repo.WithTransaction(ctx, sql.LevelSerializable, func(ctx context.Context, tx *sql.Tx) error {
		// 1. Lock the group row and get current state
		group, err := s.repo.GetGroupWithLock(ctx, tx, groupID)
		if err != nil {
			return err
		}

		// 2. Validate group state
		if group.Status != StatusOpen {
			return ErrGroupNotOpen
		}

		if group.JoinType != JoinTypeOpen {
			return ErrCannotJoinApplicationGroup
		}

		// 3. Check if already a member
		isMember, err := s.repo.IsMember(ctx, groupID, userID)
		if err != nil {
			return fmt.Errorf("check membership: %w", err)
		}
		if isMember {
			return ErrAlreadyMember
		}

		// 4. Check capacity
		if group.CurrentCount >= group.Capacity {
			return ErrGroupFull
		}

		// 5. Add member
		member := &GroupMember{
			GroupID: groupID,
			UserID:  userID,
			Role:    RoleMember,
		}

		if err := s.repo.AddMember(ctx, tx, member); err != nil {
			return fmt.Errorf("add member: %w", err)
		}

		// 6. Increment counter
		group.CurrentCount++
		if err := s.repo.UpdateGroup(ctx, tx, group); err != nil {
			return fmt.Errorf("update group count: %w", err)
		}

		// 7. Close group if full
		if group.CurrentCount >= group.Capacity {
			group.Status = StatusClosed
			if err := s.repo.UpdateGroup(ctx, tx, group); err != nil {
				return fmt.Errorf("close full group: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to join group",
			logger.Field{Key: "group_id", Value: groupID},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return err
	}

	// Invalidate cache
	s.cache.Del(ctx, fmt.Sprintf("group:%s", groupID))

	s.logger.Info(ctx, "user joined group",
		logger.Field{Key: "group_id", Value: groupID},
		logger.Field{Key: "user_id", Value: userID},
	)

	return nil
}

// ApplyToGroup submits an application to join an APPLICATION-type group
func (s *Service) ApplyToGroup(ctx context.Context, groupID, userID, pitch string) error {
	err := s.repo.WithTransaction(ctx, sql.LevelSerializable, func(ctx context.Context, tx *sql.Tx) error {
		// 1. Lock and get group
		group, err := s.repo.GetGroupWithLock(ctx, tx, groupID)
		if err != nil {
			return err
		}

		// 2. Validate group state
		if group.Status != StatusOpen {
			return ErrGroupNotOpen
		}

		if group.JoinType != JoinTypeApplication {
			return ErrCannotApplyToOpenGroup
		}

		// 3. Check if already a member
		isMember, err := s.repo.IsMember(ctx, groupID, userID)
		if err != nil {
			return fmt.Errorf("check membership: %w", err)
		}
		if isMember {
			return ErrAlreadyMember
		}

		// 4. Check if application already exists
		for _, app := range group.Applications {
			if app.UserID == userID && app.Status == ApplicationStatusPending {
				return ErrApplicationExists
			}
		}

		// 5. Check capacity (don't accept applications if full)
		if group.CurrentCount >= group.Capacity {
			return ErrGroupFull
		}

		// 6. Add application
		application := Application{
			UserID:    userID,
			Pitch:     pitch,
			Status:    ApplicationStatusPending,
			AppliedAt: time.Now(),
		}

		group.Applications = append(group.Applications, application)

		if err := s.repo.UpdateGroup(ctx, tx, group); err != nil {
			return fmt.Errorf("update group applications: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to apply to group",
			logger.Field{Key: "group_id", Value: groupID},
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return err
	}

	s.logger.Info(ctx, "application submitted",
		logger.Field{Key: "group_id", Value: groupID},
		logger.Field{Key: "user_id", Value: userID},
	)

	return nil
}

// ApproveApplication approves or rejects an application
func (s *Service) ApproveApplication(ctx context.Context, groupID, applicantUserID, ownerID string, approve bool) error {
	err := s.repo.WithTransaction(ctx, sql.LevelSerializable, func(ctx context.Context, tx *sql.Tx) error {
		// 1. Lock and get group
		group, err := s.repo.GetGroupWithLock(ctx, tx, groupID)
		if err != nil {
			return err
		}

		// 2. Verify owner
		if group.OwnerID != ownerID {
			return ErrNotGroupOwner
		}

		// 3. Find application
		applicationIndex := -1
		for i, app := range group.Applications {
			if app.UserID == applicantUserID && app.Status == ApplicationStatusPending {
				applicationIndex = i
				break
			}
		}

		if applicationIndex == -1 {
			return ErrApplicationNotFound
		}

		// 4. Update application status
		now := time.Now()
		if approve {
			group.Applications[applicationIndex].Status = ApplicationStatusApproved
			group.Applications[applicationIndex].DecidedAt = &now

			// 5. Check capacity before adding
			if group.CurrentCount >= group.Capacity {
				return ErrGroupFull
			}

			// 6. Add member
			member := &GroupMember{
				GroupID: groupID,
				UserID:  applicantUserID,
				Role:    RoleMember,
			}

			if err := s.repo.AddMember(ctx, tx, member); err != nil {
				return fmt.Errorf("add approved member: %w", err)
			}

			// 7. Increment counter
			group.CurrentCount++

			// 8. Close group if full
			if group.CurrentCount >= group.Capacity {
				group.Status = StatusClosed
			}
		} else {
			group.Applications[applicationIndex].Status = ApplicationStatusRejected
			group.Applications[applicationIndex].DecidedAt = &now
		}

		// 9. Update group
		if err := s.repo.UpdateGroup(ctx, tx, group); err != nil {
			return fmt.Errorf("update group: %w", err)
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to approve application",
			logger.Field{Key: "group_id", Value: groupID},
			logger.Field{Key: "applicant_id", Value: applicantUserID},
			logger.Field{Key: "owner_id", Value: ownerID},
			logger.Field{Key: "approve", Value: approve},
			logger.Field{Key: "error", Value: err},
		)
		return err
	}

	// Invalidate cache
	s.cache.Del(ctx, fmt.Sprintf("group:%s", groupID))

	s.logger.Info(ctx, "application processed",
		logger.Field{Key: "group_id", Value: groupID},
		logger.Field{Key: "applicant_id", Value: applicantUserID},
		logger.Field{Key: "approved", Value: approve},
	)

	return nil
}

// DiscoverGroups finds matching groups for a user
func (s *Service) DiscoverGroups(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error) {
	matches, err := s.matcher.FindMatches(ctx, userProfile, filters)
	if err != nil {
		s.logger.Error(ctx, "failed to discover groups",
			logger.Field{Key: "user_id", Value: userProfile.UserID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	s.logger.Info(ctx, "groups discovered",
		logger.Field{Key: "user_id", Value: userProfile.UserID},
		logger.Field{Key: "count", Value: len(matches)},
	)

	return matches, nil
}

// GetGroup retrieves a group by ID
func (s *Service) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	group, err := s.repo.GetGroupByID(ctx, groupID)
	if err != nil {
		s.logger.Error(ctx, "failed to get group",
			logger.Field{Key: "group_id", Value: groupID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	return group, nil
}

// GetUserGroups retrieves all groups a user is a member of
func (s *Service) GetUserGroups(ctx context.Context, userID string) ([]*Group, error) {
	groups, err := s.repo.GetUserGroups(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "failed to get user groups",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	return groups, nil
}

// GetGroupMembers retrieves all members of a group
func (s *Service) GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error) {
	members, err := s.repo.GetGroupMembers(ctx, groupID)
	if err != nil {
		s.logger.Error(ctx, "failed to get group members",
			logger.Field{Key: "group_id", Value: groupID},
			logger.Field{Key: "error", Value: err},
		)
		return nil, err
	}

	return members, nil
}
