package user

import "time"

// Domain Models
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Tags         []string  `json:"tags"`
	SkillLevel   string    `json:"skill_level"`
	Availability []string  `json:"availability"`
	Intent       string    `json:"intent"`
	Stats        Stats     `json:"stats"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Stats struct {
	GroupsJoined    int `json:"groups_joined"`
	GroupsCreated   int `json:"groups_created"`
	GroupsCompleted int `json:"groups_completed"`
}

// DTOs
type CreateUserRequest struct {
	Email        string   `json:"email" binding:"required,email"`
	FullName     string   `json:"full_name" binding:"required,min=2,max=255"`
	Tags         []string `json:"tags" binding:"required,min=1,max=10"`
	SkillLevel   string   `json:"skill_level" binding:"required,oneof=BEGINNER INTERMEDIATE ADVANCED"`
	Availability []string `json:"availability" binding:"required,min=1"`
	Intent       string   `json:"intent" binding:"required,oneof=CASUAL SERIOUS"`
}

type UpdateUserRequest struct {
	FullName     string   `json:"full_name" binding:"omitempty,min=2,max=255"`
	Tags         []string `json:"tags" binding:"omitempty,min=1,max=10"`
	SkillLevel   string   `json:"skill_level" binding:"omitempty,oneof=BEGINNER INTERMEDIATE ADVANCED"`
	Availability []string `json:"availability" binding:"omitempty,min=1"`
	Intent       string   `json:"intent" binding:"omitempty,oneof=CASUAL SERIOUS"`
}

type UserProfileResponse struct {
	*User
}
