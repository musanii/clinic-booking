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
  "starts_at": "2026-10-20T10:00:00+03:00"
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
  "starts_at": "2026-10-20T11:00:00+03:00"
}
```

### View availability

`GET /doctors/1/availability?date=2026-10-20`

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

## CI/CD

GitHub Actions is intended to run automated tests and build checks on pull requests and pushes to the main branch.

## Limitations

This is an assessment-oriented MVP. Authentication, authorisation, holiday and leave management, and production observability may require additional implementation.
