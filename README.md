# LifeLink — Blood Donation Management System

LifeLink is a lightweight, educational backend application built with **Go** and **Gin** for the CSE 250 course project. It is designed to demonstrate clean Go backend fundamentals, clean project separation, and RESTful API architecture.

---

## Current Status

- [x] **Milestone 1: Project skeleton and health endpoint**
- [x] **Milestone 2: MongoDB connection and clean shutdown**
- [x] **Milestone 3: Donor data model and repository**
- [x] **Milestone 4: Donor CRUD endpoints**
- [x] **Milestone 5 (Part 1): Blood request model and repository**
- [x] **Milestone 5 (Part 2): Blood request HTTP endpoints**
- [x] **Milestone 6: Input validation and standardized error handling**
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
│   ├── models/             # Data models (Donor, BloodRequest structs with BSON/JSON tags)
│   ├── repositories/       # MongoDB data access layer (Donor and BloodRequest repositories)
│   ├── handlers/           # HTTP controllers (Health, Donor, and Blood Request handlers)
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

## Models & Repositories

### 1. Donor Model & Repository (`internal/models/donor.go`)
Represents registered donors in MongoDB collection `donors`:

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

**Repository Methods:** `Create(ctx, donor)`, `List(ctx)`, `FindByID(ctx, id)`, `Update(ctx, id, donor)`, `Delete(ctx, id)`.

---

### 2. Blood Request Model & Repository (`internal/models/blood_request.go`)
Represents urgent blood requests in MongoDB collection `blood_requests`:

| Field | Type | Description | BSON Key | JSON Key |
|---|---|---|---|---|
| `ID` | `bson.ObjectID` | Primary document identifier | `_id` | `id` |
| `PatientName` | `string` | Patient or contact person's name | `patient_name` | `patient_name` / `patientName` |
| `BloodGroup` | `string` | Required blood group (e.g., `B+`, `O-`) | `blood_group` | `blood_group` / `bloodGroup` |
| `Hospital` | `string` | Hospital or clinic name | `hospital` | `hospital` |
| `City` | `string` | City location | `city` | `city` |
| `ContactPhone` | `string` | Emergency contact phone number | `contact_phone` | `contact_phone` / `contactPhone` |
| `Status` | `string` | Request status (`open`, `fulfilled`, `cancelled`) | `status` | `status` |
| `CreatedAt` | `time.Time` | Creation timestamp (UTC) | `created_at` | `created_at` |
| `UpdatedAt` | `time.Time` | Last update timestamp (UTC) | `updated_at` | `updated_at` |

**Status Values:**
- `open`: Request is active and awaiting matching donors (default on creation).
- `fulfilled`: Request has been fulfilled with successful donation.
- `cancelled`: Request is closed or no longer required.

**Repository Methods (`internal/repositories/blood_request_repository.go`):**
- `Create(ctx, req)`: Inserts a blood request, sets ObjectID, defaults empty status to `"open"`, and stamps UTC timestamps.
- `List(ctx)`: Retrieves all requests; guarantees an empty slice `[]` rather than `null` when no records exist.
- `FindByID(ctx, id)`: Fetches a single request by its hex ObjectID.

**Sentinel Errors:**
- `repositories.ErrBloodRequestNotFound`: Record with given ID does not exist.
- `repositories.ErrInvalidBloodRequestID`: ID cannot be parsed as a 24-character hexadecimal ObjectID.

---

## Input Validation Rules

LifeLink enforces basic, consistent input validation across both Donor and Blood Request APIs:

### 1. Donor Validation (`POST /api/donors`, `PUT /api/donors/:id`)
- **`name`**: Required string (non-empty, non-whitespace).
- **`email`**: Required string; must match standard email address format (e.g. `user@example.com`).
- **`phone`**: Required string (non-empty).
- **`blood_group`**: Required string; must be one of the 8 supported ABO/Rh groups: `A+`, `A-`, `B+`, `B-`, `AB+`, `AB-`, `O+`, `O-`.
- **`city`**: Required string (non-empty).

### 2. Blood Request Validation (`POST /api/requests`)
- **`patient_name`**: Required string (non-empty).
- **`blood_group`**: Required string; must be one of the 8 supported ABO/Rh groups: `A+`, `A-`, `B+`, `B-`, `AB+`, `AB-`, `O+`, `O-`.
- **`hospital`**: Required string (non-empty).
- **`city`**: Required string (non-empty).
- **`contact_phone`**: Required string (non-empty).
- **`status`**: Optional on creation (defaults to `open`); if provided, must be one of: `open`, `fulfilled`, `cancelled`.

### 3. Standard Error Format
When validation fails, the API responds with HTTP `400 Bad Request` and a clear error message in JSON:
```json
{
  "error": "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-"
}
```

---

## API Endpoints (Currently Implemented)

All API endpoints use the `/api` prefix and return standard JSON.

### 1. Health Check
- **Method & Path:** `GET /api/health`
- **Status Code:** `200 OK`
- **Response Example:**
  ```json
  {
    "status": "ok",
    "message": "LifeLink API is running"
  }
  ```

### 2. Create Donor
- **Method & Path:** `POST /api/donors`
- **Status Code:** `201 Created`
- **Request Body Example:**
  ```json
  {
    "name": "Tanvir Hasan",
    "email": "tanvir@example.com",
    "phone": "+8801712345678",
    "blood_group": "A+",
    "city": "Dhaka",
    "availability": true
  }
  ```

### 3. List All Donors
- **Method & Path:** `GET /api/donors`
- **Status Code:** `200 OK`

### 4. Get Donor by ID
- **Method & Path:** `GET /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

### 5. Update Donor
- **Method & Path:** `PUT /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

### 6. Delete Donor
- **Method & Path:** `DELETE /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

### 7. Create Blood Request
- **Method & Path:** `POST /api/requests`
- **Status Codes:** `201 Created`, `400 Bad Request`, `500 Internal Server Error`
- **Request Body Example (snake_case or camelCase supported):**
  ```json
  {
    "patient_name": "Rahim Uddin",
    "blood_group": "B+",
    "hospital": "Dhaka Medical College Hospital",
    "city": "Dhaka",
    "contact_phone": "+8801711223344"
  }
  ```
- **Response Example (201 Created):**
  ```json
  {
    "id": "6717a0b5f5e27a14e9123456",
    "patient_name": "Rahim Uddin",
    "blood_group": "B+",
    "hospital": "Dhaka Medical College Hospital",
    "city": "Dhaka",
    "contact_phone": "+8801711223344",
    "status": "open",
    "created_at": "2026-10-22T09:15:00Z",
    "updated_at": "2026-10-22T09:15:00Z"
  }
  ```

### 8. List All Blood Requests
- **Method & Path:** `GET /api/requests`
- **Status Codes:** `200 OK`, `500 Internal Server Error`
- **Response Example (200 OK):**
  ```json
  [
    {
      "id": "6717a0b5f5e27a14e9123456",
      "patient_name": "Rahim Uddin",
      "blood_group": "B+",
      "hospital": "Dhaka Medical College Hospital",
      "city": "Dhaka",
      "contact_phone": "+8801711223344",
      "status": "open",
      "created_at": "2026-10-22T09:15:00Z",
      "updated_at": "2026-10-22T09:15:00Z"
    }
  ]
  ```
  *(Returns `[]` empty JSON array when no blood requests are present.)*

### 9. Get Blood Request by ID
- **Method & Path:** `GET /api/requests/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`
- **Response Example (200 OK):**
  ```json
  {
    "id": "6717a0b5f5e27a14e9123456",
    "patient_name": "Rahim Uddin",
    "blood_group": "B+",
    "hospital": "Dhaka Medical College Hospital",
    "city": "Dhaka",
    "contact_phone": "+8801711223344",
    "status": "open",
    "created_at": "2026-10-22T09:15:00Z",
    "updated_at": "2026-10-22T09:15:00Z"
  }
  ```
- **Error Responses:**
  - `400 Bad Request` (Invalid hex ObjectID format):
    ```json
    {
      "error": "Invalid blood request ID format"
    }
    ```
  - `404 Not Found` (Record does not exist):
    ```json
    {
      "error": "Blood request not found"
    }
    ```
  - `500 Internal Server Error` (Unexpected database error):
    ```json
    {
      "error": "Failed to retrieve blood request"
    }
    ```

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

## Running Tests

Run all unit and integration tests across the repository:
```powershell
go test -v ./...
```
*(Tests include route response validation, database connection lifecycle, model JSON deserialization, donor repository CRUD, donor HTTP handlers, blood request repository operations, blood request HTTP handlers, model validation methods, and HTTP input validation tests.)*
