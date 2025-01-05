# Email Tracking System

A lightweight Go server that provides email tracking functionality using tracking pixels. This system allows you to track when and from which IP addresses your emails are opened.

## Features

- Generate unique tracking pixels for emails
- Track email opens with IP address logging
- View detailed logs of email opens
- Web UI for easy management
- SQLite database for persistent storage
- Protection against sender self-opens
- Lightweight and fast response times

## Prerequisites

- Go 1.22 or higher (uses new router syntax)
- SQLite3

## Installation

1. Clone the repository
2. Install dependencies:
```bash
go mod init emailtracker
go get github.com/mattn/go-sqlite3
```

## Configuration

The server uses SQLite as its database. By default, it creates a database file named `emails.db` in the root directory. You can modify the database path in `main.go` if needed.

## Running the Server

```bash
go run .
```

The server will start on port 8080 by default.

## API Endpoints

### 1. Create Tracking Pixel
```
POST /create/{id}
```
Creates a new tracking entry for an email. The `id` should be a unique identifier for your email.

**Response:**
- 201: Successfully created
- 400: Missing ID
- 409: ID already exists
- 500: Server error

### 2. Track Email Opens
```
GET /track/{id}
```
Returns a 1x1 tracking pixel GIF and logs the email open. Place this URL in your email HTML:
```html
<img src="http://your-server:8080/track/{id}" />
```

**Response:**
- 200: Returns tracking pixel
- 400: Missing ID
- 404: Email ID not found
- 500: Server error

### 3. View Logs
```
GET /logs/{id}
```
Retrieves all opens for a specific email ID.

**Response:**
```json
[
  {
    "ID": "email_id_20240104150405_abc123",
    "EmailID": "email_id",
    "TimeStamp": "2024-01-04T15:04:05Z",
    "IP": "192.168.1.1"
  }
]
```

### 4. Web Interface
```
GET /
```
Provides a web interface for managing and viewing email tracking data.

## Security Features

- IP address logging for tracking opens
- Prevention of duplicate logging from sender's IP
- Transaction-based database operations
- Input validation
- No-cache headers for tracking pixels

## Database Schema

### Emails Table
```sql
CREATE TABLE emails (
    id TEXT PRIMARY KEY NOT NULL,
    ip TEXT NOT NULL,
    created DATETIME NOT NULL,
    last_read DATETIME
)
```

### Logs Table
```sql
CREATE TABLE logs (
    id TEXT PRIMARY KEY NOT NULL,
    email_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    ip TEXT NOT NULL,
    FOREIGN KEY (email_id) REFERENCES emails(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
)
```
