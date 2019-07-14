package incident

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PublicComponents are a list of things that can break
// from the public POV
var PublicComponents = []string{
	"Website",
	"Mobile apps",
	"Action API",
	"REST api",
	"Multimedia",
	"Thumbnails",
	"Other",
}

// Status of the incident.
const (
	StatusOpen int64 = iota
	StatusClosed
)

// The Incident struct is used to contain data about an incident.
// Such data can be used to perform various actions like updating
// an IRC channel topic.
type Incident struct {
	severity    int64
	startedAt   time.Time
	updatedAt   time.Time
	components  []string
	Description string
	Status      int64
	ID          int64
}

// NewIncident creates an Incident object, and returns it
func NewIncident(severity int64, components []string) (*Incident, error) {
	if severity > 5 || severity < 1 {
		return nil, errors.New("Severity must be between 1 and 5")
	}
	var normalized []string
	for _, component := range components {
		c := strings.ToLower(component)
		var found string
		for _, comp := range PublicComponents {
			if c == strings.ToLower(comp) {
				found = comp
			}
		}
		if found != "" {
			normalized = append(normalized, found)
		} else {
			return nil, fmt.Errorf("Unknown component '%s'", component)
		}
	}
	inc := Incident{
		severity:   severity,
		components: normalized,
		startedAt:  time.Now(),
		updatedAt:  time.Now(),
		Status:     StatusOpen,
	}
	return &inc, nil
}

// Save allows to persist an incident to the database.
func (i *Incident) Save(db *sql.DB) error {
	var query string
	if i.ID == 0 {
		query = "INSERT INTO incidents (severity, components, started_at, updated_at, status, description) VALUES (?, ?, ?, ?, ?, ?)"
	} else {
		query = "UPDATE incidents SET severity=?, components=?, started_at=?, updated_at=?, status=?, description=? WHERE id = ?"
	}
	statement, err := db.Prepare(query)
	if err != nil {
		return err
	}
	started := i.startedAt.Format(time.RFC3339)
	updated := i.updatedAt.Format(time.RFC3339)
	components := strings.Join(i.components, ", ")
	var result sql.Result
	if i.ID == 0 {
		result, err = statement.Exec(i.severity, components, started, updated, i.Status, i.Description)
	} else {
		result, err = statement.Exec(i.severity, components, started, updated, i.Status, i.Description, i.ID)
	}
	i.ID, err = result.LastInsertId()
	return err
}

// Summary formats a simple summary of an incident
func (i *Incident) Summary() string {
	if i.Status == StatusClosed {
		return "Up"
	}
	var severity string
	if i.severity <= 3 {
		severity = "degraded"
	} else {
		severity = "down"
	}
	return fmt.Sprintf("%s %s (#%d)", strings.Join(i.components, ", "), severity, i.ID)
}

func incidentFromDbRows(rows *sql.Rows) (*Incident, error) {
	inc := Incident{}
	var components string
	var updated string
	var started string
	err := rows.Scan(&inc.ID, &inc.severity, &components, &started, &updated, &inc.Status, &inc.Description)
	if err != nil {
		return nil, err
	}
	inc.components = strings.Split(components, ", ")
	inc.startedAt, err = time.Parse(time.RFC3339, started)
	if err != nil {
		return nil, err
	}
	inc.updatedAt, err = time.Parse(time.RFC3339, updated)
	if err != nil {
		return nil, err
	}
	return &inc, err
}

func getFromDb(statement *sql.Stmt, arg int64) ([]*Incident, error) {
	rows, err := statement.Query(arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var incidents []*Incident
	for rows.Next() {
		inc, err := incidentFromDbRows(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}
	return incidents, err
}

// GetByID fetches one incident from the database
func GetByID(db *sql.DB, id int64) (*Incident, error) {
	statement, err := db.Prepare("SELECT id, severity, components, started_at, updated_at, status, description from incidents WHERE id = ?")
	if err != nil {
		return nil, err
	}
	incidents, err := getFromDb(statement, id)
	if err != nil || incidents == nil {
		return nil, err
	}
	return incidents[0], nil

}

// GetOpenIncidents returns the currently open incidents
func GetOpenIncidents(db *sql.DB) ([]*Incident, error) {
	statement, err := db.Prepare("SELECT id, severity, components, started_at, updated_at, status, description from incidents WHERE status = ?")
	if err != nil {
		return nil, err
	}
	return getFromDb(statement, StatusOpen)
}
