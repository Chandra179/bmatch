package group

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"bmatch/pkg/db"
	"bmatch/pkg/logger"

	"github.com/lib/pq"
)

type Repository interface {
	// Group operations
	CreateGroup(ctx context.Context, tx *sql.Tx, group *Group) error
	GetGroupByID(ctx context.Context, groupID string) (*Group, error)
	GetGroupWithLock(ctx context.Context, tx *sql.Tx, groupID string) (*Group, error)
	UpdateGroup(ctx context.Context, tx *sql.Tx, group *Group) error
	FindGroupsByTags(ctx context.Context, tags []string, filters DiscoverGroupsRequest) ([]*Group, error)
	GetUserGroups(ctx context.Context, userID string) ([]*Group, error)

	// Member operations
	AddMember(ctx context.Context, tx *sql.Tx, member *GroupMember) error
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error)
	GetMemberCount(ctx context.Context, groupID string) (int, error)

	// Transaction helper
	WithTransaction(ctx context.Context, isolation sql.IsolationLevel, fn db.TxFunc) error
}

type repository struct {
	db     db.SQLExecutor
	logger logger.AppLogger
}

func NewRepository(database db.SQLExecutor, logger logger.AppLogger) Repository {
	return &repository{
		db:     database,
		logger: logger,
	}
}

// CreateGroup creates a new group
func (r *repository) CreateGroup(ctx context.Context, tx *sql.Tx, group *Group) error {
	tagsJSON, err := json.Marshal(group.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	appsJSON, err := json.Marshal(group.Applications)
	if err != nil {
		return fmt.Errorf("marshal applications: %w", err)
	}

	query := `
		INSERT INTO groups (id, owner_id, title, description, proposal, tags, capacity, current_count, join_type, status, applications)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		group.ID,
		group.OwnerID,
		group.Title,
		group.Description,
		group.Proposal,
		tagsJSON,
		group.Capacity,
		group.CurrentCount,
		group.JoinType,
		group.Status,
		appsJSON,
	).Scan(&group.CreatedAt, &group.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert group: %w", err)
	}

	return nil
}

// GetGroupByID retrieves a group by ID
func (r *repository) GetGroupByID(ctx context.Context, groupID string) (*Group, error) {
	query := `
		SELECT id, owner_id, title, description, proposal, tags, capacity, current_count, 
		       join_type, status, applications, created_at, updated_at
		FROM groups
		WHERE id = $1
	`

	var group Group
	var tagsJSON, appsJSON []byte

	err := r.db.QueryRowContext(ctx, query, groupID).Scan(
		&group.ID,
		&group.OwnerID,
		&group.Title,
		&group.Description,
		&group.Proposal,
		&tagsJSON,
		&group.Capacity,
		&group.CurrentCount,
		&group.JoinType,
		&group.Status,
		&appsJSON,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query group: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	if err := json.Unmarshal(appsJSON, &group.Applications); err != nil {
		return nil, fmt.Errorf("unmarshal applications: %w", err)
	}

	return &group, nil
}

// GetGroupWithLock retrieves a group with row-level lock for updates
func (r *repository) GetGroupWithLock(ctx context.Context, tx *sql.Tx, groupID string) (*Group, error) {
	query := `
		SELECT id, owner_id, title, description, proposal, tags, capacity, current_count, 
		       join_type, status, applications, created_at, updated_at
		FROM groups
		WHERE id = $1
		FOR UPDATE
	`

	var group Group
	var tagsJSON, appsJSON []byte

	err := tx.QueryRowContext(ctx, query, groupID).Scan(
		&group.ID,
		&group.OwnerID,
		&group.Title,
		&group.Description,
		&group.Proposal,
		&tagsJSON,
		&group.Capacity,
		&group.CurrentCount,
		&group.JoinType,
		&group.Status,
		&appsJSON,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query group with lock: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	if err := json.Unmarshal(appsJSON, &group.Applications); err != nil {
		return nil, fmt.Errorf("unmarshal applications: %w", err)
	}

	return &group, nil
}

// UpdateGroup updates a group
func (r *repository) UpdateGroup(ctx context.Context, tx *sql.Tx, group *Group) error {
	tagsJSON, err := json.Marshal(group.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	appsJSON, err := json.Marshal(group.Applications)
	if err != nil {
		return fmt.Errorf("marshal applications: %w", err)
	}

	query := `
		UPDATE groups
		SET title = $2, description = $3, proposal = $4, tags = $5, 
		    capacity = $6, current_count = $7, join_type = $8, status = $9, applications = $10
		WHERE id = $1
	`

	result, err := tx.ExecContext(ctx, query,
		group.ID,
		group.Title,
		group.Description,
		group.Proposal,
		tagsJSON,
		group.Capacity,
		group.CurrentCount,
		group.JoinType,
		group.Status,
		appsJSON,
	)

	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrGroupNotFound
	}

	return nil
}

// FindGroupsByTags finds groups using GIN index on tags
func (r *repository) FindGroupsByTags(ctx context.Context, tags []string, filters DiscoverGroupsRequest) ([]*Group, error) {
	fmt.Printf("DEBUG: Received tags: %+v (length: %d)\n", tags, len(tags))
	query := `
        SELECT id, owner_id, title, description, proposal, tags, capacity, current_count, 
               join_type, status, applications, created_at, updated_at
        FROM groups
        WHERE status = 'OPEN'
          AND current_count < capacity
    `

	args := []interface{}{}
	argIdx := 1

	if len(tags) > 0 {
		fmt.Printf("DEBUG: pq.Array(tags): %+v\n", pq.Array(tags))
		query += fmt.Sprintf(" AND tags ?| $%d", argIdx)
		args = append(args, pq.Array(tags)) // Use pq.Array instead of json.Marshal
		argIdx++
	}

	if filters.JoinType != "" {
		query += fmt.Sprintf(" AND join_type = $%d", argIdx)
		args = append(args, filters.JoinType)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	limit := filters.Limit
	if limit == 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	r.logger.Debug(ctx, "FindGroupsByTags", logger.Field{Key: "query", Value: query})
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer rows.Close()

	groups := make([]*Group, 0)
	for rows.Next() {
		var group Group
		var tagsJSON, appsJSON []byte

		err := rows.Scan(
			&group.ID,
			&group.OwnerID,
			&group.Title,
			&group.Description,
			&group.Proposal,
			&tagsJSON,
			&group.Capacity,
			&group.CurrentCount,
			&group.JoinType,
			&group.Status,
			&appsJSON,
			&group.CreatedAt,
			&group.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		if err := json.Unmarshal(appsJSON, &group.Applications); err != nil {
			return nil, fmt.Errorf("unmarshal applications: %w", err)
		}

		groups = append(groups, &group)
	}

	return groups, nil
}

// GetUserGroups retrieves all groups a user is a member of
func (r *repository) GetUserGroups(ctx context.Context, userID string) ([]*Group, error) {
	query := `
		SELECT g.id, g.owner_id, g.title, g.description, g.proposal, g.tags, g.capacity, 
		       g.current_count, g.join_type, g.status, g.applications, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
		ORDER BY gm.joined_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user groups: %w", err)
	}
	defer rows.Close()

	groups := make([]*Group, 0)
	for rows.Next() {
		var group Group
		var tagsJSON, appsJSON []byte

		err := rows.Scan(
			&group.ID,
			&group.OwnerID,
			&group.Title,
			&group.Description,
			&group.Proposal,
			&tagsJSON,
			&group.Capacity,
			&group.CurrentCount,
			&group.JoinType,
			&group.Status,
			&appsJSON,
			&group.CreatedAt,
			&group.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &group.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}

		if err := json.Unmarshal(appsJSON, &group.Applications); err != nil {
			return nil, fmt.Errorf("unmarshal applications: %w", err)
		}

		groups = append(groups, &group)
	}

	return groups, nil
}

// AddMember adds a member to a group
func (r *repository) AddMember(ctx context.Context, tx *sql.Tx, member *GroupMember) error {
	query := `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING joined_at
	`

	err := tx.QueryRowContext(ctx, query, member.GroupID, member.UserID, member.Role).Scan(&member.JoinedAt)
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}

	return nil
}

// IsMember checks if a user is a member of a group
func (r *repository) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, groupID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}

	return exists, nil
}

// GetGroupMembers retrieves all members of a group
func (r *repository) GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error) {
	query := `
		SELECT group_id, user_id, role, joined_at
		FROM group_members
		WHERE group_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()

	members := make([]*GroupMember, 0)
	for rows.Next() {
		var member GroupMember
		err := rows.Scan(&member.GroupID, &member.UserID, &member.Role, &member.JoinedAt)
		if err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, &member)
	}

	return members, nil
}

// GetMemberCount gets the current member count
func (r *repository) GetMemberCount(ctx context.Context, groupID string) (int, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, groupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}

	return count, nil
}

// WithTransaction executes a function within a database transaction
func (r *repository) WithTransaction(ctx context.Context, isolation sql.IsolationLevel, fn db.TxFunc) error {
	return r.db.WithTransaction(ctx, isolation, fn)
}
