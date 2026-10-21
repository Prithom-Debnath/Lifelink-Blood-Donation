# LifeLink — Blood Donation Management System

LifeLink is a lightweight, educational backend application built with **Go** and **Gin** for the CSE 250 course project. It is designed to demonstrate clean Go backend fundamentals, clean project separation, and RESTful API architecture.

---

## Current Status (Milestone 1)

- [x] **Milestone 1: Project skeleton and health endpoint**
- [ ] Milestone 2: MongoDB connection and clean shutdown
- [ ] Milestone 3: Donor data model and repository
- [ ] Milestone 4: Donor CRUD endpoints
- [ ] Milestone 5: Blood request model and endpoints
- [ ] Milestone 6: Input validation and standardized error handling
- [ ] Milestone 7: User registration and password hashing with bcrypt
- [ ] Milestone 8: JWT authentication middleware for write operations
- [ ] Milestone 9: Blood compatibility matching logic
- [ ] Milestone 10: Minimal web frontend
- [ ] Milestone 11: Final project cleanup and documentation review

---

## Architecture Overview

The project uses a standard, beginner-friendly modular structure to separate concerns without unnecessary over-engineering:

```text
.
├── cmd/
│   └── server/
│       └── main.go         # Application entry point; loads config and boots server
├── internal/
│   ├── config/             # Reads environment variables with defaults
│   ├── handlers/           # HTTP controllers handling request and response logic
│   ├── routes/             # Gin engine router setup, route grouping, and tests
│   ├── database/           # (Upcoming) Database connection & lifecycle
│   ├── models/             # (Upcoming) Data models (Donor, Request, User)
│   ├── repositories/       # (Upcoming) MongoDB data access layer
│   └── middleware/         # (Upcoming) Authentication and logging middleware
├── web/                    # (Upcoming) Simple static frontend files
├── .env.example            # Sample environment variables
├── .gitignore              # Files excluded from git
├── go.mod                  # Go module definition
├── go.sum                  # Checksums for Go dependencies
└── README.md               # Project documentation
```

---

## Prerequisites

- **Go**: Version 1.22+ (tested with Go 1.27)
- **PowerShell** or any standard terminal

---

## Setup & Running

### 1. Configure Environment Variables
Copy `.env.example` to `.env` (already done for local development):
```powershell
Copy-Item .env.example .env
```

Default values in `.env`:
```env
PORT=8080
APP_ENV=development
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=lifelink_db
JWT_SECRET=supersecretdevkey_change_in_production
```

### 2. Download Dependencies
```powershell
go mod tidy
```

### 3. Run the Server
```powershell
go run cmd/server/main.go
```

The server will start listening on `http://localhost:8080`.

---

## API Endpoints (Currently Implemented)

### Health Check
- **Endpoint:** `GET /api/health`
- **Authentication:** None
- **Response:** `200 OK`
```json
{
  "status": "ok",
  "message": "LifeLink API is running"
}
```

---

## Running Tests

Run all unit and integration tests using:
```powershell
go test -v ./...
```
