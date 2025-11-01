package models

import "time"

// Secret represents a secret from Akeyless with its metadata
type Secret struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// SecretItem represents a secret path from Akeyless listing
type SecretItem struct {
	ItemName string
	ItemType string
}
