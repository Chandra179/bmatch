package group

import "errors"

var (
	// Group errors
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupFull          = errors.New("group capacity reached")
	ErrGroupNotOpen       = errors.New("group is not accepting members")
	ErrInvalidGroupStatus = errors.New("invalid group status")
	ErrInvalidJoinType    = errors.New("invalid join type")

	// Member errors
	ErrAlreadyMember     = errors.New("user already in group")
	ErrNotMember         = errors.New("user is not a member of this group")
	ErrNotGroupOwner     = errors.New("only owner can perform this action")
	ErrCannotLeaveAsOwner = errors.New("owner cannot leave group")

	// Application errors
	ErrApplicationExists    = errors.New("application already submitted")
	ErrApplicationNotFound  = errors.New("application not found")
	ErrInvalidApplicationStatus = errors.New("invalid application status")
	ErrCannotApplyToOpenGroup   = errors.New("cannot apply to open group, use join instead")
	ErrCannotJoinApplicationGroup = errors.New("cannot join application group, submit application instead")

	// Generic errors
	ErrInvalidInput     = errors.New("invalid input")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInternalError    = errors.New("internal server error")
)