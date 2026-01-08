# Frontend Clean Architecture Structure

This frontend follows the Clean Architecture pattern, organized by domain.

## Domain Structure

Each domain (authentication, integrations, sync, connections) follows this structure:

```
domain/
├── core/
│   └── lib/
│       ├── entities/       # Domain models and business rules
│       ├── useCases/       # Application-specific business rules
│       ├── services/       # Service interfaces
│       ├── adapters/       # Framework adapters
│       └── frameworks/     # Framework-specific setups
└── web/
    └── src/
        ├── components/     # React components
        ├── stylesheets/    # CSS files
        └── assets/         # Static assets
```

## Migration Status

- ✅ Structure created
- 🔄 Authentication domain - In progress
- ⏳ Integrations domain - Pending
- ⏳ Sync domain - Pending
- ⏳ Connections domain - Pending

## Current Implementation

The old HTML/JS files are still in the root `frontend/` directory and will be gradually migrated to the new structure.





