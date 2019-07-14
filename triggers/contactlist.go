package triggers

import (
	"blabber/bot"
	"database/sql"
	"fmt"
	"regexp"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Contact struct {
	name  string
	phone string
	email string
}

// Get a contact from the db
func GetContact(db *sql.DB, name string) (*Contact, error) {
	var c Contact
	err := db.QueryRow(
		"SELECT name, phone, email FROM contacts WHERE name = ?",
		name).Scan(&c.name, &c.phone, &c.email)
	return &c, err
}

func (self *Contact) insert(db *sql.DB) error {
	statement, err := db.Prepare("INSERT INTO contacts VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = statement.Exec(self.name, self.phone, self.email)
	return err
}

func (self *Contact) update(db *sql.DB) error {
	statement, err := db.Prepare("UPDATE contacts SET phone = ?, email = ? WHERE name = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(self.phone, self.email, self.name)
	return err
}

func (self *Contact) Save(db *sql.DB) error {
	_, err := GetContact(db, self.name)
	if err != nil {
		return self.insert(db)
	} else {
		return self.update(db)
	}
}

func (self *Contact) PrettyPrint() string {
	return fmt.Sprintf("%s: %s (%s)", self.name, self.phone, self.email)
}

func (self *Contact) Remove(db *sql.DB) error {
	statement, err := db.Prepare("DELETE FROM contacts WHERE name = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(self.name)
	return err
}

// closures
func AddContactHandler(c *bot.Configuration, db *sql.DB) *EvHandler {
	// TODO: use some proper regexp for the email
	base := regexp.MustCompile("^\\!add_contact\\s+")
	command := regexp.MustCompile("^\\!add_contact\\s+(\\w+)\\s+(\\+\\d{5,15})\\s+(.*)$")

	return &EvHandler{
		func(bot *hbot.Bot, m *hbot.Message) bool {
			return m.Command == "PRIVMSG" && base.MatchString(m.Content)
		},
		func(bot *hbot.Bot, m *hbot.Message) bool {
			matches := command.FindStringSubmatch(m.Content)
			if matches == nil {
				bot.Reply(m, "Could not recognize input!")
				bot.Reply(m, "Accepted format: ")
				bot.Reply(m, "!add_contact <name> <phone number, intl> <email>")
				return true
			}
			c := Contact{matches[1], matches[2], matches[3]}
			err := c.Save(db)
			if err == nil {
				bot.Reply(m, "Contact added successfully.")
			} else {
				bot.Reply(m, "Trouble saving the contact, please try again later.")
				log.Error(err.Error())
			}
			return true
		},
		"Add a contact (privmsg only). Format: !add_contact <name> <int-phone-number> <email>",
	}
}

func GetContactHandler(c *bot.Configuration, db *sql.DB) *EvHandler {
	base := regexp.MustCompile("^\\!get_contact\\s+")
	command := regexp.MustCompile("^\\!get_contact\\s+(\\w+)")

	return &EvHandler{
		func(bot *hbot.Bot, m *hbot.Message) bool {
			return m.Command == "PRIVMSG" && base.MatchString(m.Content)
		},
		func(bot *hbot.Bot, m *hbot.Message) bool {
			matches := command.FindStringSubmatch(m.Content)
			c, err := GetContact(db, matches[1])
			if err != nil {
				bot.Reply(m, "Couldn't find the contact you searched for")
				log.Error(err.Error())
				return true
			} else if c.phone == "" {
				bot.Reply(m, "No phone data for the contact")
				return true
			} else {
				bot.Reply(m, c.PrettyPrint())
			}
			return true
		},
		"Gets a contact information (privmsg only). Format: !get_contact <name>",
	}
}
