package server

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// Location holds the geographical layout of an IP.
type Location struct {
	Country string  `json:"country"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// GeoIP encapsulates lookups so we don't spam APIs.
type GeoIP struct {
	db *geoip2.Reader
}

// NewGeoIP attempts to open a MaxMind .mmdb database. 
// If it fails (file not found), it gracefully degrades to a free web API.
func NewGeoIP(dbPath string) (*GeoIP, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		// Log silently and continue without MaxMind, we'll use fallback.
		return &GeoIP{db: nil}, nil
	}
	return &GeoIP{db: db}, nil
}

// Lookup converts an IP into Coordinates.
func (g *GeoIP) Lookup(ipStr string) Location {
	if ipStr == "127.0.0.1" || ipStr == "::1" || ipStr == "" || ipStr == "localhost" {
		// Local IP for local testing
		return Location{Country: "Localhost", City: "Local", Lat: 0, Lon: 0}
	}

	ip := net.ParseIP(ipStr)
	if ip != nil && g.db != nil {
		// Try using MaxMind DB securely.
		record, err := g.db.City(ip)
		if err == nil {
			return Location{
				Country: record.Country.Names["en"],
				City:    record.City.Names["en"],
				Lat:     record.Location.Latitude,
				Lon:     record.Location.Longitude,
			}
		}
	}

	// Graceful fallback to Free API (good for Portfolio execution if no MMDB is provided).
	return fallbackLookup(ipStr)
}

func fallbackLookup(ip string) Location {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		return Location{Country: "Unknown", City: "Unknown"}
	}
	defer resp.Body.Close()

	var result struct {
		Status  string  `json:"status"`
		Country string  `json:"country"`
		City    string  `json:"city"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Location{Country: "Unknown", City: "Unknown"}
	}

	if result.Status == "success" {
		return Location{
			Country: result.Country,
			City:    result.City,
			Lat:     result.Lat,
			Lon:     result.Lon,
		}
	}

	return Location{Country: "Unknown", City: "Unknown"}
}
