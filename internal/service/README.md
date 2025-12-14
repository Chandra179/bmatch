# Internal Service
These 3 files mandatory for service, if service.go to big seperated into new files.

```
├── service/                           # service root directory
│   ├── auth/                          # example: package name auth
│	│   ├── handler.go                 # endpoint handler using gin
│	│   ├── service.go                 # service logic (business logic, query, etc..)
│	│   ├── types.go                   # struct, const, etc..

```

## Group service
current group service implementation "/internal/service/group/"

```go
type GroupMatcher interface {
	FindMatches(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error)
}
func (m *PostgresMatcher) FindMatches(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error)
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
func (s *Service) CreateGroup(ctx context.Context, ownerID string, req CreateGroupRequest) (*Group, error) 
func (s *Service) JoinGroup(ctx context.Context, groupID, userID string) error 
func (s *Service) ApplyToGroup(ctx context.Context, groupID, userID, pitch string) error 
func (s *Service) ApproveApplication(ctx context.Context, groupID, applicantUserID, ownerID string, approve bool) error
func (s *Service) DiscoverGroups(ctx context.Context, userProfile UserProfile, filters DiscoverGroupsRequest) ([]GroupMatch, error)
func (s *Service) GetGroup(ctx context.Context, groupID string) (*Group, error) 
func (s *Service) GetUserGroups(ctx context.Context, userID string) ([]*Group, error)
func (s *Service) GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error)

//internal/app/routes.go
func (o *Routes) setupGroupRoutes(auth *auth.Handler, gv *group.Service) {
	groupHandler := group.NewHandler(gv)

	o.r.GET("/groups/discover", groupHandler.DiscoverGroups)
	o.r.GET("/groups/:id", groupHandler.GetGroup)

	authorized := o.r.Group("/", auth.AuthMiddleware())
	{
		authorized.POST("/groups", groupHandler.CreateGroup)
		authorized.POST("/groups/:id/join", groupHandler.JoinGroup)
		authorized.POST("/groups/:id/apply", groupHandler.ApplyToGroup)
		authorized.POST("/groups/:id/applications/:user_id/approve", groupHandler.ApproveApplication)
		authorized.GET("/my-groups", groupHandler.GetMyGroups)
	}
}

```

## User Service
current group service implementation "/internal/service/user/"

```go
type Repository interface {
	GetUserByID(ctx context.Context, userID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	UpdateUserStats(ctx context.Context, userID string, stats Stats) error
}
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error)
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error)
func (s *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error)
func (s *Service) IncrementGroupsJoined(ctx context.Context, userID string) error
func (s *Service) IncrementGroupsCreated(ctx context.Context, userID string) error
func (s *Service) IncrementGroupsCompleted(ctx context.Context, userID string) error 

//internal/app/routes.go
func (o *Routes) setupUserRoutes(auth *auth.Handler, uv *user.Service) {
	userHandler := user.NewHandler(uv)

	o.r.GET("/users/:id", userHandler.GetUser)

	authorized := o.r.Group("/", auth.AuthMiddleware())
	{
		authorized.GET("/profile", userHandler.GetProfile)
		authorized.PUT("/profile", userHandler.UpdateProfile)
	}
}

```

## File Structure
### handler.go
```go
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}
```

### service.go
```go
type Service struct {
}

func NewService() *Service {
	return &Service{
	}
}
```

### types.go
```go
type A struct {
}

type B struct {
}

const (
)
```

### errors.go
```go
//example
var (
	ErrGroupNotFound      = errors.New("error1")
	ErrGroupFull          = errors.New("error2")
)
```