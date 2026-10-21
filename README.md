# LifeLink — Blood Donation Management System

LifeLink is a lightweight, educational backend application built with **Go** and **Gin** for the CSE 250 course project. It is designed to demonstrate clean Go backend fundamentals, clean project separation, and RESTful API architecture.

---

## Current Status

- [x] **Milestone 1: Project skeleton and health endpoint**
- [x] **Milestone 2: MongoDB connection and clean shutdown**
- [x] **Milestone 3: Donor data model and repository**
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
│   ├── models/             # Data models (Donor struct with BSON/JSON tags)
│   ├── repositories/       # MongoDB data access layer (Donor CRUD operations and error handling)
│   ├── handlers/           # HTTP controllers handling request and response logic
│   ├── routes/             # Gin engine router setup, route grouping, and tests
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

## Donor Model & Repository (Milestone 3)

### Donor Model (`internal/models/donor.go`)
Represents donor documents stored in MongoDB collection `donors`:

| Field | Type | Description | BSON Key | JSON Key |
|---|---|---|---|---|
| `ID` | `bson.ObjectID` | Primary document identifier | `_id` | `id` |
| `Name` | `string` | Full name of donor | `name` | `name` |
| `Email` | `string` | Contact email address | `email` | `email` |
| `Phone` | `string` | Contact phone number | `phone` | `phone` |
| `BloodGroup` | `string` | ABO/Rh blood type (e.g., `O+`, `A-`) | `blood_group` | `blood_group` / `bloodGroup` |
| `City` | `string` | Donor location / city | `city` | `city` |
| `Availability` | `bool` | Whether donor is currently available to donate | `availability` | `availability` |
| `CreatedAt` | `time.Time` | Record creation timestamp (UTC) | `created_at` | `created_at` |
| `UpdatedAt` | `time.Time` | Record last modification timestamp (UTC) | `updated_at` | `updated_at` |

*Note: The model includes flexible JSON deserialization that accepts both snake_case (`blood_group`) and camelCase (`bloodGroup`).*

### Donor Repository (`internal/repositories/donor_repository.go`)
Provides clean data access methods using the official MongoDB Go driver:

- `Create(ctx, donor)`: Inserts a new donor record; automatically assigns an ObjectID and sets `CreatedAt` & `UpdatedAt` timestamps.
- `List(ctx)`: Returns all donor records as a slice (guarantees an empty slice `[]` rather than `null` when no documents exist).
- `FindByID(ctx, id)`: Finds a single donor by hex ObjectID string.
- `Update(ctx, id, donor)`: Modifies donor fields and updates the `UpdatedAt` timestamp.
- `Delete(ctx, id)`: Deletes a donor record by ID.

### Error Handling
The repository exports explicit sentinel errors so callers can handle edge cases cleanly:
- `repositories.ErrInvalidID`: Returned when an ID cannot be parsed as a 24-character hexadecimal MongoDB ObjectID.
- `repositories.ErrNotFound`: Returned when a query, update, or delete targets an ID that does not exist in the database.

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
*(HTTP handlers and REST endpoints for donor CRUD will be wired in Milestone 4.)*

---

## Running Tests

Run all unit and integration tests across the repository:
```powershell
go test -v ./...
```
*(Tests include route response validation, database connection lifecycle, model JSON deserialization, and full repository CRUD tests with invalid ID and missing record checks.)*
