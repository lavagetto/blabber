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
	[]string,
	*hbot.Bot,
	*hbot.Message,
	*bot.Configuration,
	*sql.DB,
) bool

// Command encapsulates an irc command
type Command struct {
	// The command identifier - it will determine how
	// the command is called
	ID              string
	ArgumentsRegexp *regexp.Regexp
	HelpMsg         string
	privmsg         bool
	public          bool
	Action          commandClosure
	Db              *sql.DB
	Configuration   *bot.Configuration
}

// NewCommand allows to declare a full-featured IRC command.
// It allows the author to focus just on the business logic and not on the
// boilerplate of authz/authn, and also guarantees uniformity of implementation.

// Arguments:
// name string  it
// regexString string representing a regexp to match the command parameters, if any.
//   Use subgroup matching with named parameters if you want an automatic pretty-print in the help
// public bool indicating if this command can be called (and replied to) in public
// private bool indication if this command can be called (and replied to) in private message
// action a commandClosure function that describes the action to take.
func NewCommand(
	// the command identifier, that will be used to call it
	name string,
	regexString string,
	help string,
	public bool,
	private bool,
	action commandClosure,
) *Command {
	var fullRegexp string
	if regexString == "" {
		fullRegexp = fmt.Sprintf(`\!(%s)\s*$`, name)
	} else {
		fullRegexp = fmt.Sprintf("\\!%s\\s%s", name, regexString)
	}
	argRegexp := regexp.MustCompile(fullRegexp)
	command := Command{
		ID:              name,
		ArgumentsRegexp: argRegexp,
		HelpMsg:         help,
		privmsg:         private,
		public:          public,
		Action:          action,
	}
	return &command
}

// Checks if we should act on the event.
func (cmd Command) isCommand(bot *hbot.Bot, m *hbot.Message) bool {
	// The action is triggered to private messages for !command
	// or public messages for <bot-nick>: !command
	// depending on if they're enabled or not
	if m.Command != "PRIVMSG" {
		return false
	}
	if strings.HasPrefix(m.Content, "!"+cmd.ID) {
		return true
	}
	if cmd.public && strings.HasPrefix(m.To, "#") {
		comp := fmt.Sprintf("%s: !%s", cmd.Configuration.NickName, cmd.ID)
		return strings.HasPrefix(m.Content, comp)
	}
	return false
}

func (cmd Command) checkAcl(irc *hbot.Bot, m *hbot.Message) bool {
	acl, err := GetACL(cmd.ID, cmd.Db, cmd.Configuration)
	if err != nil {
		// We log the issue, but we don't stop admins from being able to perform commands.
		log.Error("Couldn't fetch the ACLs", "error", err.Error())
	}
	if !acl.IsAllowed(m) {
		irc.Reply(m, "You're not allowed to perform this action.")
		return false
	} else {
		return true
	}

}

func (cmd Command) doAction(irc *hbot.Bot, m *hbot.Message) bool {
	// Validate the content of the string
	matches := cmd.ArgumentsRegexp.FindStringSubmatch(m.Content)
	if matches == nil {
		irc.Reply(m, "The command is not properly formatted.")
		irc.Reply(m, cmd.Help())
		return false
	}
	return cmd.Action(matches[1:], irc, m, cmd.Configuration, cmd.Db)
}

func (cmd Command) Help() string {
	parameters := []string{fmt.Sprintf("!%s", cmd.ID)}
	for i, parameter := range cmd.ArgumentsRegexp.SubexpNames()[1:] {
		if parameter == "" {
			parameter = fmt.Sprintf("arg%d", i)
		}
		parameters = append(parameters, fmt.Sprintf("<%s>", parameter))
	}
	return fmt.Sprintf("%s. Format: %s", cmd.HelpMsg, strings.Join(parameters, " "))

}

func (cmd Command) Handle(irc *hbot.Bot, m *hbot.Message) bool {
	//log.Info("Handling message", "command", m.Command, "to", m.To, "content", m.Content)
	if cmd.isCommand(irc, m) && cmd.checkAcl(irc, m) {
		return cmd.doAction(irc, m)
	} else {
		return false
	}
}
