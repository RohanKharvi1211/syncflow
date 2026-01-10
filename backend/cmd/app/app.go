package app

import (
	"fmt"
	"net/http"
	"syncflow-backend/cmd/app/middlewares"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/controllers"
	authMiddleware "syncflow-backend/internal/middlewares"
	"syncflow-backend/internal/repositories"
	"syncflow-backend/internal/services"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	// Load HTML templates for OAuth callbacks
	gin.SetMode(gin.ReleaseMode)
}

type App struct {
	db          *gorm.DB
	repos       *repositories.Repositories
	controllers *controllers.Controllers
	middlewares *middlewares.Middlewares
	services    *services.Services
	router      *gin.Engine
	http        *http.Server
}

func (app *App) newDatabaseConnection(cfg *config.Config) {
	var err error
	// For RDS, use sslmode=require. For local dev, use sslmode=disable
	sslMode := "disable" // Default for local development
	if cfg.Database.Host != "localhost" && cfg.Database.Host != "127.0.0.1" {
		sslMode = "require" // Use SSL for RDS and other remote databases
	}
	
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		sslMode,
	)

	app.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("db initialization failed: %w", err))
	}

	// Set global DB for backward compatibility (used by controllers)
	config.DB = app.db

	// Enable UUID extension
	app.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// Lightweight startup migration(s)
	// NOTE:
	// We normally rely on SQL migration files + external migrate tool,
	// but some critical data fixes (like app definitions) can be safely
	// enforced here so the app \"just works\" even if migrate wasn't run.
	//
	// 1) Ensure QuickBooks app can be used as both source and destination
	//    (so it appears in the source dropdown and as a destination).
	app.db.Exec(`
		UPDATE "apps"
		SET "type" = 'both',
		    "description" = 'Sync data to/from QuickBooks accounting software'
		WHERE "name" = 'quickbooks' AND "type" <> 'both'
	`)

	// Auto migrations are handled via SQL migration files.
	// Skipping AutoMigrate here prevents accidental schema changes at runtime.
	fmt.Printf("Database connected successfully\n")
}

func (app *App) setUpHandlers(cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Note: HTML templates are served by frontend server, backend just redirects

	// Add middleware
	router.Use(app.middlewares.CORS)
	router.Use(app.middlewares.Logger)

	app.addRoutes(router)
	return router
}

func (app *App) newApp(cfg *config.Config) {
	app.newDatabaseConnection(cfg)

	// Create logger (simple implementation for now)
	baseLogger := &SimpleLogger{}
	repoLogger := &RepoLogger{SimpleLogger: baseLogger}
	serviceLogger := &ServiceLogger{SimpleLogger: baseLogger}

	app.repos = repositories.NewRepositories(app.db, repoLogger, cfg)
	app.services = services.NewServices(cfg, app.db, app.repos, serviceLogger)

	// Create controller logger
	controllerLogger := &ControllerLogger{SimpleLogger: baseLogger}
	app.controllers = controllers.NewControllers(cfg, controllerLogger, app.services)
	app.middlewares = middlewares.NewMiddlewares(cfg, baseLogger)
	app.router = app.setUpHandlers(cfg)
	app.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      app.router,
		ReadTimeout:  time.Second * 60,
		WriteTimeout: time.Second * 60,
		IdleTimeout:  time.Second * 60,
	}
}

func (app *App) start(cfg *config.Config) {
	defer func() {
		fmt.Printf("app shutting down: cleaning up\n")
	}()

	fmt.Printf("app initialized: running on port %d\n", cfg.ServerPort)

	if err := app.http.ListenAndServe(); err != nil {
		fmt.Printf("app failed to start: %v\n", err)
	}
}

func (app *App) NewApplication(cfg *config.Config) {
	app.newApp(cfg)
	app.start(cfg)
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (l *SimpleLogger) With(ctx interface{}) interface{} {
	return l
}

// RepoLogger wraps SimpleLogger for repositories
type RepoLogger struct {
	*SimpleLogger
}

func (l *RepoLogger) With(ctx interface{}) repositories.Logger {
	return l
}

// ServiceLogger wraps SimpleLogger for services
type ServiceLogger struct {
	*SimpleLogger
}

func (l *ServiceLogger) With(ctx interface{}) services.Logger {
	return l
}

// ControllerLogger wraps SimpleLogger for controllers
type ControllerLogger struct {
	*SimpleLogger
}

func (l *ControllerLogger) With(ctx interface{}) controllers.Logger {
	return l
}

func (l *ControllerLogger) WithRequest(c *gin.Context, r interface{}) interface{} {
	return c.Request.Context()
}

func (l *SimpleLogger) WithRequest(c *gin.Context, r interface{}) interface{} {
	return c.Request.Context()
}

func (app *App) addRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "syncflow-backend",
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Company routes
		companies := api.Group("/companies")
		{
			companies.GET("", app.controllers.Company.GetCompanies)
			companies.GET("/:id", app.controllers.Company.GetCompany)
			companies.POST("", app.controllers.Company.CreateCompany)
			companies.PUT("/:id", app.controllers.Company.UpdateCompany)
			companies.DELETE("/:id", app.controllers.Company.DeleteCompany)
		}

		// App routes
		apps := api.Group("/apps")
		{
			apps.GET("", app.controllers.App.GetApps)
			apps.GET("/:id", app.controllers.App.GetApp)
			apps.POST("", app.controllers.App.CreateApp)
			apps.PUT("/:id", app.controllers.App.UpdateApp)
			apps.DELETE("/:id", app.controllers.App.DeleteApp)
		}

		// User routes
		users := api.Group("/users")
		{
			users.GET("", app.controllers.User.GetUsers)
			users.GET("/:id", app.controllers.User.GetUser)
			users.POST("", app.controllers.User.CreateUser)
			users.PUT("/:id", app.controllers.User.UpdateUser)
			users.DELETE("/:id", app.controllers.User.DeleteUser)
		}

		// Connection routes (protected)
		connections := api.Group("/connections")
		connections.Use(authMiddleware.AuthMiddleware())
		{
			connections.GET("", app.controllers.Connection.GetConnections)
			connections.GET("/:id", app.controllers.Connection.GetConnection)
			connections.POST("", app.controllers.Connection.CreateConnection)
			connections.PUT("/:id", app.controllers.Connection.UpdateConnection)
			connections.DELETE("/:id", app.controllers.Connection.DeleteConnection)
		}

		// Google Sheets routes (protected)
		googleSheets := api.Group("/google-sheets")
		googleSheets.Use(authMiddleware.AuthMiddleware())
		{
			googleSheets.GET("/list", app.controllers.GoogleSheets.ListSheets)
			googleSheets.GET("/headers", app.controllers.GoogleSheets.GetSheetHeaders)
		}

		// Integration routes
		integrations := api.Group("/integrations")
		{
			integrations.GET("", app.controllers.Integration.GetIntegrations)
			integrations.GET("/:id", app.controllers.Integration.GetIntegration)
			integrations.POST("", app.controllers.Integration.CreateIntegration)
			integrations.PUT("/:id", app.controllers.Integration.UpdateIntegration)
			integrations.DELETE("/:id", app.controllers.Integration.DeleteIntegration)
		}

		// Sync routes
		sync := api.Group("/sync")
		{
			sync.POST("/integrations/:id", app.controllers.Sync.SyncIntegration)
			sync.GET("/jobs", app.controllers.Sync.GetSyncJobs)
			sync.GET("/jobs/:id", app.controllers.Sync.GetSyncJob)
			sync.GET("/records", app.controllers.Sync.GetSyncRecords)
			sync.GET("/records/:id", app.controllers.Sync.GetSyncRecord)
			sync.POST("/records/:id/retry", app.controllers.Sync.RetrySyncRecord)
			sync.GET("/stats", app.controllers.Sync.GetSyncStats)
		}

		// OAuth routes
		oauth := api.Group("/oauth")
		{
			oauth.GET("/google/initiate", app.controllers.OAuth.InitiateGoogleOAuth)
			oauth.GET("/google/callback", app.controllers.OAuth.GoogleOAuthCallback)
			oauth.GET("/quickbooks/initiate", app.controllers.OAuth.InitiateQuickBooksOAuth)
			oauth.GET("/quickbooks/callback", app.controllers.OAuth.QuickBooksOAuthCallback)
		}

		// Pipeline routes (new design, protected)
		pipelines := api.Group("/pipelines")
		pipelines.Use(authMiddleware.AuthMiddleware())
		{
			pipelines.GET("", app.controllers.Pipeline.GetPipelines)
			pipelines.GET("/:id", app.controllers.Pipeline.GetPipeline)
			pipelines.POST("", app.controllers.Pipeline.CreatePipeline)
			pipelines.PUT("/:id", app.controllers.Pipeline.UpdatePipeline)
			pipelines.DELETE("/:id", app.controllers.Pipeline.DeletePipeline)
			pipelines.POST("/:id/execute", app.controllers.Pipeline.ExecutePipeline)
			pipelines.GET("/:id/sync-runs", app.controllers.Pipeline.GetPipelineSyncRuns)
		}

		// Data Object routes (new design, protected)
		dataObjects := api.Group("/data-objects")
		dataObjects.Use(authMiddleware.AuthMiddleware())
		{
			dataObjects.GET("", app.controllers.DataObject.GetDataObjects)
			dataObjects.GET("/:id/fields", app.controllers.DataObject.GetFields) // Must be before /:id route
			dataObjects.GET("/:id", app.controllers.DataObject.GetDataObject)
			dataObjects.POST("", app.controllers.DataObject.CreateDataObject)
			dataObjects.PUT("/:id", app.controllers.DataObject.UpdateDataObject)
			dataObjects.DELETE("/:id", app.controllers.DataObject.DeleteDataObject)
		}
	}
}
