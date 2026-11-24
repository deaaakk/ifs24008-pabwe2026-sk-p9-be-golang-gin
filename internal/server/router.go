package server

import (
	_ "app/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
	"net/http"
	"time"

	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())
	au := &auth.Handler{S: &auth.Service{Repo: &auth.Repository{DB: db}, Key: cfg.Key}}
	po := &post.Handler{S: &post.Service{Repo: &post.Repository{DB: db}}}
	a := &ai.Handler{S: ai.NewService(cfg)}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/api.json", func(c *gin.Context) {
		d, _ := swag.ReadDoc()
		c.Data(http.StatusOK, "application/json", []byte(d))
	})
	api := r.Group("/api")
	pub := api.Group("/public")
	pub.GET("/categories", po.Categories)
	pub.GET("/posts", po.PublicList)
	pub.GET("/posts/:slug", po.PublicDetail)

	api.POST("/auth/register", au.Register)
	api.POST("/auth/login", middleware.RateLimiter(100, time.Minute), au.Login)
	sec := api.Group("", middleware.Auth(cfg.Key))
	sec.POST("/auth/logout", au.Logout)
	sec.GET("/auth/profile", au.Profile)
	sec.PUT("/auth/profile", au.UpdateProfile)

	w := sec.Group("/posts", middleware.RequireRole("admin", "member"))
	w.GET("", po.Mine)
	w.POST("", po.Create)
	w.GET("/stats", po.Stats)
	w.POST("/ai-generate", a.Generate)
	w.GET("/:id", po.Get)
	w.PUT("/:id", po.Update)
	w.PATCH("/:id", po.Update)
	w.PATCH("/:id/toggle-publish", po.Toggle)
	w.DELETE("/:id", po.Delete)

	ad := sec.Group("/admin", middleware.RequireRole("admin"))
	ad.GET("/users", po.AdminUsers)
	ad.GET("/posts", po.AdminPosts)
	ad.GET("/users/:id/posts", po.AdminPosts)

	r.NoRoute(func(c *gin.Context) { response.Fail(c, http.StatusNotFound, "Endpoint not found") })
	return r
}
