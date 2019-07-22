package incident

import (
	"blabber/bot"
	"blabber/triggers"
	"database/sql"

	hbot "github.com/whyrusleeping/hellabot"
)

// RplTopic is the numeric TOPIC reply command (RFC 1459 section 6.2)
const RplTopic = "332"

func isTopicChange(bot *hbot.Bot, m *hbot.Message) bool {
	return m.Command == RplTopic || m.Command == "TOPIC"
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
			var channel string
			if m.Command == RplTopic {
				channel = m.Params[1]
			} else {
				channel = m.To
			}
			saveTopic(db, &channel, &m.Content)
			bot.Logger.Info("Logging topic change!", "channel", channel, "topic", m.Content)

			return false
		},
		"",
	)
}
