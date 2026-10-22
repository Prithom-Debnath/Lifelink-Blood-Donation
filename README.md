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
- [x] **Milestone 7: User registration and password hashing with bcrypt**
- [x] **Milestone 8: JWT authentication middleware and protected profile endpoint**
- [x] **Milestone 9: Blood compatibility matching logic**
- [x] **Milestone 10: Minimal web frontend**
- [x] **Milestone 11: Typed location entry and area-based donor search**
- [x] **Milestone 12: Blood request urgency levels (Normal, Urgent, Critical) with sorting and filtering**
- [x] **Milestone 13: Educational disclaimer and documentation review**

---

## Architecture Overview

The project uses a standard, beginner-friendly modular structure to separate concerns without unnecessary over-engineering:

```text
.
├── cmd/
│   └── server/
│       └── main.go         # Entry point; coordinates database, repositories, router, and graceful shutdown
├── internal/
│   ├── config/             # Reads environment variables with defaults (.env)
│   ├── database/           # MongoDB connection lifecycle, Ping reachability check, and cleanup
│   ├── models/             # Data models (User, Donor, BloodRequest structs with BSON/JSON tags, validation & bcrypt)
│   ├── repositories/       # MongoDB data access layer (User, Donor, and BloodRequest repositories)
│   ├── handlers/           # HTTP controllers (Auth, Health, Donor, Blood Request, and Match handlers)
│   ├── middleware/         # Authentication middleware (JWT verification, claims extraction, token generation)
│   └── routes/             # Gin engine router setup, static file serving, route grouping, and tests
├── web/                    # Simple static frontend files (HTML, CSS, browser JavaScript)
├── .env.example            # Sample environment variables
├── .gitignore              # Files excluded from git
├── go.mod                  # Go module definition
├── go.sum                  # Checksums for Go dependencies
└── README.md               # Project documentation
```

---

## Important Distinction: User Accounts vs. Volunteer Donors

In LifeLink, **User Accounts** and **Volunteer Donors** serve two distinct, complementary purposes:

| Feature | User Account (`/api/auth/*`) | Volunteer Blood Donor (`/api/donors`) |
|---|---|---|
| **Purpose** | Authentication, member profile management, and system login credentials. | Public registry of volunteer donors available to donate blood to recipients. |
| **Credentials** | Email and secure bcrypt-hashed password. | Contact phone, email, and city. |
| **Medical Info** | None (purely personal account). | ABO/Rh blood group (`A+`, `O-`, etc.) and active donation availability flag. |
| **Authentication** | Issues time-limited signed JWT tokens (`JWT_SECRET`). | Publicly queryable for emergency matching (`/api/matches`). |
| **Storage Collection** | MongoDB collection `users` | MongoDB collection `donors` |

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
| `JWT_SECRET` | Secret key used for signing and verifying JWT tokens | `supersecretdevkey_change_in_production` |

## Models & Repositories

### 1. User Model & Repository (`internal/models/user.go`, `internal/repositories/user_repository.go`)
Represents registered user accounts in MongoDB collection `users`:

| Field | Type | Description | BSON Key | JSON Key |
|---|---|---|---|---|
| `ID` | `bson.ObjectID` | Primary document identifier | `_id` | `id` |
| `Name` | `string` | User's full name | `name` | `name` |
| `Email` | `string` | User's email address (normalized to lowercase) | `email` | `email` |
| `Password` | `string` | Bcrypt hashed password | `password` | `json:"-"` *(omitted from all JSON responses)* |
| `CreatedAt` | `time.Time` | Account creation timestamp (UTC) | `created_at` | `created_at` |
| `UpdatedAt` | `time.Time` | Account last update timestamp (UTC) | `updated_at` | `updated_at` |

#### Security & Password Hashing:
- Passwords are **never stored in plain text**. During registration (`POST /api/auth/register`), plain-text passwords are encrypted with `bcrypt.GenerateFromPassword` using the default work factor.
- Passwords are **never returned in HTTP responses**. The `Password` field is tagged with `json:"-"`, ensuring neither plain-text passwords nor hashes are ever exposed over the wire.
- Password comparison during login (`POST /api/auth/login`) is performed securely using `bcrypt.CompareHashAndPassword`.

#### Repository Methods (`internal/repositories/user_repository.go`):
- `Create(ctx, user)`: Inserts a new user, ensuring case-insensitive email uniqueness.
- `FindByEmail(ctx, email)`: Retrieves user by case-insensitive email.
- `FindByID(ctx, id)`: Fetches a single user by hexadecimal ObjectID string.

#### Sentinel Errors:
- `repositories.ErrUserNotFound`: Account does not exist.
- `repositories.ErrUserAlreadyExists`: An account with this email address is already registered.
- `repositories.ErrInvalidUserID`: Provided ID string is not a 24-character hexadecimal MongoDB ObjectID.

---

### 2. Donor Model & Repository (`internal/models/donor.go`, `internal/repositories/donor_repository.go`)
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

### 3. Blood Request Model & Repository (`internal/models/blood_request.go`, `internal/repositories/blood_request_repository.go`)
Represents urgent blood requests in MongoDB collection `blood_requests`:

| Field | Type | Description | BSON Key | JSON Key |
|---|---|---|---|---|
| `ID` | `bson.ObjectID` | Primary document identifier | `_id` | `id` |
| `PatientName` | `string` | Patient or contact person's name | `patient_name` | `patient_name` / `patientName` |
| `BloodGroup` | `string` | Required blood group (e.g., `B+`, `O-`) | `blood_group` | `blood_group` / `bloodGroup` |
| `Urgency` | `string` | Urgency level: `normal` (default), `urgent`, `critical` | `urgency` | `urgency` |
| `Hospital` | `string` | Hospital or clinic name | `hospital` | `hospital` |
| `City` | `string` | City location | `city` | `city` |
| `ContactPhone` | `string` | Emergency contact phone number | `contact_phone` | `contact_phone` / `contactPhone` |
| `Status` | `string` | Request status (`open`, `fulfilled`, `cancelled`) | `status` | `status` |
| `CreatedAt` | `time.Time` | Creation timestamp (UTC) | `created_at` | `created_at` |
| `UpdatedAt` | `time.Time` | Last update timestamp (UTC) | `updated_at` | `updated_at` |

---

## Authentication & JWT Middleware (`internal/middleware/auth.go`)

LifeLink implements stateless JWT session authentication using the HMAC-SHA256 (`HS256`) signing algorithm and the configured `JWT_SECRET`:

- **Token Lifespan:** Login tokens are generated with a time-limited 24-hour expiration (`ExpiresAt`).
- **Claims:** Tokens embed the authenticated user's `UserID` and `Email`.
- **Authorization Header:** Protected endpoints require standard Bearer token formatting:
  ```http
  Authorization: Bearer <your_jwt_token_here>
  ```
- **Auth Middleware:** Verifies the token signature, checks expiration, and attaches the user's ID to the Gin request context (`c.Set("userID", claims.UserID)`).
- **Error Responses:** Missing, invalid, or expired tokens immediately return HTTP `401 Unauthorized`.

---

## API Endpoints

All API endpoints use the `/api` prefix and communicate using standard JSON.

### Authentication Endpoints

#### 1. Register User Account
- **Method & Path:** `POST /api/auth/register`
- **Status Codes:** `201 Created`, `400 Bad Request`, `409 Conflict`, `500 Internal Server Error`
- **Request Body:**
  ```json
  {
    "name": "Dr. Amina Rahman",
    "email": "amina@example.com",
    "password": "securepassword123"
  }
  ```
- **Validation Rules:**
  - `name`: Required, non-empty.
  - `email`: Required, valid email format.
  - `password`: Required, minimum 6 characters.
- **Response Example (201 Created):**
  ```json
  {
    "message": "User registered successfully",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6717a0b5f5e27a14e9123456",
      "name": "Dr. Amina Rahman",
      "email": "amina@example.com",
      "created_at": "2026-10-22T10:00:00Z",
      "updated_at": "2026-10-22T10:00:00Z"
    }
  }
  ```

#### 2. Log In
- **Method & Path:** `POST /api/auth/login`
- **Status Codes:** `200 OK`, `400 Bad Request`, `401 Unauthorized`, `500 Internal Server Error`
- **Request Body:**
  ```json
  {
    "email": "amina@example.com",
    "password": "securepassword123"
  }
  ```
- **Response Example (200 OK):**
  ```json
  {
    "message": "Login successful",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6717a0b5f5e27a14e9123456",
      "name": "Dr. Amina Rahman",
      "email": "amina@example.com",
      "created_at": "2026-10-22T10:00:00Z",
      "updated_at": "2026-10-22T10:00:00Z"
    }
  }
  ```
- **Invalid Credentials Response (401 Unauthorized):**
  ```json
  {
    "error": "Invalid email or password"
  }
  ```

#### 3. Get Authenticated User Profile (Protected)
- **Method & Path:** `GET /api/auth/me`
- **Headers:** `Authorization: Bearer <token>`
- **Status Codes:** `200 OK`, `401 Unauthorized`, `404 Not Found`, `500 Internal Server Error`
- **Response Example (200 OK):**
  ```json
  {
    "id": "6717a0b5f5e27a14e9123456",
    "name": "Dr. Amina Rahman",
    "email": "amina@example.com",
    "created_at": "2026-10-22T10:00:00Z",
    "updated_at": "2026-10-22T10:00:00Z"
  }
  ```

---

### Core Donation & Request Endpoints

#### 4. Health Check
- **Method & Path:** `GET /api/health`
- **Status Code:** `200 OK`
- **Response:** `{"status": "ok", "message": "LifeLink API is running"}`

#### 6. Register Volunteer Blood Donor
- **Method & Path:** `POST /api/donors`
- **Status Code:** `201 Created`
- **Request Body:**
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

#### 7. List All Donors
- **Method & Path:** `GET /api/donors`
- **Status Code:** `200 OK`

#### 8. Get Donor by ID
- **Method & Path:** `GET /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

#### 9. Update Donor
- **Method & Path:** `PUT /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

#### 10. Delete Donor
- **Method & Path:** `DELETE /api/donors/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

#### 11. Create Blood Request
- **Method & Path:** `POST /api/requests`
- **Status Code:** `201 Created`
- **Request Body:**
  ```json
  {
    "patient_name": "Rahim Uddin",
    "blood_group": "B+",
    "urgency": "critical",
    "hospital": "Dhaka Medical College Hospital",
    "city": "Dhaka",
    "contact_phone": "+8801711223344",
    "status": "open"
  }
  ```
- **Urgency Levels:** `normal` (default), `urgent`, `critical`.
- *Note:* Urgency level indicates operational triage priority and does **not** alter biological blood compatibility rules.

#### 12. List Blood Requests (with Urgency Filtering & Sorting)
- **Method & Path:** `GET /api/requests?urgency=critical&sort=urgency`
- **Query Parameters:**
  - `urgency`: Optional filter (`normal`, `urgent`, `critical`).
  - `sort`: Optional sort order (`urgency` sorts Critical > Urgent > Normal, then newest; default sorts newest first).
- **Status Code:** `200 OK`

#### 13. Get Blood Request by ID
- **Method & Path:** `GET /api/requests/:id`
- **Status Codes:** `200 OK`, `400 Bad Request`, `404 Not Found`

#### 14. Emergency Compatibility Matching by Typed Location
- **Method & Path:** `GET /api/matches?bloodGroup=B%2B&city=Gulshan`
- **Query Parameters:**
  - `bloodGroup`: Required ABO/Rh blood type (e.g., `B+`).
  - `city`: Optional text location such as an area, city, or both (`Gulshan, Dhaka`).
- **Location matching:** The search ignores letter case and matches the typed text within the donor's saved location. For example, searching `Dhaka` can find a donor who entered `Dhanmondi, Dhaka`.
- **Status Codes:** `200 OK`, `400 Bad Request`
- **Educational Disclaimer & Compatibility Reference Table:**
  > [!IMPORTANT]
  > **Educational Demonstration Disclaimer:** Blood compatibility rules in LifeLink represent simplified red blood cell antigen/antibody interactions for CSE 250 demonstration purposes. In real-world clinical medicine, blood transfusions require rigorous laboratory cross-matching, antibody screening, and medical oversight. Urgency levels do **not** alter compatibility rules.

  | Recipient Blood Group | Compatible Donor Blood Groups |
  |---|---|
  | `AB+` | `AB+`, `AB-`, `A+`, `A-`, `B+`, `B-`, `O+`, `O-` (Universal Recipient) |
  | `AB-` | `AB-`, `A-`, `B-`, `O-` |
  | `A+` | `A+`, `A-`, `O+`, `O-` |
  | `A-` | `A-`, `O-` |
  | `B+` | `B+`, `B-`, `O+`, `O-` |
  | `B-` | `B-`, `O-` |
  | `O+` | `O+`, `O-` |
  | `O-` | `O-` (Universal Donor) |
---|---|
  | `AB+` | `AB+`, `AB-`, `A+`, `A-`, `B+`, `B-`, `O+`, `O-` (Universal Recipient) |
  | `AB-` | `AB-`, `A-`, `B-`, `O-` |
  | `A+` | `A+`, `A-`, `O+`, `O-` |
  | `A-` | `A-`, `O-` |
  | `B+` | `B+`, `B-`, `O+`, `O-` |
  | `B-` | `B-`, `O-` |
  | `O+` | `O+`, `O-` |
  | `O-` | `O-` (Universal Donor) |

---

## Setup & Running Instructions

### 1. Configure Environment Variables
Copy `.env.example` to `.env` if you have not already:
```powershell
Copy-Item .env.example .env
```
Ensure `MONGO_URI` points to your running MongoDB instance and `JWT_SECRET` is set. No map service or map API key is needed.

### 2. Verify / Download Dependencies
```powershell
go mod tidy
```

### 3. Run the Server
```powershell
go run cmd/server/main.go
```
The server will start listening on port 8080 (e.g. `http://localhost:8080/`).

### 4. Access the Web Frontend
Open your web browser and navigate to:
```text
http://localhost:8080/
```

---

## Web Frontend Usage Guide

LifeLink includes a clean, beginner-friendly single-page frontend built exclusively with **plain HTML, CSS, and vanilla browser JavaScript (`fetch()`)**. No Node.js, npm, or frontend frameworks are used.

The top navigation opens a focused view for each item. It includes account pages, donor search, donor registration and directory, and blood request pages. Navigation updates the URL hash, so browser Back and Forward also switch between views.

### 1. User Account Portal:
- **Sign In:** Top navigation opens the email and password form. After login, LifeLink opens My Profile.
- **Create Account:** Top navigation opens the Name, Email, and Password form (min 6 characters). After registration, LifeLink opens My Profile.
- **My Profile:** Top navigation displays the signed-in user's name, email, unique account ID, and member creation timestamp, along with a "Log Out" button. If signed out, it asks the user to sign in first.
- **Header Status Indicator:** Shows the current authentication state (`"Guest"` vs. `"Signed in as [Name]"`).

### 2. Volunteer Blood Donor Registration:
- Register with a name, email, phone, blood group, and a location typed as an area and city (for example, `Dhanmondi, Dhaka`).
- Enter a familiar area name, city, or both so recipients can find the donor by typing part of that location.

### 3. Blood Request Submission:
- Submit the patient name, required blood group, urgency (`Normal`, `Urgent`, or `Critical`), hospital name, typed area and city, and emergency contact phone.
- The active request list provides urgency filtering and sorting.

### 4. Emergency Donor Matching:
- Select the recipient blood group and optionally type an area or city, such as `Gulshan` or `Dhaka`.
- Matching uses the typed text against donor locations without opening a map or calling a paid map service. Leaving the location blank searches all locations.
- The results table shows donor name, blood group, typed location, contact information, and availability.
## Running Tests

Run all unit and integration tests across the project:
```powershell
go test -v ./...
```
Or run focused tests for specific packages:
```powershell
# Test Models and bcrypt password hashing
go test -v ./internal/models

# Test Authentication middleware and JWT validation
go test -v ./internal/middleware

# Test User repository operations
go test -v ./internal/repositories

# Test Auth handlers (register, login, get profile)
go test -v ./internal/handlers -run TestAuthHandler

# Test Router integration
go test -v ./internal/routes
```

---

## Graceful Shutdown

When you press `Ctrl+C` (or send `SIGINT` / `SIGTERM`):
1. The server traps the signal and stops accepting new HTTP connections.
2. The server gives existing active requests up to 5 seconds to finish cleanly.
3. The MongoDB client disconnects its connection pool safely.
4. The server exits with code 0.
