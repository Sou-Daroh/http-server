package server

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
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

// Database manages the SQLite connection for the honeypot.
type Database struct {
	db *sql.DB
	mu sync.Mutex
}

// NewDatabase initializes a new SQLite database at the given path.
func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dsn, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	d := &Database{db: db}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return d, nil
}

func (d *Database) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS attacks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ip TEXT NOT NULL,
		country TEXT NOT NULL,
		city TEXT NOT NULL,
		lat REAL NOT NULL,
		lon REAL NOT NULL,
		payload TEXT,
		target TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_attacks_timestamp ON attacks(timestamp);
	`
	_, err := d.db.Exec(query)
	return err
}

func (d *Database) LogAttack(event AttackEvent) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `
	INSERT INTO attacks (ip, country, city, lat, lon, payload, target, timestamp)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
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
	LIMIT ?
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
	LIMIT ?
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
