package app

import (
	"bmatch/internal/service/auth"
	"bmatch/internal/service/group"
	"bmatch/internal/service/user"
	"bmatch/pkg/oauth2"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Routes struct {
	r *gin.Engine
}

func NewRoutes(r *gin.Engine) *Routes {
	return &Routes{
		r: r,
	}
}

func (o *Routes) setupInfraRoutes() {
	o.r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	o.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	o.r.GET("/docs", docsHandler)
}

func (o *Routes) setupAuthRoutes(handler *auth.Handler, oauth2mgr *oauth2.Manager) {
	auth := o.r.Group("/auth")
	{
		auth.POST("/login", handler.LoginHandler())
		auth.POST("/logout", handler.LogoutHandler())
		auth.GET("/callback/google", oauth2.GoogleCallbackHandler(oauth2mgr))
	}
}

func docsHandler(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>...`
	c.String(200, html)
}

// setupUserRoutes registers user-related endpoints
func (o *Routes) setupUserRoutes(auth *auth.Handler, uv *user.Service) {
	userHandler := user.NewHandler(uv)

	o.r.GET("/users/:id", userHandler.GetUser)

	authorized := o.r.Group("/", auth.AuthMiddleware())
	{
		authorized.GET("/profile", userHandler.GetProfile)
		authorized.PUT("/profile", userHandler.UpdateProfile)
	}
}

// setupGroupRoutes registers group-related endpoints
func (o *Routes) setupGroupRoutes(auth *auth.Handler, gv *group.Service) {
	groupHandler := group.NewHandler(gv)

	o.r.GET("/groups/discover", groupHandler.DiscoverGroups)
	o.r.GET("/groups/:id", groupHandler.GetGroup)

	authorized := o.r.Group("/", auth.AuthMiddleware())
	{
		authorized.POST("/groups", groupHandler.CreateGroup)
		authorized.POST("/groups/:id/join", groupHandler.JoinGroup)
		authorized.POST("/groups/:id/apply", groupHandler.ApplyToGroup)

		// Application management (owner only, validated in handler)
		authorized.POST("/groups/:id/applications/:user_id/approve", groupHandler.ApproveApplication)

		// User's groups
		authorized.GET("/my-groups", groupHandler.GetMyGroups)
	}
}
