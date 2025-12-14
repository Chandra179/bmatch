package group

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateGroup handles POST /api/v1/groups
func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	group, err := h.service.CreateGroup(c.Request.Context(), userID.(string), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// JoinGroup handles POST /api/v1/groups/:id/join
func (h *Handler) JoinGroup(c *gin.Context) {
	groupID := c.Param("group_id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.service.JoinGroup(c.Request.Context(), groupID, userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully joined group"})
}

// ApplyToGroup handles POST /api/v1/groups/:id/apply
func (h *Handler) ApplyToGroup(c *gin.Context) {
	groupID := c.Param("group_id")

	var req ApplyToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.service.ApplyToGroup(c.Request.Context(), groupID, userID.(string), req.Pitch)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "application submitted successfully"})
}

// ApproveApplication handles POST /api/v1/groups/:id/applications/:user_id/approve
func (h *Handler) ApproveApplication(c *gin.Context) {
	groupID := c.Param("group_id")
	applicantUserID := c.Param("user_id")

	var req struct {
		Approve bool `json:"approve" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.service.ApproveApplication(c.Request.Context(), groupID, applicantUserID, ownerID.(string), req.Approve)
	if err != nil {
		h.handleError(c, err)
		return
	}

	status := "rejected"
	if req.Approve {
		status = "approved"
	}

	c.JSON(http.StatusOK, gin.H{"message": "application " + status})
}

// GetGroup handles GET /api/v1/groups/:id
func (h *Handler) GetGroup(c *gin.Context) {
	groupID := c.Param("group_id")

	group, err := h.service.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Get current user ID (optional, might not be authenticated)
	userID, _ := c.Get("user_id")

	// Get members
	members, err := h.service.GetGroupMembers(c.Request.Context(), groupID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Build response
	response := GroupResponse{
		Group:   group,
		Members: make([]GroupMemberResponse, len(members)),
	}

	for i, member := range members {
		response.Members[i] = GroupMemberResponse{
			UserID:   member.UserID,
			Role:     member.Role,
			JoinedAt: member.JoinedAt,
		}

		if userID != nil && member.UserID == userID.(string) {
			response.IsMember = true
			if member.Role == RoleLeader {
				response.IsOwner = true
			}
		}
	}

	// Check for pending application
	if userID != nil {
		for _, app := range group.Applications {
			if app.UserID == userID.(string) && app.Status == ApplicationStatusPending {
				response.PendingApplication = &app
				break
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// DiscoverGroups handles GET /api/v1/groups/discover
func (h *Handler) DiscoverGroups(c *gin.Context) {
	var filters DiscoverGroupsRequest
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID (optional for discovery)
	userID, exists := c.Get("user_id")

	// For now, use query tags as user profile
	// In production, fetch user profile from database
	userProfile := UserProfile{
		Tags: filters.Tags,
	}

	if exists {
		userProfile.UserID = userID.(string)
	}

	matches, err := h.service.DiscoverGroups(c.Request.Context(), userProfile, filters)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response := DiscoverGroupsResponse{
		Groups: matches,
		Total:  len(matches),
	}

	c.JSON(http.StatusOK, response)
}

// GetMyGroups handles GET /api/v1/my-groups
func (h *Handler) GetMyGroups(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	groups, err := h.service.GetUserGroups(c.Request.Context(), userID.(string))
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"total":  len(groups),
	})
}

// handleError maps domain errors to HTTP status codes
func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrGroupNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrGroupFull):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAlreadyMember):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotGroupOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrGroupNotOpen):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrApplicationExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrApplicationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrCannotApplyToOpenGroup):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrCannotJoinApplicationGroup):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidJoinType):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
