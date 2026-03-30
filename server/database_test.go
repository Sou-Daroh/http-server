package server

import (
	"testing"
	"time"
)

func TestDatabase_LogAndRetrieveAttacks(t *testing.T) {
	// Use an in-memory database exclusively for testing
	// This prevents creating lingering .db files during `go test`
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize memory database: %v", err)
	}

	now := time.Now().UTC()
	event := AttackEvent{
		IP:        "192.168.1.1",
		Country:   "United States",
		City:      "New York",
		Lat:       40.7128,
		Lon:       -74.0060,
		Payload:   "GET /.env HTTP/1.1",
		Target:    "/.env",
		Timestamp: now,
	}

	err = db.LogAttack(event)
	if err != nil {
		t.Fatalf("Failed to log attack: %v", err)
	}

	recent, err := db.GetRecentAttacks(10)
	if err != nil {
		t.Fatalf("Failed to get recent attacks: %v", err)
	}

	if len(recent) != 1 {
		t.Fatalf("Expected 1 recent attack, got %d", len(recent))
	}

	if recent[0].IP != "192.168.1.1" || recent[0].Target != "/.env" {
		t.Errorf("Retrieved event data mismatch: %+v", recent[0])
	}
}

func TestDatabase_GetTopCountries(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize memory database: %v", err)
	}

	// Insert 3 attacks from US, 1 from CA
	events := []AttackEvent{
		{Country: "United States", Target: "/wp-login.php", Timestamp: time.Now()},
		{Country: "United States", Target: "/.git/config", Timestamp: time.Now()},
		{Country: "Canada", Target: "/wp-admin", Timestamp: time.Now()},
		{Country: "United States", Target: "/.env", Timestamp: time.Now()},
	}

	for _, e := range events {
		if err := db.LogAttack(e); err != nil {
			t.Fatalf("Failed to log attack: %v", err)
		}
	}

	stats, err := db.GetTopCountries(5)
	if err != nil {
		t.Fatalf("Failed to get top countries: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("Expected exactly 2 country stats (US and CA), but got %d", len(stats))
	}

	// The query uses ORDER BY count DESC, so US should always be [0]
	if stats[0].Country != "United States" || stats[0].Count != 3 {
		t.Errorf("Expected most targeted country to be 'United States' with 3 hits, got %s with %d hits", stats[0].Country, stats[0].Count)
	}

	if stats[1].Country != "Canada" || stats[1].Count != 1 {
		t.Errorf("Expected second country to be 'Canada' with 1 hit, got %s with %d hits", stats[1].Country, stats[1].Count)
	}
}
