package triggers

import (
	"blabber/bot"
	"database/sql"
	"fmt"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

/*
	ACLs management.
*/
type commandACL struct {
	nicks    map[string]bool
	channels map[string]bool
}

func (acl *commandACL) IsAllowed(m *hbot.Message) bool {
	// First check the nickname
	if _, ok := acl.nicks[m.Name]; ok {
		return true
	}
	// Then the channel
	if _, ok := acl.channels[m.To]; ok {
		return true
	}
	return false
}

// CRD operations on ACLs
// GetACL returns a full commandACL that can be used in a command.
func GetACL(ID string, db *sql.DB, conf *bot.Configuration) (*commandACL, error) {
	var c commandACL
	// Admins are always allowed to perform any action.
	c.nicks = make(map[string]bool, 0)
	for _, admin := range conf.Admins {
		c.nicks[admin] = true
	}
	c.channels = make(map[string]bool, 0)
	statement, err := db.Prepare("SELECT identifier FROM acls WHERE command = ?")
	if err != nil {
		return &c, err
	}
	rows, err := statement.Query(ID)
	if err != nil {
		return &c, err
	}
	defer rows.Close()
	for rows.Next() {
		var identifier string
		err := rows.Scan(&identifier)
		if err != nil {
			return &c, err
		}
		if strings.HasPrefix(identifier, "#") {
			c.channels[identifier] = true
		} else {
			c.nicks[identifier] = true
		}
	}
	return &c, err
}

func ExistsACL(command string, identifier string, db *sql.DB) bool {
	statement, err := db.Prepare("SELECT count(1)  FROM acls WHERE command = ? AND identifier = ?")
	if err != nil {
		return false
	}
	var isPresent int
	err = statement.QueryRow(command, identifier).Scan(&isPresent)
	return err == nil && isPresent == 1
}

func SaveACL(command string, identifier string, db *sql.DB) error {
	statement, err := db.Prepare("INSERT INTO acls VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("Could not prepare the statement to add ACLs: %s", err)
	}
	_, err = statement.Exec(command, identifier)
	return err
}

func DeleteACL(command string, identifier string, db *sql.DB) error {
	statement, err := db.Prepare("DELETE FROM acls WHERE command = ? AND identifier = ?")
	if err != nil {
		return fmt.Errorf("Could not prepare the statement to remove the  ACL: %s", err)
	}
	_, err = statement.Exec(command, identifier)
	return err
}

// IRC actions
func addACL(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	if len(args) != 2 {
		irc.Reply(m, "Somehow we got the wrong number of arguments.")
		return false
	}
	command := args[0]
	identifier := args[1]
	// First let's check if the ACL is already present.
	if ExistsACL(command, identifier, db) {
		irc.Reply(m, "This ACL is already present.")
		return false
	} else {
		err := SaveACL(command, identifier, db)
		if err != nil {
			log.Error("Problem saving ACLs:", "error", err.Error())
			irc.Reply(m, "Couldn't save the new ACL.")
			return false
		}
	}
	irc.Reply(m, "The ACL was saved.")
	return true
}

// Special command to remove an acl rule
func removeAcl(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	if len(args) != 2 {
		irc.Reply(m, "Somehow we got the wrong number of arguments.")
		return false
	}
	command := args[0]
	identifier := args[1]
	// First let's check if the ACL is already present.
	if !ExistsACL(command, identifier, db) {
		irc.Reply(m, "This ACL is not present.")
		return false
	} else {
		err := DeleteACL(command, identifier, db)
		if err != nil {
			log.Error("Problem removing ACLs:", "error", err.Error())
			irc.Reply(m, "Couldn't remove the ACL.")
			return false
		}
	}
	irc.Reply(m, "The ACL was succesfully removed.")
	return true
}

func readAcl(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	command := args[0]
	myAcl, err := GetACL(command, db, c)
	if err != nil {
		irc.Reply(m, "Could not fetch the requested ACL:")
		irc.Reply(m, err.Error())
		return true
	}
	irc.Reply(m, fmt.Sprintf("ACL for %s", command))
	irc.Reply(m, "Users:")
	for nick := range myAcl.nicks {
		irc.Reply(m, fmt.Sprintf("\t%s", nick))
	}
	irc.Reply(m, "Channels:")
	for channel := range myAcl.channels {
		irc.Reply(m, fmt.Sprintf("\t%s", channel))
	}
	return true
}

func changePass(args []string, irc *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	newPass := args[0]
	// Make a message to nickserv. I know this is hacky, but better than forging a message from scratch.
	requestor := m.From
	m.From = "NickServ"
	irc.Reply(m, fmt.Sprintf("SET PASSWORD %s", newPass))
	m.From = requestor
	irc.Reply(m, "Password changed. Do not forget to change the configuration too.")
	return false
}
