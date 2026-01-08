# Backend Restructuring Summary

This document outlines the restructuring of the backend to follow the be-ledger service architecture pattern.

## New Directory Structure

```
backend/
├── cmd/
│   ├── app/
│   │   ├── app.go              # Application initialization
│   │   ├── routes.go           # Route definitions
│   │   └── middlewares/        # HTTP middlewares
│   └── server/
│       └── main.go             # Application entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── controllers/            # HTTP handlers (thin layer)
│   ├── models/                 # Domain models
│   ├── repositories/           # Data access layer
│   ├── requests/               # Request DTOs
│   ├── response/               # Response DTOs
│   └── services/               # Business logic
└── pkg/                        # Shared utilities
```

## Architecture Pattern

The code follows a clean architecture pattern:

1. **Controllers** - Handle HTTP requests, decode requests, call services, return responses
2. **Services** - Contain business logic, call repositories
3. **Repositories** - Handle database operations
4. **Models** - Domain entities
5. **Requests/Responses** - DTOs for API boundaries

## Completed

- ✅ Created directory structure
- ✅ Moved models to `internal/models/`
- ✅ Moved config to `internal/config/`
- ✅ Moved controllers to `internal/controllers/`
- ✅ Moved services to `internal/services/`
- ✅ Created `cmd/app/app.go` and `cmd/server/main.go`
- ✅ Created access types for each layer
- ✅ Created main.go files for repositories, services, controllers

## TODO

### 1. Update Import Paths
All files need their import paths updated from:
- `syncflow-backend/models` → `syncflow-backend/internal/models`
- `syncflow-backend/config` → `syncflow-backend/internal/config`
- `syncflow-backend/handlers` → `syncflow-backend/internal/controllers`
- `syncflow-backend/services` → `syncflow-backend/internal/services`

### 2. Create Repository Layer
Create repository implementations for:
- `internal/repositories/repo_user.go`
- `internal/repositories/repo_connection.go`
- `internal/repositories/repo_integration.go`
- `internal/repositories/repo_sync_job.go`
- `internal/repositories/repo_sync_record.go`

Each should implement the interface defined in `main.go`.

### 3. Refactor Controllers
Update existing controllers in `internal/controllers/` to:
- Use the `ControllerAccess` pattern
- Implement interfaces defined in `main.go`
- Use request/response DTOs
- Call services instead of directly accessing DB

### 4. Refactor Services
Update existing services in `internal/services/` to:
- Use the `ServiceAccess` pattern
- Implement interfaces defined in `main.go`
- Call repositories instead of directly accessing DB
- Remove direct DB access

### 5. Create Request/Response DTOs
Create DTOs in:
- `internal/requests/` - for incoming requests
- `internal/response/` - for API responses

### 6. Update Database Connection
Update `internal/config/database.go` to:
- Return `*gorm.DB` instead of global variable
- Be called from `cmd/app/app.go`

### 7. Remove Old Files
Clean up:
- Old `main.go` in root
- Old `config/` directory (if still exists)
- Old `handlers/` directory (if still exists)
- Old `services/` directory (if still exists)
- Old `middleware/` directory (if still exists)

## Migration Strategy

1. Keep old `main.go` working during migration
2. Gradually move functionality to new structure
3. Test each layer as it's migrated
4. Once complete, remove old files

## Notes

- The new structure follows the be-ledger pattern exactly
- All layers use dependency injection via access structs
- Interfaces are defined in main.go files for each layer
- Controllers are thin, services contain business logic, repositories handle data access





