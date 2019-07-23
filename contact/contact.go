package contact

import (
	"blabber/bot"
	"blabber/triggers"
	"database/sql"
	"fmt"

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

func addContactAction(args []string, bot *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	contact := Contact{name: args[0], phone: args[1], email: args[2]}
	err := contact.Save(db)
	if err == nil {
		bot.Reply(m, "Contact added successfully.")
	} else {
		bot.Reply(m, "Trouble saving the contact, please try again later.")
		log.Error(err.Error())
	}
	return true
}

func removeContactAction(args []string, bot *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	contact, err := GetContact(db, args[0])
	if err != nil {
		bot.Reply(m, "Couldn't find the contact you searched for")
		log.Error(err.Error())
		return true
	}
	err = contact.Remove(db)
	if err != nil {
		bot.Reply(m, "Couldn't remove contact, check logs for the error.")
		log.Error("Error removing contact:", "error", err.Error(), "contact", contact.PrettyPrint())
	} else {
		bot.Reply(m, "Contact successfully removed.")
	}
	return true
}

func getContactAction(args []string, bot *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	contact, err := GetContact(db, args[0])
	if err != nil {
		bot.Reply(m, "Couldn't find the contact you searched for")
		log.Error(err.Error())
	} else if contact.phone == "" {
		bot.Reply(m, "No phone data for the contact")
	} else {
		bot.Reply(m, contact.PrettyPrint())
	}
	return true
}

// Commands
var IrcCommands = []*triggers.Command{
	triggers.NewCommand(
		"contact_add",
		"(?P<name>\\w+)\\s+(?P<intl_phone>\\+\\d{5,15})\\s+(P?<email>\\S+)$",
		"Add a contact (privmsg only)",
		false,
		true,
		addContactAction,
	),
	triggers.NewCommand(
		"contact_get",
		"(?P<name>\\w+)",
		"Gets information about a contact (privmsg only)",
		false,
		true,
		getContactAction,
	),
	triggers.NewCommand(
		"contact_remove",
		"(?P<name>\\w+)",
		"Gets information about a contact (privmsg only)",
		false,
		true,
		removeContactAction,
	),
}
