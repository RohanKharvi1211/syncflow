# SyncFlow Frontend

React application built with Clean Architecture principles.

## Architecture

```
src/
├── core/              # Core configuration and types
├── domains/           # Domain-specific code (Clean Architecture)
│   ├── auth/
│   │   ├── entities/     # Domain models
│   │   ├── useCases/      # Business logic
│   │   └── adapters/      # API adapters
│   ├── companies/
│   ├── connections/
│   ├── pipelines/
│   └── integrations/
├── shared/            # Shared code
│   ├── components/    # Reusable UI components
│   ├── store/         # State management (Zustand)
│   └── adapters/      # Shared adapters (HTTP client)
└── pages/             # Page components
```

## Clean Architecture Layers

1. **Entities** - Domain models and business rules
2. **Use Cases** - Application-specific business logic
3. **Adapters** - Framework adapters (API, storage)
4. **Frameworks** - React components, routing

## Setup

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Environment Variables

Create a `.env` file:

```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_GOOGLE_CLIENT_ID=your-google-client-id
VITE_GOOGLE_REDIRECT_URI=http://localhost:8080/api/oauth/google/callback
```

## Features

- ✅ Authentication (Email & Google OAuth)
- ✅ Company Management
- ✅ Connection Management
- ✅ Pipeline Management
- ✅ Dashboard with Statistics
- ✅ Clean Architecture
- ✅ TypeScript
- ✅ Tailwind CSS
- ✅ React Query for data fetching
- ✅ Zustand for state management


