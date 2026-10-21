# Clinic Appointment Booking API

A REST API built with Go and PostgreSQL for managing clinic appointments. It supports doctor availability, appointment booking, cancellation, rescheduling, and viewing a patient's upcoming appointments.

## Features

- View a doctor's available appointment slots for a date.
- Book 30-minute appointments during a doctor's working hours.
- Prevent duplicate bookings for the same doctor and start time.
- Cancel appointments with a cancellation reason.
- Reschedule active appointments.
- Retrieve a patient's upcoming appointments in chronological order.
- Validate appointment times, working hours, and booking conflicts.

## Tech Stack

- **Language:** Go
- **HTTP:** Go standard library `net/http`
- **Database:** PostgreSQL
- **PostgreSQL driver:** `pgx`
- **Clinic timezone:** `Africa/Nairobi`

## Prerequisites

- Go installed
- PostgreSQL installed and running
- A PostgreSQL database and user with the required table permissions

## Configuration

Set `DATABASE_URL` to your PostgreSQL connection string.

PowerShell:

```powershell
$env:DATABASE_URL = "postgres://USER:URL_ENCODED_PASSWORD@localhost:5432/DATABASE?sslmode=disable"
```

Replace the placeholders with your local database details. Do not commit credentials to source control.

## Running the API

From the project root:

```powershell
go run ./cmd/api
```

The API should start on the address configured by the application.

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/doctors/{id}/availability?date=YYYY-MM-DD` | View available slots |
| `POST` | `/appointments` | Book an appointment |
| `PATCH` | `/appointments/{id}/cancel` | Cancel an appointment |
| `PATCH` | `/appointments/{id}/reschedule` | Reschedule an appointment |
| `GET` | `/patients/{id}/appointments` | View upcoming patient appointments |

### Book an appointment

`POST /appointments`

Example JSON:

```json
{
  "doctor_id": 1,
  "patient_id": 1,
  "starts_at": "2026-10-22T10:00:00+03:00"
}
```

### Cancel an appointment

`PATCH /appointments/1/cancel`

```json
{
  "reason": "Patient requested cancellation"
}
```

### Reschedule an appointment

`PATCH /appointments/1/reschedule`

```json
{
  "starts_at": "2026-10-22T11:00:00+03:00"
}
```

### View availability

`GET /doctors/1/availability?date=2026-10-22`

### View upcoming patient appointments

`GET /patients/1/appointments`

## Validation and errors

The API validates doctor and patient IDs, appointment times, working hours, and booking conflicts.

Errors use a JSON structure similar to:

```json
{
  "error": {
    "code": "slot_unavailable",
    "message": "This appointment slot is already booked."
  }
}
```

Common HTTP status codes include:

- `200 OK` — successful retrieval or update
- `201 Created` — appointment created
- `204 No Content` — cancellation succeeded
- `400 Bad Request` — invalid input
- `404 Not Found` — resource not found
- `409 Conflict` — slot unavailable or incompatible appointment state
- `500 Internal Server Error` — unexpected server error

## Testing

Run all tests:

```powershell
go test ./...
```

Build all packages:

```powershell
go build ./...
```

Some scheduling tests require `DATABASE_URL` and a working PostgreSQL database.

## Design Notes

- Appointment slots are 30 minutes long.
- Working hours are stored by weekday.
- Times are validated using the `Africa/Nairobi` timezone.
- PostgreSQL enforces uniqueness for active appointments at the same doctor's start time.
- Cancelled appointments are excluded from active slot conflicts.

## System Architecture and Trade-offs

### Architecture

The application separates HTTP handling, scheduling logic, and data persistence:

- `cmd/api` starts the HTTP server and establishes the PostgreSQL connection.
- `internal/httpapi` handles HTTP requests, parses JSON, validates request formats, and returns appropriate HTTP status codes.
- `internal/scheduling` implements appointment booking, availability, cancellation, rescheduling, and working-hours rules.
- PostgreSQL stores doctors, patients, working hours, and appointments.

This separation makes scheduling rules easier to test independently of HTTP request handling.

### Key decisions and trade-offs

**PostgreSQL for persistence:** Appointment data is relational, and database constraints help maintain consistency. PostgreSQL introduces database setup and operational overhead, but provides durable storage and transactional operations.

**Database-level protection against double-booking:** A partial unique index prevents multiple active appointments for the same doctor and start time. This adds a second layer of protection beyond application validation. The current approach assumes fixed 30-minute appointments; variable durations would require a more flexible conflict-detection strategy.

**Fixed appointment durations:** Using 30-minute slots simplifies availability calculation and makes scheduling predictable. The trade-off is that appointments requiring different durations are not supported by the current model.

**Recurring weekly working hours:** Working periods are stored by doctor and weekday, allowing doctors to have different schedules. Holidays, leave, and one-off schedule changes would require additional scheduling rules or data structures.

**Managed deployment:** Render provides a public API endpoint and managed PostgreSQL without requiring the project to operate its own server infrastructure. This simplifies deployment for the assessment, although service availability and resource limits depend on the selected plan.

### Independent Technical Decisions

**1. Choosing PostgreSQL for concurrency safety**

I chose PostgreSQL because application-level validation in Go alone cannot reliably prevent duplicate bookings when requests arrive concurrently. Two requests could both check the same slot before either booking is saved. I therefore used database-level unique indexes as an additional safeguard, so the database can reject conflicting active appointments even if application checks race. This taught me that data integrity should be enforced at the database level as well as in application logic.

**2. Choosing Go as the implementation language**

I chose Go because I have been learning it and wanted this project to be part of that learning journey. Rather than treating the assessment only as a task to complete, I wanted to practise applying Go to a practical backend API, including HTTP handlers, database access, error handling, and tests. This meant balancing the need to deliver a working application with the goal of understanding the code I was writing.

### AI Usage Reflection

**1. A helpful AI suggestion**

- **Prompt:** “Let's put it on Render first. Walk me through the steps.”
- **Suggestion:** During deployment, configure the Go HTTP server to read the `PORT` environment variable supplied by Render, while retaining a local default port.
- **How I used it:** I updated the server configuration, ran `go test ./...` and `go build ./...`, and deployed the application to Render. The service started successfully and became accessible through its public URL.
- **What I learned:** Deploying an application requires accounting for the hosting environment, not just making it work locally.

**2. An incomplete AI troubleshooting attempt**

While testing appointment cancellation, the API returned HTTP 400 even though I supplied a cancellation reason. The initial troubleshooting did not immediately identify the underlying problem. Inspecting the cancellation handler revealed that it decoded the same request body twice. The first decode consumed the body, so the second decode encountered an end-of-file error.

I removed the duplicate decode, pushed the fix, and tested the endpoint again. Cancellation then succeeded, and checking the doctor's availability confirmed that the released appointment slot became available again.

This experience taught me not to treat an AI-generated explanation as proof. I need to reproduce the problem, inspect the relevant code, check the response, and verify the fix with a real test.

**Conclusion**

I used AI as a development assistant, but I remained responsible for technical decisions and verification. The project reinforced the importance of understanding the code, testing assumptions, and combining application-level checks with database-level integrity constraints.

## Deployment

**Public API URL:** https://clinic-booking-b0zc.onrender.com

The API is deployed on Render and uses a managed PostgreSQL database. The application reads its database connection string from the `DATABASE_URL` environment variable and listens on the port supplied by Render through `PORT` (defaulting to `8080` for local development).

### CI/CD

GitHub Actions runs automated tests and a build check on pull requests targeting `main` and on pushes to `main`.

Render is configured to deploy the `main` branch after CI checks pass. Merging a pull request into `main` therefore triggers the CI workflow, followed by deployment when the required checks succeed.

### Verifying the deployment

Availability can be checked with:

`GET https://clinic-booking-b0zc.onrender.com/doctors/1/availability?date=2026-10-22`

The deployed API has been manually tested for availability lookup, appointment booking, cancellation, and rescheduling.

## Limitations

This is an assessment-oriented MVP. Authentication, authorisation, holiday and leave management, and production observability may require additional implementation.
