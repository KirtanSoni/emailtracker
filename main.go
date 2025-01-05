package main

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed ui.html
var uihtml embed.FS

type Server struct {
	db *DB
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewServer(dbPath string) (*Server, error) {
	db, err := InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}
	return &Server{db: db}, nil
}

func sendJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func (s *Server) UIHandler(w http.ResponseWriter, r *http.Request) {
   
    file, err := uihtml.ReadFile("ui.html")
    if err != nil {
        sendJSONError(w, http.StatusInternalServerError, "Failed to load UI")
        return
    }
    
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write(file)
}


func (s *Server) TrackHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONError(w, http.StatusBadRequest, "ID is required")
		return
	}

	// Get IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	// Check if email exists first
	exists, err := s.db.EmailExists(id)
	if err != nil {
		log.Printf("Error checking email existence: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if !exists {
		sendJSONError(w, http.StatusNotFound, "Email ID not found")
		return
	}

	// Log the read
	err = s.db.LogRead(id, ip)
	if err != nil {
		log.Printf("Error logging read: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Failed to log email read")
		return
	}

	// Serve tracking pixel
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Content-Length", "35")
    w.Header().Set("Connection", "close") 
	w.Write([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b})
}

func (s *Server) CreateTracker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		sendJSONError(w, http.StatusBadRequest, "ID is required")
		return
	}

	// Check if email already exists
	exists, err := s.db.EmailExists(id)
	if err != nil {
		log.Printf("Error checking email existence: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if exists {
		sendJSONError(w, http.StatusConflict, "Email ID already exists")
		return
	}
		
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	err = s.db.Track(id,ip)
	if err != nil {
		log.Printf("Error creating tracker: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Failed to create tracker")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) ShowLogs(w http.ResponseWriter, r *http.Request) {

	id := r.PathValue("id")
	if id == "" {
		sendJSONError(w, http.StatusBadRequest, "ID is required")
		return
	}

	// Check if email exists
	exists, err := s.db.EmailExists(id)
	if err != nil {
		log.Printf("Error checking email existence: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if !exists {
		sendJSONError(w, http.StatusNotFound, "Email ID not found")
		return
	}

	logs, err := s.db.ShowLogs(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			sendJSONError(w, http.StatusNotFound, "No logs found for this email")
			return
		}
		log.Printf("Error getting logs: %v", err)
		sendJSONError(w, http.StatusInternalServerError, "Failed to retrieve logs")
		return
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func main() {
	server, err := NewServer("emails.db")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Register routes
	http.HandleFunc("GET /", server.UIHandler)
	http.HandleFunc("GET /track/{id}", server.TrackHandler)
	http.HandleFunc("POST /create/{id}", server.CreateTracker)
	http.HandleFunc("GET /logs/{id}", server.ShowLogs)

	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}


type Email struct {
	ID       string
	Created  time.Time
	IP string
	LastRead sql.NullTime
}

type LogEntry struct {
	ID        string     
	EmailID   string
	TimeStamp time.Time
	IP        string
}

type DB struct {
	*sql.DB
}

func InitDB(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}


	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}


	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS emails (
			id TEXT PRIMARY KEY NOT NULL,
			ip TEXT NOT NULL,
			created DATETIME NOT NULL,
			last_read DATETIME
		)
	`)
	if err != nil {
		return nil, err
	}


	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id TEXT PRIMARY KEY NOT NULL,
			email_id TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			ip TEXT NOT NULL,
			FOREIGN KEY (email_id) REFERENCES emails(id)
				ON DELETE CASCADE
				ON UPDATE CASCADE
		)
	`)
	if err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) Track(id string,ip string) error {
	_, err := db.Exec(`
		INSERT INTO emails (id, created, ip, last_read)
		VALUES (?, ?, ?, NULL)
	`, id, time.Now(),ip)
	return err
}

func (db *DB) LogRead(id string, ip string) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()


	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM emails WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}



	var senderIP string
	row := tx.QueryRow("SELECT ip FROM emails WHERE id = ?", id)
	err = row.Scan(&senderIP)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no email found with id %v", id)
		}
		return fmt.Errorf("error querying email: %v", err)
	}
	
	if senderIP == ip {
		return nil
	}

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE emails 
		SET last_read = ?
		WHERE id = ?
	`, now, id)
	if err != nil {
		return err
	}

	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomString := hex.EncodeToString(randomBytes)
		
	logID:= fmt.Sprintf("%s_%s_%s", id, now.Format("20060102150405.000000000"), randomString)

	_, err = tx.Exec(`
		INSERT INTO logs (id, email_id, timestamp, ip)
		VALUES (?, ?, ?, ?)
	`, logID, id, now, ip)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) ShowLogs(id string) ([]LogEntry, error) {
	rows, err := db.Query(`
		SELECT id, email_id, timestamp, ip
		FROM logs
		WHERE email_id = ?
		ORDER BY timestamp DESC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var log LogEntry
		err := rows.Scan(&log.ID, &log.EmailID, &log.TimeStamp, &log.IP)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (db *DB) EmailExists(id string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM emails WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

