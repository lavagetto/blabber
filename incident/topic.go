package incident

import (
	"blabber/bot"
	"blabber/triggers"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
)

// RplTopic is the numeric TOPIC reply command (RFC 1459 section 6.2)
const RplTopic = "332"

func isTopicChange(bot *hbot.Bot, m *hbot.Message) bool {
	return m.Command == RplTopic
}

// Db interface

// GetTopic gets the topic for a channel from the db
func GetTopic(db *sql.DB, channel string) (*string, error) {
	var topic string
	err := db.QueryRow(
		"SELECT topic FROM topics WHERE channel = ?",
		channel).Scan(&topic)
	return &topic, err
}

func saveTopic(db *sql.DB, channel *string, topic *string) error {
	query := "UPDATE topics SET topic = ? WHERE channel = ?"
	_, err := GetTopic(db, *channel)

	if err != nil {
		query = "INSERT INTO topics (topic, channel) VALUES (?, ?)"
	}

	statement, err := db.Prepare(query)
	if err != nil {
		return err
	}
	_, err = statement.Exec(topic, channel)
	return err
}

func removeTopic(db *sql.DB, channel *string) error {
	statement, err := db.Prepare("DELETE FROM topics WHERE channel = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(channel)
	return err
}

// Handler functions
// StoreTopic stores the topic of a channel when it changes.
func StoreTopic(c *bot.Configuration, db *sql.DB) *triggers.EvHandler {
	return triggers.NewHandler(
		isTopicChange,
		func(bot *hbot.Bot, m *hbot.Message) bool {
			saveTopic(db, &m.Params[1], &m.Content)
			return false
		},
		"",
	)
}

// LogTopic logs a topic change
func LogTopic(c *bot.Configuration, db *sql.DB) *triggers.EvHandler {
	return triggers.NewHandler(
		isTopicChange,
		func(bot *hbot.Bot, m *hbot.Message) bool {
			bot.Logger.Info("Logging topic change!", "channel", m.Params[1], "topic", m.Content)
			return false
		},
		"",
	)
}

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

// StartIncident handles starting an incident
func StartIncident(c *bot.Configuration, db *sql.DB) *triggers.EvHandler {
	helpmsg := "!start_incident <severity> <component1>, <component2>..."
	cmdRegexp := regexp.MustCompile(`^\!start_incident\s+(\d+)\s+(.*)$`)
	return triggers.NewHandler(
		func(irc *hbot.Bot, m *hbot.Message) bool {
			return m.Command == "PRIVMSG" && cmdRegexp.MatchString(m.Content)
		},
		func(irc *hbot.Bot, m *hbot.Message) bool {
			matches := cmdRegexp.FindStringSubmatch(m.Content)
			severity, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				irc.Reply(m, "Couldn't parse severity")
				irc.Reply(m, helpmsg)
				return true
			}
			components := strings.Split(matches[2], ", ")
			inc, err := NewIncident(severity, components)
			if err != nil {
				irc.Reply(m, "Error creating incident:")
				irc.Reply(m, err.Error())
				return true
			}
			if err = inc.Save(db); err != nil {
				irc.Reply(m, "Error saving the incident:")
				irc.Reply(m, err.Error())
				return true
			}
			irc.Reply(m, fmt.Sprintf("Incident saved: %s", inc.Summary()))
			return true
		},
		helpmsg,
	)
}
