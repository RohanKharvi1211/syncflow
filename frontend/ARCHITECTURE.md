# Frontend Clean Architecture

This document explains the Clean Architecture implementation in the SyncFlow frontend.

## Overview

The frontend is built with React, TypeScript, and follows Clean Architecture principles to ensure:
- **Separation of Concerns**: Business logic is separated from UI
- **Dependency Inversion**: Dependencies point inward (toward business logic)
- **Testability**: Business logic can be tested without UI
- **Independence**: Framework and UI can be swapped without changing business logic

## Architecture Layers

### 1. Entities (Domain Models)
**Location**: `src/domains/{domain}/entities/`

Pure TypeScript interfaces representing domain concepts. No dependencies on frameworks.

**Examples**:
- `User.ts` - User entity
- `Company.ts` - Company entity
- `Pipeline.ts` - Pipeline, DataObject, Checkpoint, SyncRun entities

### 2. Use Cases (Business Logic)
**Location**: `src/domains/{domain}/useCases/`

Contains application-specific business rules. Orchestrates entities and adapters.

**Examples**:
- `signInUseCase.ts` - Handles sign-in logic

### 3. Adapters (Interface Adapters)
**Location**: `src/domains/{domain}/adapters/` and `src/shared/adapters/`

Converts data between external systems (API, storage) and the application.

**Examples**:
- `authApi.ts` - API calls for authentication
- `pipelineApi.ts` - API calls for pipelines
- `httpClient.ts` - HTTP client wrapper

### 4. Frameworks (UI Layer)
**Location**: `src/pages/` and `src/shared/components/`

React components, routing, state management. This is the outermost layer.

**Examples**:
- `LoginPage.tsx` - Login page component
- `PipelinesPage.tsx` - Pipelines management page
- `Layout.tsx` - Shared layout component

## Domain Structure

Each domain follows this structure:

```
domains/
└── {domain}/
    ├── entities/       # Domain models
    ├── useCases/       # Business logic
    └── adapters/       # API adapters
```

### Current Domains

1. **auth** - Authentication and user management
2. **companies** - Company/tenant management
3. **connections** - OAuth connections management
4. **pipelines** - Pipeline and data object management
5. **integrations** - Integration management (legacy)

## Data Flow

```
User Action (UI)
    ↓
Page Component
    ↓
Use Case (Business Logic)
    ↓
Adapter (API Call)
    ↓
Backend API
```

## State Management

### Zustand Stores
**Location**: `src/shared/store/`

- `authStore.ts` - Authentication state
- `companyStore.ts` - Selected company state

### React Query
Used for server state management (caching, refetching, etc.)

## Key Principles

### 1. Dependency Rule
Dependencies point inward:
- UI depends on Use Cases
- Use Cases depend on Entities
- Adapters depend on Entities and Use Cases

### 2. Interface Segregation
Each adapter implements a specific interface. Use cases depend on interfaces, not implementations.

### 3. Single Responsibility
Each use case has one responsibility. Each entity represents one concept.

## Adding a New Feature

1. **Create Entity** (`entities/Feature.ts`)
   ```typescript
   export interface Feature {
     id: string;
     // ... properties
   }
   ```

2. **Create Adapter** (`adapters/featureApi.ts`)
   ```typescript
   class FeatureApi {
     async getFeatures(): Promise<Feature[]> {
       return httpClient.get('/features');
     }
   }
   ```

3. **Create Use Case** (`useCases/getFeaturesUseCase.ts`)
   ```typescript
   export class GetFeaturesUseCase {
     async execute(): Promise<Feature[]> {
       return featureApi.getFeatures();
     }
   }
   ```

4. **Create Page Component** (`pages/FeaturesPage.tsx`)
   ```typescript
   export function FeaturesPage() {
     const { data } = useQuery('features', () => {
       const useCase = new GetFeaturesUseCase();
       return useCase.execute();
     });
     // ... render
   }
   ```

## Benefits

1. **Testability**: Business logic can be tested without React
2. **Maintainability**: Clear separation makes code easier to understand
3. **Flexibility**: Can swap React for another framework without changing business logic
4. **Scalability**: Easy to add new features following the same pattern

## Technology Stack

- **React 18** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool
- **React Router** - Routing
- **React Query** - Server state management
- **Zustand** - Client state management
- **Tailwind CSS** - Styling
- **Axios** - HTTP client


