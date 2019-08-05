package incident

import (
	"blabber/bot"
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
	Document    RemoteDocument
}

// NewIncident creates an Incident object, and returns it
func NewIncident(severity int64, components []string, c *bot.Configuration) (*Incident, error) {
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
	// Try to create the remote document.
	document := NewGoogleDoc()
	if document != nil {
		date := time.Now().Format("2006-01-02")
		title := fmt.Sprintf("%s - %s", date, strings.Join(components, ", "))
		if err := document.NewFromTemplate(title, c); err != nil {
			log.Error("Error saving the document", "error", err)
		} else {
			inc.Document = document
		}
	}
	return &inc, nil
}

// Save allows to persist an incident to the database.
func (i *Incident) Save(db *sql.DB) error {
	var query string
	if i.ID == 0 {
		query = "INSERT INTO incidents (severity, components, started_at, updated_at, status, description, document_id) VALUES (?, ?, ?, ?, ?, ?, ?)"
	} else {
		query = "UPDATE incidents SET severity=?, components=?, updated_at=?, status=?, description=?, document_id=? WHERE id = ?"
	}
	statement, err := db.Prepare(query)
	if err != nil {
		return err
	}
	started := i.startedAt.Format(time.RFC3339)
	updated := i.updatedAt.Format(time.RFC3339)
	components := strings.Join(i.components, ", ")
	var documentID string
	if i.Document != nil {
		documentID = i.Document.Id()
	}
	if i.ID == 0 {
		var result sql.Result
		result, err = statement.Exec(i.severity, components, started, updated, i.Status, i.Description, documentID)
		i.ID, err = result.LastInsertId()
	} else {
		_, err = statement.Exec(i.severity, components, updated, i.Status, i.Description, documentID, i.ID)
	}
	return err
}

// UpdateDescription updates the current description with an update.
// If the description is not empty, a date header gets added to mark this as an update.
func (i *Incident) UpdateDescription(update string) {
	if i.Description != "" {
		humanTime := time.Now().Format("15:04 Jan 2 2006")
		i.Description += fmt.Sprintf("UPDATE %s\n", humanTime)
	}
	i.Description += fmt.Sprintf("%s\n-- \n", update)
}

// Summary formats a simple summary of an incident
func (i *Incident) Summary(extended bool) string {
	if i.Status == StatusClosed {
		return "Up"
	}
	var severity string
	if i.severity <= 3 {
		severity = "degraded"
	} else {
		severity = "down"
	}
	if extended && i.Document != nil {
		return fmt.Sprintf("%s %s (#%d - docs at %s)", strings.Join(i.components, ", "), severity, i.ID, i.Document.Url())
	}
	return fmt.Sprintf("%s %s (#%d)", strings.Join(i.components, ", "), severity, i.ID)
}

func incidentFromDbRows(rows *sql.Rows) (*Incident, error) {
	inc := Incident{}
	var components string
	var updated string
	var started string
	var docId string
	err := rows.Scan(&inc.ID, &inc.severity, &components, &started, &updated, &inc.Status, &inc.Description, &docId)
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
	doc := NewGoogleDoc()
	if doc != nil {
		if err := doc.GetFromId(docId); err != nil {
			log.Error("Could not find the document", "id", doc, "error", err)
		} else {
			inc.Document = doc
		}
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
	statement, err := db.Prepare("SELECT id, severity, components, started_at, updated_at, status, description, document_id from incidents WHERE id = ?")
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
	statement, err := db.Prepare("SELECT id, severity, components, started_at, updated_at, status, description, document_id from incidents WHERE status = ?")
	if err != nil {
		return nil, err
	}
	return getFromDb(statement, StatusOpen)
}

// IRC action functions
// The topic gets updated with the current incident status
func updateTopic(irc *hbot.Bot, db *sql.DB, channel string, c *bot.Configuration) error {
	incidents, err := GetOpenIncidents(db)
	if err != nil {
		return err
	}

	topic, err := NewTopic(db, channel).Get()
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
			// Only publish a full summary (including the gdoc address) if in a public channel.
			summaries = append(summaries, incident.Summary(c.IsPublicChannel(channel)))
		}
		status = strings.Join(summaries, " / ")
	}
	topicRegex := regexp.MustCompile("^(.*)\\| Status: ([^\\|]+)(.*)$")
	matches := topicRegex.FindStringSubmatch(topic)
	// No status line. Append it to the end of the topic
	if matches == nil {
		irc.Topic(channel, fmt.Sprintf("%s | Status: %s", topic, status))
	} else {
		if matches[3] != "" {
			// Add Padding
			status += " "
		}
		// If the topic needs an update, do it.
		// Note we don't save the topic to the db,
		// that will be handled by another handler.
		if status != matches[2] {
			newTopic := fmt.Sprintf("%s | Status: %s%s", matches[1], status, matches[3])
			irc.Topic(channel, newTopic)
		}
	}
	return err
}

func parseSeverity(severityString string, irc *hbot.Bot, m *hbot.Message) int64 {
	severity, err := strconv.ParseInt(severityString, 10, 64)
	if err != nil || severity > 5 || severity < 1 {
		irc.Reply(m, "Couldn't parse severity. It's supposed to be a number between 1 and 5.")
		return 0
	}
	return severity
}

func saveIncident(incident *Incident, db *sql.DB, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration) bool {
	err := incident.Save(db)
	if err != nil {
		irc.Reply(m, "Could not save the incident, please check the logs for errors")
		log.Error("Could not update incident", "error", err.Error(), "incident", incident.ID)
		return false
	}

	// Change the topic in all channels, report errors just to the issuer of the command in private.
	for _, channel := range c.Channels {
		// In general, we report failures in private to the issuer of the command. But if the channel is
		// the one where the command was issued, respond in public.
		// We therefore mangle m.To
		myMessage := *m
		if channel != m.To {
			myMessage.To = irc.Nick
		}
		if err = updateTopic(irc, db, channel, c); err != nil {
			irc.Reply(&myMessage, "Could not update the channel topic. Check my permissions please.")
			log.Error("Error updating the channel topic", "error", err.Error())
		}
	}
	return true
}

// startIncident handles starting an incident
func startIncident(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	splitRegex := regexp.MustCompile(",\\s*")
	severity := parseSeverity(args[0], irc, m)
	if severity == 0 {
		return true
	}
	components := splitRegex.Split(args[1], -1)
	inc, err := NewIncident(severity, components, c)
	if err != nil {
		irc.Reply(m, "Invalid parameters: ")
		irc.Reply(m, err.Error())
		return true
	}
	if saveIncident(inc, db, irc, m, c) {
		irc.Reply(m, fmt.Sprintf("Incident saved: %s", inc.Summary(true)))
	} else {
		irc.Reply(m, "Error creating the incident, check the logs for details.")
		log.Error("Error saving a new incident", "error", err.Error())
	}
	return true
}

func getIncidentFromIDParam(idString string, irc *hbot.Bot, m *hbot.Message, db *sql.DB) *Incident {
	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		irc.Reply(m, "Couldn't parse the incident id.")
		return nil
	}

	inc, err := GetByID(db, id)
	if inc == nil {
		irc.Reply(m, "Incident not found.")
		if err != nil {
			log.Error("Could not get incident by id", "error", err.Error(), "id", id)
		}
		return nil
	}
	return inc
}

func stopIncident(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	inc := getIncidentFromIDParam(args[0], irc, m, db)
	if inc == nil {
		return true
	}
	if inc.Status == StatusClosed {
		irc.Reply(m, "This incident is already closed.")
		return true
	}
	inc.Status = StatusClosed
	if saveIncident(inc, db, irc, m, c) {
		irc.Reply(m, fmt.Sprintf("Incident closed: %d", inc.ID))
	} else {
		irc.Reply(m, "Could not close the incident, see logs for details.")
	}
	return true
}

func updateIncident(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	inc := getIncidentFromIDParam(args[0], irc, m, db)
	if inc == nil {
		return true
	}
	if inc.Status == StatusClosed {
		irc.Reply(m, fmt.Sprintf("Incident %d was closed, reopening it", inc.ID))
		inc.Status = StatusOpen
	}
	if args[1] == "severity" {
		severity := parseSeverity(args[2], irc, m)
		if severity == 0 {
			return true
		}
		inc.severity = severity
	} else {
		inc.UpdateDescription(args[2])
	}
	if saveIncident(inc, db, irc, m, c) {
		irc.Reply(m, fmt.Sprintf("Incident %d updated.", inc.ID))
	} else {
		irc.Reply(m, "Update failed. Please see the logs for details.")
	}
	return true
}

func listOpenIncidents(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	incidents, err := GetOpenIncidents(db)
	if err != nil {
		irc.Reply(m, "Could not retrieve the list of open incidents. Please check the logs")
		log.Error("Could not retrieve the list of open incidents from the database", "error", err)
		return false
	} else if len(incidents) == 0 {
		irc.Reply(m, "No open incidents! 👍")

	} else {
		irc.Reply(m, "Open incidents:")
		for _, incident := range incidents {
			line := fmt.Sprintf("  * %s", incident.Summary(false))
			irc.Reply(m, line)
		}
	}
	return false
}

func formatIncident(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	inc := getIncidentFromIDParam(args[0], irc, m, db)
	if inc == nil {
		return true
	}
	if inc.Status == StatusClosed {
		irc.Reply(m, fmt.Sprintf("Incident %d is closed. Last update was at %v", inc.ID, inc.updatedAt))
	} else {
		irc.Reply(m, "-- ")
		irc.Reply(m, "== "+inc.Summary(false))
		irc.Reply(m, "Description:")
		for _, line := range strings.Split(inc.Description, "\n") {
			irc.Reply(m, line)
		}
		if inc.Document != nil && inc.Document.Url() != "<not available>" {
			irc.Reply(m, " \n")
			irc.Reply(m, "Google Doc: "+inc.Document.Url())
		}
	}
	return false
}
