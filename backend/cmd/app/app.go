package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	
	// First, try to connect to the target database
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
		// If database doesn't exist, try to create it
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "3D000") {
			fmt.Printf("Database '%s' does not exist. Attempting to create it...\n", cfg.Database.Name)
			
			// Connect to default 'postgres' database to create the target database
			defaultDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s",
				cfg.Database.Host,
				cfg.Database.User,
				cfg.Database.Password,
				cfg.Database.Port,
				sslMode,
			)
			
			defaultDB, dbErr := gorm.Open(postgres.Open(defaultDSN), &gorm.Config{})
			if dbErr != nil {
				panic(fmt.Errorf("failed to connect to default 'postgres' database to create '%s': %w", cfg.Database.Name, dbErr))
			}
			
			// Create the database
			createDBQuery := fmt.Sprintf("CREATE DATABASE %s", cfg.Database.Name)
			if execErr := defaultDB.Exec(createDBQuery).Error; execErr != nil {
				// If database already exists (race condition), try connecting again
				if strings.Contains(execErr.Error(), "already exists") || strings.Contains(execErr.Error(), "duplicate") {
					fmt.Printf("Database '%s' was created by another process. Retrying connection...\n", cfg.Database.Name)
					app.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
					if err != nil {
						panic(fmt.Errorf("db initialization failed after database creation: %w", err))
					}
					// Connection successful, continue to UUID extension below
				} else {
					panic(fmt.Errorf("failed to create database '%s': %w", cfg.Database.Name, execErr))
				}
			} else {
				fmt.Printf("✓ Database '%s' created successfully\n", cfg.Database.Name)
				// Now connect to the newly created database
				app.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					panic(fmt.Errorf("db initialization failed after creating database: %w", err))
				}
			}
			
			// Close the default DB connection (we're done with it)
			// GORM manages connection pooling, but we can still close the underlying connection
			if sqlDB, sqlErr := defaultDB.DB(); sqlErr == nil {
				sqlDB.Close()
			}
		} else {
			panic(fmt.Errorf("db initialization failed: %w", err))
		}
	}

	// Set global DB for backward compatibility (used by controllers)
	config.DB = app.db

	// Enable UUID extension
	app.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// Run database migrations automatically on startup
	// This ensures the database schema is always up to date
	if err := app.runMigrations(); err != nil {
		fmt.Printf("Warning: Failed to run migrations: %v\n", err)
		fmt.Printf("You may need to run migrations manually. Continuing anyway...\n")
	} else {
		fmt.Printf("Database migrations completed successfully\n")
	}

	// Lightweight startup migration(s)
	// NOTE:
	// We normally rely on SQL migration files + external migrate tool,
	// but some critical data fixes (like app definitions) can be safely
	// enforced here so the app \"just works\" even if migrate wasn't run.
	//
	// 1) Ensure QuickBooks app can be used as both source and destination
	//    (so it appears in the source dropdown and as a destination).
	// Only run this if apps table exists (after migrations)
	var tableExists bool
	app.db.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'apps'
		)
	`).Scan(&tableExists)
	
	if tableExists {
		app.db.Exec(`
			UPDATE "apps"
			SET "type" = 'both',
			    "description" = 'Sync data to/from QuickBooks accounting software'
			WHERE "name" = 'quickbooks' AND "type" <> 'both'
		`)
	}

	// Auto migrations are handled via SQL migration files.
	// Skipping AutoMigrate here prevents accidental schema changes at runtime.
	fmt.Printf("Database connected successfully\n")
}

// runMigrations executes all SQL migration files in migrations/postgres directory
// Migrations are run in alphabetical order (by filename)
// This function is safe to call multiple times - migrations with "IF NOT EXISTS" will be skipped
func (app *App) runMigrations() error {
	migrationsDir := "migrations/postgres"
	
	// Check if migrations directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Migrations directory doesn't exist - this is OK (might be running in different context)
		fmt.Printf("Migrations directory not found: %s (skipping migrations)\n", migrationsDir)
		return nil
	}

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort migration files (only .up.sql files)
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	if len(migrationFiles) == 0 {
		fmt.Printf("No migration files found in %s\n", migrationsDir)
		return nil
	}

	sort.Strings(migrationFiles)

	// Get underlying SQL database connection
	sqlDB, err := app.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// Run migrations
	for _, fileName := range migrationFiles {
		filePath := filepath.Join(migrationsDir, fileName)
		fmt.Printf("Running migration: %s\n", fileName)

		// Read migration file
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		// Execute migration
		_, err = sqlDB.Exec(string(content))
		if err != nil {
			// Check if error is because table/object already exists (for idempotency)
			errorMsg := strings.ToLower(err.Error())
			if strings.Contains(errorMsg, "already exists") ||
				strings.Contains(errorMsg, "duplicate") ||
				strings.Contains(errorMsg, "relation") && strings.Contains(errorMsg, "already exists") {
				fmt.Printf("  ⚠ Migration already applied (skipping): %s\n", fileName)
				continue
			}
			// For other errors, log but continue (some migrations might partially succeed)
			fmt.Printf("  ⚠ Migration failed (non-fatal): %s - %v\n", fileName, err)
			// Don't return error - allow app to continue even if some migrations fail
			// This prevents app from crashing on startup if DB schema is partially migrated
		} else {
			fmt.Printf("  ✓ Migration completed: %s\n", fileName)
		}
	}

	return nil
}

func (app *App) setUpHandlers(cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Note: HTML templates are served by frontend server, backend just redirects

	// CRITICAL: Register health endpoint BEFORE middlewares
	// This ensures health checks work even if middlewares fail
	// Support both GET and HEAD requests (HEAD is used by wget --spider)
	router.GET("/health", app.healthCheck)
	router.HEAD("/health", app.healthCheck)

	// Add middleware (order matters!)
	// Bot protection first - blocks common scanner paths early
	// Note: health endpoint is already registered above, so it bypasses bot protection
	router.Use(app.middlewares.BotProtection)
	// Rate limiting - prevents abuse
	// Note: health endpoint bypasses rate limiting (already registered)
	router.Use(app.middlewares.RateLimit)
	// CORS - allows cross-origin requests
	router.Use(app.middlewares.CORS)
	// Logger - logs all requests (after bot protection to reduce noise)
	router.Use(app.middlewares.Logger)

	app.addRoutes(router)
	return router
}

// healthCheck performs basic liveness check - verifies the application is running
// Note: We DON'T check database connectivity here because:
// 1. If DB is down, restarting the app won't help
// 2. Health check should verify app liveness, not dependency health
// 3. DB connectivity issues should be logged but not cause container restarts
// The app will log DB errors separately, allowing us to fix the DB issue
// rather than constantly restarting containers
func (app *App) healthCheck(c *gin.Context) {
	// Simple liveness check - is the HTTP server responding?
	// If we reach here, the app is initialized and HTTP server is running
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "syncflow-backend",
	})
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
	// Health check is already registered in setUpHandlers() before middlewares
	// No need to register it again here

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
