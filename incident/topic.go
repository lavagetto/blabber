package incident

import (
	"blabber/bot"
	"database/sql"

	hbot "github.com/whyrusleeping/hellabot"
)

// RplTopic is the numeric TOPIC reply command (RFC 1459 section 6.2)
const RplTopic = "332"

func isTopicChange(bot *hbot.Bot, m *hbot.Message) bool {
	return m.Command == RplTopic || m.Command == "TOPIC"
}

// Topic structure
type Topic struct {
	Channel string
	db      *sql.DB
}

// NewTopic returns a new topic.
func NewTopic(db *sql.DB, channel string) *Topic {
	return &Topic{Channel: channel, db: db}
}

// Get returns the topic, fetching it from the database.
// It also returns any error found while fetching the data.
func (t *Topic) Get() (string, error) {
	var topic string
	err := t.db.QueryRow(
		"SELECT topic FROM topics WHERE channel = ?",
		t.Channel).Scan(&topic)
	return topic, err
}

// Save persists the topic to the database.
func (t *Topic) Save(topic *string) error {
	var query string
	if _, err := t.Get(); err != nil {
		query = "INSERT INTO topics (topic, channel) VALUES (?, ?)"
	} else {
		query = "UPDATE topics SET topic = ? WHERE channel = ?"
	}

	statement, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	_, err = statement.Exec(&topic, &t.Channel)
	return err
}

// Clean removes the topic from the database
func (t *Topic) Clean() error {
	statement, err := t.db.Prepare("DELETE FROM topics WHERE channel = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(t.Channel)
	return err
}

// Handler functions

// StoreTopic stores the topic of a channel when it changes.
func StoreTopic(irc *hbot.Bot, m *hbot.Message, db *sql.DB, c *bot.Configuration) bool {
	if isTopicChange(irc, m) {
		var channel string
		// The channel is stored in Params[1] when joining a channel
		// and in m.To when we did a topic change.
		if m.Command == RplTopic {
			channel = m.Params[1]
		} else {
			channel = m.To
		}
		t := NewTopic(db, channel)
		// This can block a bit when we're joining the channels.
		go func() {
			if err := t.Save(&m.Content); err != nil {
				irc.Logger.Error("Could not save the topic to the database", "channel", t.Channel, "error", err)
			} else {
				irc.Logger.Info("Logging topic change!", "channel", channel, "topic", m.Content)
			}
		}()
	}
	// we don't stop processing in any case.
	return false
}
