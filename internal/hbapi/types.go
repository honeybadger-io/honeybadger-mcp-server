package hbapi

import "time"

// User represents a Honeybadger user
type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Account represents a Honeybadger account
type Account struct {
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
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Active               bool       `json:"active"`
	CreatedAt            time.Time  `json:"created_at"`
	EarliestNoticeAt     *time.Time `json:"earliest_notice_at"`
	LastNoticeAt         *time.Time `json:"last_notice_at"`
	Environments         []string   `json:"environments"`
	FaultCount           int        `json:"fault_count"`
	UnresolvedFaultCount int        `json:"unresolved_fault_count"`
	Token                string     `json:"token"`
	Owner                Account    `json:"owner"`
	Sites                []Site     `json:"sites"`
	Teams                []Team     `json:"teams"`
	Users                []User     `json:"users"`
}

// Fault represents a Honeybadger fault
type Fault struct {
	ID                  int        `json:"id"`
	Action              string     `json:"action"`
	Assignee            *User      `json:"assignee"`
	CommentsCount       int        `json:"comments_count"`
	Component           string     `json:"component"`
	CreatedAt           time.Time  `json:"created_at"`
	Environment         string     `json:"environment"`
	Ignored             bool       `json:"ignored"`
	Klass               string     `json:"klass"`
	LastNoticeAt        *time.Time `json:"last_notice_at"`
	Message             string     `json:"message"`
	NoticesCount        int        `json:"notices_count"`
	NoticesCountInRange *int       `json:"notices_count_in_range,omitempty"` // Added when fault list search query affects notice count
	ProjectID           int        `json:"project_id"`
	Resolved            bool       `json:"resolved"`
	Tags                []string   `json:"tags"`
	URL                 string     `json:"url"`
}

// NoticeEnvironment represents the environment information for a notice
type NoticeEnvironment struct {
	EnvironmentName string                 `json:"environment_name"`
	Hostname        string                 `json:"hostname"`
	ProjectRoot     interface{}            `json:"project_root"` // Can be string or object
	Revision        *string                `json:"revision"`
	Stats           map[string]interface{} `json:"stats"`
	Time            string                 `json:"time"`
	PID             int                    `json:"pid"`
}

// NoticeRequest represents the HTTP request information for a notice
type NoticeRequest struct {
	Action    *string                `json:"action"`
	Component *string                `json:"component"`
	Context   map[string]interface{} `json:"context"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	URL       *string                `json:"url"`
	User      map[string]interface{} `json:"user"`
}

// BacktraceEntry represents a single entry in the error backtrace
type BacktraceEntry struct {
	Number  string                 `json:"number"`
	File    string                 `json:"file"`
	Method  string                 `json:"method"`
	Source  map[string]interface{} `json:"source,omitempty"`
	Context string                 `json:"context,omitempty"`
}

// Notice represents a Honeybadger notice (individual error occurrence)
type Notice struct {
	ID               string                 `json:"id"`
	CreatedAt        time.Time              `json:"created_at"`
	Environment      NoticeEnvironment      `json:"environment"`
	EnvironmentName  string                 `json:"environment_name"`
	Cookies          map[string]interface{} `json:"cookies"`
	FaultID          int                    `json:"fault_id"`
	URL              string                 `json:"url"`
	Message          string                 `json:"message"`
	WebEnvironment   map[string]interface{} `json:"web_environment"`
	Request          NoticeRequest          `json:"request"`
	Backtrace        []BacktraceEntry       `json:"backtrace"`
	ApplicationTrace []BacktraceEntry       `json:"application_trace"`
	Deploy           interface{}            `json:"deploy"` // Can be null or object
}

// PaginationLinks represents the pagination links in API responses
type PaginationLinks struct {
	Next string `json:"next"`
	Prev string `json:"prev"`
	Self string `json:"self"`
}

// ListResponse represents a generic list response with results and pagination links
type ListResponse[T any] struct {
	Results []T             `json:"results"`
	Links   PaginationLinks `json:"links"`
}
