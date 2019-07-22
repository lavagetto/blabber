package triggers

import (
	"blabber/bot"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

/*
	Commands section
*/
type commandClosure func(
	args []string,
	c *bot.Configuration,
	db *sql.DB,
) TriggerFunc

// Command encapsulates an irc command
type Command struct {
	ID              string
	ArgumentsRegexp *regexp.Regexp
	HelpMsg         string
	privmsg         bool
	public          bool
	action          commandClosure
}

// NewCommand allows to declare a full-featured IRC command.
// It allows the author to focus just on the business logic and not on the
// boilerplate of authz/authn, and also guarantees uniformity of implementation.

// Arguments:
// name string the command identifier, that will be used to call it
// regexString string representing a regexp to match the command parameters, if any.
//   Use subgroup matching with named parameters if you want an automatic pretty-print in the help
// public bool indicating if this command can be called (and replied to) in public
// private bool indication if this command can be called (and replied to) in private message
// action a commandClosure function that describes the action to take.
func NewCommand(
	name string,
	regexString string,
	help string,
	public bool,
	private bool,
	action commandClosure,
) *Command {
	var fullRegexp string
	if regexString == "" {
		fullRegexp = fmt.Sprintf("\\!%s\\s*$", name)
	}
	fullRegexp = fmt.Sprintf("\\!%s\\s%s", name, regexString)
	argRegexp := regexp.MustCompile(fullRegexp)
	command := Command{
		ID:              name,
		ArgumentsRegexp: argRegexp,
		HelpMsg:         help,
		privmsg:         private,
		public:          public,
		action:          action,
	}
	return &command
}

// Returns the condition
func (cmd *Command) getCondition(c *bot.Configuration) TriggerFunc {
	return func(bot *hbot.Bot, m *hbot.Message) bool {
		// The action is triggered to private messages for !command
		// or public messages for <bot-nick>: !command
		// depending on if they're enabled or not
		if m.Command != "PRIVMSG" {
			return false
		}

		if cmd.privmsg && m.To == c.NickName {
			return strings.HasPrefix(m.Content, "!"+cmd.ID)
		}
		if cmd.public && strings.HasPrefix(m.To, "#") {
			comp := fmt.Sprintf("%s: !%s", c.NickName, cmd.ID)
			return strings.HasPrefix(m.Content, comp)
		}
		return false
	}
}

func (cmd *Command) getAction(c *bot.Configuration, db *sql.DB) TriggerFunc {
	return func(irc *hbot.Bot, m *hbot.Message) bool {
		acl, err := GetACL(cmd.ID, db, c)
		if err != nil {
			// We log the issue, but we don't stop admins from being able to perform commands.
			log.Error("Couldn't fetch the ACLs", "error", err.Error())
		}
		if !acl.IsAllowed(m) {
			irc.Reply(m, "You're not allowed to perform this action.")
			return true
		}
		// Now validate the content of the string
		matches := cmd.ArgumentsRegexp.FindStringSubmatch(m.Content)
		if matches == nil {
			irc.Reply(m, "The command is not properly formatted.")
			irc.Reply(m, cmd.formatHelp())
			return false
		}
		actionFunc := cmd.action(matches[1:], c, db)
		return actionFunc(irc, m)
	}
}

func (cmd *Command) formatHelp() string {
	parameters := []string{fmt.Sprintf("!%s", cmd.ID)}
	for i, parameter := range cmd.ArgumentsRegexp.SubexpNames()[1:] {
		if parameter == "" {
			parameter = fmt.Sprintf("arg%d", i)
		}
		parameters = append(parameters, fmt.Sprintf("<%s>", parameter))
	}
	return fmt.Sprintf("%s. Format: %s", cmd.HelpMsg, strings.Join(parameters, " "))

}

func (cmd *Command) getHandler() handlerClosure {
	return func(c *bot.Configuration, db *sql.DB) *EvHandler {
		condition := cmd.getCondition(c)
		action := cmd.getAction(c, db)
		return NewHandler(condition, action, cmd.formatHelp())
	}
}
