package hbapi

import "time"

// User represents a Honeybadger user
type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Team represents a Honeybadger team
type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Site represents a monitoring site
type Site struct {
	ID            string     `json:"id"`
	Active        bool       `json:"active"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	Name          string     `json:"name"`
	State         string     `json:"state"`
	URL           string     `json:"url"`
}

// Project represents a Honeybadger project
type Project struct {
	ID                     int        `json:"id"`
	Name                   string     `json:"name"`
	Active                 bool       `json:"active"`
	CreatedAt              time.Time  `json:"created_at"`
	EarliestNoticeAt       *time.Time `json:"earliest_notice_at"`
	LastNoticeAt           *time.Time `json:"last_notice_at"`
	Environments           []string   `json:"environments"`
	FaultCount             int        `json:"fault_count"`
	UnresolvedFaultCount   int        `json:"unresolved_fault_count"`
	Token                  string     `json:"token"`
	Owner                  User       `json:"owner"`
	Sites                  []Site     `json:"sites"`
	Teams                  []Team     `json:"teams"`
	Users                  []User     `json:"users"`
}

// ProjectsResponse represents the API response for listing projects
type ProjectsResponse struct {
	Results []Project `json:"results"`
}