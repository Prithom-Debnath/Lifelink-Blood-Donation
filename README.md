# LifeLink — Blood Donation Management System

LifeLink is a lightweight, educational backend application built with **Go** and **Gin** for the CSE 250 course project. It is designed to demonstrate clean Go backend fundamentals, clean project separation, and RESTful API architecture.

---

## Current Status

- [x] **Milestone 1: Project skeleton and health endpoint**
- [x] **Milestone 2: MongoDB connection and clean shutdown**
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
│       └── main.go         # Entry point; coordinates database, router, and graceful shutdown
├── internal/
│   ├── config/             # Reads environment variables with defaults (.env)
│   ├── database/           # MongoDB connection lifecycle, Ping reachability check, and cleanup
│   ├── handlers/           # HTTP controllers handling request and response logic
│   ├── routes/             # Gin engine router setup, route grouping, and tests
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
- **MongoDB**: Community Server (v6.0+) running locally on port 27017, or a MongoDB Atlas connection string.
- **PowerShell** or any standard terminal

---

## MongoDB Setup Instructions

### Option A: Local MongoDB Windows Service (Standard)
If MongoDB Community Server is installed on Windows:
1. Verify the MongoDB service is running:
   ```powershell
   Get-Service -Name MongoDB
   ```
2. If it is stopped, start it with administrator privileges:
   ```powershell
   Start-Service -Name MongoDB
   ```

### Option B: Run MongoDB via Docker (Alternative)
If you prefer running MongoDB in a container:
```powershell
docker run -d --name lifelink-mongo -p 27017:27017 mongo:latest
```

---

## Environment Variables

Environment variables are loaded from the local `.env` file (which is excluded from Git via `.gitignore`). A template is provided in `.env.example`:

| Variable | Description | Default Value |
|---|---|---|
| `PORT` | HTTP server port | `8080` |
| `APP_ENV` | Application environment (`development`, `production`, `test`) | `development` |
| `MONGO_URI` | MongoDB connection string URI | `mongodb://localhost:27017` |
| `MONGO_DB_NAME` | MongoDB database name | `lifelink_db` |
| `JWT_SECRET` | Secret key used for signing JWT tokens | `supersecretdevkey_change_in_production` |

---

## Setup & Running

### 1. Configure Environment Variables
Copy `.env.example` to `.env` if you have not already:
```powershell
Copy-Item .env.example .env
```

### 2. Download Dependencies
```powershell
go mod tidy
```

### 3. Run the Server
```powershell
go run cmd/server/main.go
```

When started, the server will:
1. Load configuration from `.env`.
2. Connect to MongoDB and verify reachability using `Ping` with a 5-second deadline.
3. Start the HTTP server in a non-blocking goroutine on the specified port.
4. Listen for termination signals (`Ctrl+C`, `SIGINT`, `SIGTERM`) to trigger a clean shutdown of both the HTTP server and the MongoDB connection.

---

## Graceful Shutdown Behavior

When you press `Ctrl+C` in the terminal:
1. The server catches the interrupt signal (`os.Interrupt`).
2. The HTTP server stops accepting new connections and gives active requests up to 5 seconds to complete (`srv.Shutdown`).
3. The MongoDB client cleanly disconnects its connection pool (`db.Close()`).
4. The process terminates safely without leaking sockets or database connections.

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

Run all unit and integration tests across the repository:
```powershell
go test -v ./...
```
*(Tests include route response validation and database ping/connect/disconnect tests.)*
