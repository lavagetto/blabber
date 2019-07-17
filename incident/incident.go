package incident

import (
	"blabber/bot"
	"blabber/triggers"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
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
		query = "UPDATE incidents SET severity=?, components=?, updated_at=?, status=?, description=? WHERE id = ?"
	}
	statement, err := db.Prepare(query)
	if err != nil {
		return err
	}
	started := i.startedAt.Format(time.RFC3339)
	updated := i.updatedAt.Format(time.RFC3339)
	components := strings.Join(i.components, ", ")
	if i.ID == 0 {
		var result sql.Result
		result, err = statement.Exec(i.severity, components, started, updated, i.Status, i.Description)
		i.ID, err = result.LastInsertId()
	} else {
		_, err = statement.Exec(i.severity, components, updated, i.Status, i.Description, i.ID)
	}
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

// IRC action functions
// The topic gets updated with the current incident status
func updateTopic(irc *hbot.Bot, db *sql.DB, channel string) error {
	incidents, err := GetOpenIncidents(db)
	if err != nil {
		return err
	}
	topic, err := GetTopic(db, channel)
	if err != nil {
		return err
	}
	var status string
	// Status up
	if incidents == nil {
		status = "Up"
	} else {
		var summaries []string
		for _, incident := range incidents {
			summaries = append(summaries, incident.Summary())
		}
		status = strings.Join(summaries, " / ")
	}
	topicRegex := regexp.MustCompile("^(.*)\\| Status: [^\\|]+(.*)$")
	matches := topicRegex.FindStringSubmatch(*topic)
	// No status line. Append it to the end of the topic
	if matches == nil {
		irc.Topic(channel, fmt.Sprintf("%s | Status: %s", *topic, status))
	} else {
		irc.Topic(channel, fmt.Sprintf("%s | Status: %s%s", matches[1], status, matches[2]))
	}
	return err
}

// startIncident handles starting an incident
func startIncident(args []string, c *bot.Configuration, db *sql.DB) triggers.TriggerFunc {
	splitRegex := regexp.MustCompile(",\\s*")
	return func(irc *hbot.Bot, m *hbot.Message) bool {
		severity, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			irc.Reply(m, "Couldn't parse severity. It's supposed to be a number between 1 and 5.")
			return true
		}
		components := splitRegex.Split(args[1], -1)
		inc, err := NewIncident(severity, components)
		if err != nil {
			irc.Reply(m, "Invalid parameters: ")
			irc.Reply(m, err.Error())
			return true
		}
		if err = inc.Save(db); err != nil {
			irc.Reply(m, "Error saving the incident.")
			log.Error("Error saving a new incident", "error", err.Error())
			return true
		}
		if err = updateTopic(irc, db, m.To); err != nil {
			irc.Reply(m, "Could not update the channel topic. Check my permissions please.")
			log.Error("Error updating the topic", "error", err.Error())
		}
		irc.Reply(m, fmt.Sprintf("Incident saved: %s", inc.Summary()))
		return true
	}
}

func stopIncident(args []string, c *bot.Configuration, db *sql.DB) triggers.TriggerFunc {
	return func(irc *hbot.Bot, m *hbot.Message) bool {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			irc.Reply(m, "Couldn't parse the incident id.")
			return true
		}

		inc, err := GetByID(db, id)
		if inc == nil {
			irc.Reply(m, "Incident not found.")
			if err != nil {
				log.Error("Could not get incident by id", "error", err.Error(), "id", id)
			}
			return true
		}
		inc.Status = StatusClosed
		err = inc.Save(db)
		if err != nil {
			irc.Reply(m, "Could not close the incident, see logs for details.")
			log.Error("Could not update incident", "error", err.Error(), "incident", inc.Summary())
		}
		if err = updateTopic(irc, db, m.To); err != nil {
			irc.Reply(m, "Could not update the channel topic. Check my permissions please.")
			log.Error("Error updating the topic", "error", err.Error())
		}
		irc.Reply(m, fmt.Sprintf("Incident closed: %d", inc.ID))
		return true
	}
}
