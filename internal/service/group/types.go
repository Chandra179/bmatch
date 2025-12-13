package group

import "time"

// Enums
const (
	// Skill Levels
	SkillLevelBeginner     = "BEGINNER"
	SkillLevelIntermediate = "INTERMEDIATE"
	SkillLevelAdvanced     = "ADVANCED"

	// Intent
	IntentCasual  = "CASUAL"
	IntentSerious = "SERIOUS"

	// Join Types
	JoinTypeOpen        = "OPEN"
	JoinTypeApplication = "APPLICATION"

	// Group Status
	StatusOpen      = "OPEN"
	StatusClosed    = "CLOSED"
	StatusCompleted = "COMPLETED"

	// Member Roles
	RoleLeader = "LEADER"
	RoleMember = "MEMBER"

	// Application Status
	ApplicationStatusPending  = "PENDING"
	ApplicationStatusApproved = "APPROVED"
	ApplicationStatusRejected = "REJECTED"
)

// Domain Models
type Group struct {
	ID           string        `json:"id"`
	OwnerID      string        `json:"owner_id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Proposal     string        `json:"proposal"`
	Tags         []string      `json:"tags"`
	Capacity     int           `json:"capacity"`
	CurrentCount int           `json:"current_count"`
	JoinType     string        `json:"join_type"`
	Status       string        `json:"status"`
	Applications []Application `json:"applications,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type GroupMember struct {
	GroupID  string    `json:"group_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type Application struct {
	UserID    string    `json:"user_id"`
	Pitch     string    `json:"pitch"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"applied_at"`
	DecidedAt *time.Time `json:"decided_at,omitempty"`
}

type GroupMatch struct {
	Group           *Group  `json:"group"`
	SimilarityScore float64 `json:"similarity_score"`
}

// DTOs
type CreateGroupRequest struct {
	Title       string   `json:"title" binding:"required,min=3,max=255"`
	Description string   `json:"description" binding:"required,min=10"`
	Proposal    string   `json:"proposal" binding:"required,min=20"`
	Tags        []string `json:"tags" binding:"required,min=1,max=10"`
	Capacity    int      `json:"capacity" binding:"omitempty,min=2,max=10"`
	JoinType    string   `json:"join_type" binding:"required,oneof=OPEN APPLICATION"`
}

type JoinGroupRequest struct {
	GroupID string `json:"group_id" binding:"required,uuid"`
}

type ApplyToGroupRequest struct {
	GroupID string `json:"group_id" binding:"required,uuid"`
	Pitch   string `json:"pitch" binding:"required,min=50,max=500"`
}

type ApproveApplicationRequest struct {
	GroupID string `json:"group_id" binding:"required,uuid"`
	UserID  string `json:"user_id" binding:"required,uuid"`
	Approve bool   `json:"approve" binding:"required"`
}

type DiscoverGroupsRequest struct {
	Tags       []string `json:"tags" form:"tags"`
	SkillLevel string   `json:"skill_level" form:"skill_level"`
	JoinType   string   `json:"join_type" form:"join_type"`
	Limit      int      `json:"limit" form:"limit" binding:"omitempty,min=1,max=100"`
}

type GroupResponse struct {
	*Group
	Members          []GroupMemberResponse `json:"members,omitempty"`
	IsMember         bool                  `json:"is_member"`
	IsOwner          bool                  `json:"is_owner"`
	PendingApplication *Application        `json:"pending_application,omitempty"`
}

type GroupMemberResponse struct {
	UserID   string    `json:"user_id"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type DiscoverGroupsResponse struct {
	Groups []GroupMatch `json:"groups"`
	Total  int          `json:"total"`
}

type UserProfile struct {
	UserID       string   `json:"user_id"`
	Tags         []string `json:"tags"`
	SkillLevel   string   `json:"skill_level"`
	Availability []string `json:"availability"`
	Intent       string   `json:"intent"`
}