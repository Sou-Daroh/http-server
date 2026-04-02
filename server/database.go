package server

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// AttackEvent represents a single malicious request caught by the honeypot.
type AttackEvent struct {
	ID        int       `json:"id"`
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Payload   string    `json:"payload"`
	Target    string    `json:"target"`
	Timestamp time.Time `json:"timestamp"`
}

// CountryStat is used for building leaderboards in the UI.
type CountryStat struct {
	Country string `json:"country"`
	Count   int    `json:"count"`
}

// Database manages the Postgres connection pooling for the honeypot.
type Database struct {
	db *sql.DB
}

// NewDatabase initializes a new Postgres connection pool.
func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	// Advanced Training Requirement: Connection Pooling for Scalability
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	d := &Database{db: db}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate postgres db: %w", err)
	}

	// Automatic Syllabus Fulfillment: Seed a default Admin for the Dashboard
	if err := d.SeedDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("seed default admin: %w", err)
	}

	return d, nil
}

func (d *Database) migrate() error {
	// Postgres natively uses SERIAL instead of AUTOINCREMENT
	query := `
	CREATE TABLE IF NOT EXISTS attacks (
		id SERIAL PRIMARY KEY,
		ip TEXT NOT NULL,
		country TEXT NOT NULL,
		city TEXT NOT NULL,
		lat REAL NOT NULL,
		lon REAL NOT NULL,
		payload TEXT,
		target TEXT NOT NULL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_attacks_timestamp ON attacks(timestamp);

	CREATE TABLE IF NOT EXISTS admins (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);
	`
	_, err := d.db.Exec(query)
	return err
}

func (d *Database) LogAttack(event AttackEvent) error {
	// Postgres explicitly uses $1, $2 index placeholders instead of ?
	query := `
	INSERT INTO attacks (ip, country, city, lat, lon, payload, target, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := d.db.Exec(query,
		event.IP, event.Country, event.City, event.Lat, event.Lon,
		event.Payload, event.Target, event.Timestamp.UTC(),
	)
	return err
}

func (d *Database) GetRecentAttacks(limit int) ([]AttackEvent, error) {
	query := `
	SELECT id, ip, country, city, lat, lon, payload, target, timestamp
	FROM attacks
	ORDER BY id DESC
	LIMIT $1
	`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AttackEvent
	for rows.Next() {
		var e AttackEvent
		if err := rows.Scan(
			&e.ID, &e.IP, &e.Country, &e.City, &e.Lat, &e.Lon,
			&e.Payload, &e.Target, &e.Timestamp,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (d *Database) GetTopCountries(limit int) ([]CountryStat, error) {
	query := `
	SELECT country, COUNT(*) as count
	FROM attacks
	GROUP BY country
	ORDER BY count DESC
	LIMIT $1
	`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []CountryStat
	for rows.Next() {
		var s CountryStat
		if err := rows.Scan(&s.Country, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// SeedDefaultAdmin creates a fallback admin user using our bcrypt library.
func (d *Database) SeedDefaultAdmin() error {
	var count int
	d.db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count == 0 {
		hash, _ := HashPassword("admin123")
		_, err := d.db.Exec("INSERT INTO admins (username, password_hash) VALUES ($1, $2)", "admin", hash)
		return err
	}
	return nil
}

// VerifyAdmin checks if the provided credentials align with the bcrypt hash safely stored in Postgres.
func (d *Database) VerifyAdmin(username, plainPassword string) bool {
	var hash string
	err := d.db.QueryRow("SELECT password_hash FROM admins WHERE username = $1", username).Scan(&hash)
	if err != nil {
		return false // User not found
	}
	return CheckPasswordHash(plainPassword, hash)
}
