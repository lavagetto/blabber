package triggers

import (
	"blabber/bot"
	"database/sql"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
)

// Commands.

// RickRoll
var lyrics = []string{
	"Never gonna give you up",
	"Never gonna let you down",
	"Never gonna run around and desert you",
	"Never gonna make you cry",
	"Never gonna say goodbye",
	"Never gonna tell a lie and hurt you",
}

func rickRollAction(args []string, bot *hbot.Bot, m *hbot.Message, c *bot.Configuration, db *sql.DB) bool {
	for _, line := range lyrics {
		bot.Reply(m, line)
		time.Sleep(800 * time.Millisecond)
	}
	return true
}

var IrcCommands = []*Command{
	NewCommand(
		"sing",
		"",
		"Sings for you a nice tune",
		true,
		false,
		rickRollAction,
	),
	NewCommand(
		"acl_add",
		"(?P<command>\\S+)\\s+(?P<nick_or_chan>\\S+)\\s*$",
		"Adds the ability for a command to be used by a single user or in a channel",
		false,
		true,
		addACL,
	),
	NewCommand(
		"acl_remove",
		"(?P<command>\\S+)\\s+(?P<nick_or_chan>\\S+)\\s*$",
		"Removes a user from the ACL",
		false,
		true,
		removeAcl,
	),
	NewCommand(
		"acl_get",
		"(?P<command>\\S+)\\s*$",
		"Gets the defined ACLs for a command",
		false,
		true,
		readAcl,
	),
	NewCommand(
		"change_pass",
		"(?P<password>\\S+)\\s*$",
		"Changes the nickserv password",
		false,
		true,
		changePass,
	),
}
