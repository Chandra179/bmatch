package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"bmatch/pkg/db"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	UpdateUserStats(ctx context.Context, userID string, stats Stats) error
}

type repository struct {
	db db.SQLExecutor
}

func NewRepository(database db.SQLExecutor) Repository {
	return &repository{
		db: database,
	}
}

// CreateUser creates a new user
func (r *repository) CreateUser(ctx context.Context, user *User) error {
	tagsJSON, err := json.Marshal(user.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	availabilityJSON, err := json.Marshal(user.Availability)
	if err != nil {
		return fmt.Errorf("marshal availability: %w", err)
	}

	statsJSON, err := json.Marshal(user.Stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	query := `
		INSERT INTO users (id, email, full_name, tags, skill_level, availability, intent, stats)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.FullName,
		tagsJSON,
		user.SkillLevel,
		availabilityJSON,
		user.Intent,
		statsJSON,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (r *repository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT id, email, full_name, tags, skill_level, availability, intent, stats, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	var tagsJSON, availabilityJSON, statsJSON []byte

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&tagsJSON,
		&user.SkillLevel,
		&availabilityJSON,
		&user.Intent,
		&statsJSON,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &user.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	if err := json.Unmarshal(availabilityJSON, &user.Availability); err != nil {
		return nil, fmt.Errorf("unmarshal availability: %w", err)
	}

	if err := json.Unmarshal(statsJSON, &user.Stats); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, full_name, tags, skill_level, availability, intent, stats, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	var tagsJSON, availabilityJSON, statsJSON []byte

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&tagsJSON,
		&user.SkillLevel,
		&availabilityJSON,
		&user.Intent,
		&statsJSON,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &user.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}

	if err := json.Unmarshal(availabilityJSON, &user.Availability); err != nil {
		return nil, fmt.Errorf("unmarshal availability: %w", err)
	}

	if err := json.Unmarshal(statsJSON, &user.Stats); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}

	return &user, nil
}

// UpdateUser updates a user
func (r *repository) UpdateUser(ctx context.Context, user *User) error {
	tagsJSON, err := json.Marshal(user.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	availabilityJSON, err := json.Marshal(user.Availability)
	if err != nil {
		return fmt.Errorf("marshal availability: %w", err)
	}

	query := `
		UPDATE users
		SET full_name = $2, tags = $3, skill_level = $4, availability = $5, intent = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.FullName,
		tagsJSON,
		user.SkillLevel,
		availabilityJSON,
		user.Intent,
	)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateUserStats updates user statistics
func (r *repository) UpdateUserStats(ctx context.Context, userID string, stats Stats) error {
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	query := `UPDATE users SET stats = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, userID, statsJSON)
	if err != nil {
		return fmt.Errorf("update user stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
