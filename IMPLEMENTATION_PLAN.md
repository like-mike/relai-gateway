# RelAI Gateway Database Implementation Plan

## Overview
This document outlines the step-by-step implementation plan to build the database schema and API handlers based on the DATABASE_SCHEMA_DESIGN.md.

## Phase 1: Database Setup

### 1.1 Create Database Schema File
**File**: `shared/db/schema.sql`
```sql
-- Complete database schema with all tables
-- Organizations, Users, Models, API Keys, etc.
-- Indexes and constraints
-- Sample data for testing
```

### 1.2 Update Database Initialization
**File**: `shared/db/init.go`
- Update to use new schema
- Add migration support
- Initialize with sample data

### 1.3 Create Migration System
**Directory**: `shared/db/migrations/`
- `001_initial_schema.up.sql`
- `001_initial_schema.down.sql`

## Phase 2: Go Models and Structs

### 2.1 Core Models
**Directory**: `shared/models/`

**File**: `shared/models/organization.go`
```go
type Organization struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Description *string   `json:"description" db:"description"`
    IsActive    bool      `json:"is_active" db:"is_active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
```

**File**: `shared/models/api_key.go`
```go
type APIKey struct {
    ID             string     `json:"id" db:"id"`
    Name           string     `json:"name" db:"name"`
    Description    *string    `json:"description" db:"description"`
    KeyHash        string     `json:"-" db:"key_hash"`
    KeyPrefix      string     `json:"key" db:"key_prefix"`
    OrganizationID string     `json:"organization_id" db:"organization_id"`
    UserID         *string    `json:"user_id" db:"user_id"`
    Type           string     `json:"type" db:"type"`
    MaxTokens      int        `json:"max_tokens" db:"max_tokens"`
    Permissions    []string   `json:"permissions" db:"permissions"`
    IsActive       bool       `json:"active" db:"is_active"`
    LastUsed       *time.Time `json:"last_used" db:"last_used"`
    CreatedAt      time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
    Organization   *Organization `json:"organization,omitempty"`
    User           *User      `json:"user,omitempty"`
}
```

**File**: `shared/models/model.go`
```go
type Model struct {
    ID            string          `json:"id" db:"id"`
    Name          string          `json:"name" db:"name"`
    Description   *string         `json:"description" db:"description"`
    Provider      string          `json:"provider" db:"provider"`
    ModelID       string          `json:"model_id" db:"model_id"`
    IsActive      bool            `json:"active" db:"is_active"`
    CreatedAt     time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
    Organizations []Organization  `json:"organizations,omitempty"`
}
```

**File**: `shared/models/user.go`
```go
type User struct {
    ID           string     `json:"id" db:"id"`
    Username     string     `json:"username" db:"username"`
    Email        *string    `json:"email" db:"email"`
    PasswordHash string     `json:"-" db:"password_hash"`
    IsActive     bool       `json:"is_active" db:"is_active"`
    LastLogin    *time.Time `json:"last_login" db:"last_login"`
    CreatedAt    time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}
```

**File**: `shared/models/quota.go`
```go
type OrganizationQuota struct {
    ID             string    `json:"id" db:"id"`
    OrganizationID string    `json:"organization_id" db:"organization_id"`
    TotalQuota     int       `json:"total_quota" db:"total_quota"`
    UsedTokens     int       `json:"used_tokens" db:"used_tokens"`
    ResetDate      time.Time `json:"reset_date" db:"reset_date"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type QuotaStats struct {
    TotalUsage     string `json:"total_usage"`
    RemainingQuota string `json:"remaining_quota"`
    PercentUsed    string `json:"percent_used"`
}
```

### 2.2 Database Operations
**File**: `shared/db/operations.go`
- CRUD operations for all models
- Complex queries with joins
- Transaction support

## Phase 3: API Handlers Implementation

### 3.1 Organizations Handler
**File**: `ui/routes/admin/organizations.go`
```go
func OrganizationsHandler(c *gin.Context) {
    // Get all organizations from database
    // Return JSON response
}
```

### 3.2 API Keys Handler Updates
**File**: `ui/routes/admin/api_keys.go` (Update existing)
```go
func APIKeysHandler(c *gin.Context) {
    // Get API keys from database with organization info
    // Render api-keys-table.html template with real data
}

func CreateAPIKeyHandler(c *gin.Context) {
    // Parse form data
    // Generate secure API key
    // Store in database
    // Return updated table HTML
}

func DeleteAPIKeyHandler(c *gin.Context) {
    // Delete API key from database
    // Return updated table HTML
}
```

### 3.3 Models Handler
**File**: `ui/routes/admin/models.go` (New file)
```go
func ModelsHandler(c *gin.Context) {
    // Get models from database with organization access
    // Return JSON response for JavaScript to render
}

func CreateModelHandler(c *gin.Context) {
    // Create new model in database
}

func DeleteModelHandler(c *gin.Context) {
    // Delete model from database
}

func ManageModelAccessHandler(c *gin.Context) {
    // Update model-organization access relationships
}
```

### 3.4 Quota Handler Updates
**File**: `ui/routes/admin/quota.go` (Update existing)
```go
func GetQuotaHandler(c *gin.Context) {
    // Get quota stats from database
    // Calculate usage percentages
    // Render quota-cards.html template with real data
}
```

## Phase 4: Database Integration

### 4.1 Update App Initialization
**File**: `ui/app.go`
- Ensure database connection is working
- Add error handling for database operations

### 4.2 Add Database Context
- Ensure all handlers have access to database connection
- Add proper error handling

## Phase 5: Frontend Integration

### 5.1 Update HTMX Endpoints
- Connect existing HTMX calls to real API handlers
- Update JavaScript in templates to handle real data

### 5.2 Test UI Workflows
- API Key creation flow
- Model management flow  
- Organization selection
- Quota display

## Phase 6: Testing and Validation

### 6.1 End-to-End Testing
- Test complete API key creation workflow
- Test model management with organization access
- Test quota calculations and display
- Test organization filtering

### 6.2 Data Validation
- Ensure proper data validation in API handlers
- Test error handling and display
- Verify data consistency

## Success Criteria

✅ Database schema created and initialized
✅ All Go models implemented with proper tags
✅ API handlers return real data from database
✅ UI forms create/update/delete real database records
✅ Organization filtering works across all pages
✅ Quota calculations are accurate and real-time
✅ API key generation and management is fully functional
✅ Model management with organization access control works

## Implementation Order

1. **Database Setup** (schema.sql, init.go)
2. **Core Models** (Go structs)
3. **Database Operations** (CRUD functions)
4. **API Handlers** (Connect to database)
5. **Frontend Integration** (Wire up HTMX)
6. **Testing** (End-to-end validation)

This plan provides a complete roadmap for implementing the database-backed API system that will replace the current placeholder endpoints with fully functional data management.