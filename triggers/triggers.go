package triggers

import (
	"blabber/bot"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	log "gopkg.in/inconshreveable/log15.v2"

	hbot "github.com/whyrusleeping/hellabot"
)

type TriggerFunc func(bot *hbot.Bot, m *hbot.Message) bool

// EvHandler is the basic structure for holding information about an
// irc trigger. For most interactive commands, where you want to define ACLs,
// input validation, etc. you should use Command instead.
type EvHandler struct {
	condition TriggerFunc
	action    TriggerFunc
	HelpMsg   string
}

// NewHandler generates a new handler for use in code.
func NewHandler(condition TriggerFunc, action TriggerFunc, HelpMsg string) *EvHandler {
	return &EvHandler{condition: condition, action: action, HelpMsg: HelpMsg}
}
func (ev *EvHandler) getTrigger() hbot.Trigger {
	return hbot.Trigger{Condition: ev.condition, Action: ev.action}
}

// Registry is a container for all event handlers.
type Registry struct {
	handlers map[string]*EvHandler
	config   *bot.Configuration
	db       *sql.DB
}

func NewRegistry(c *bot.Configuration, db *sql.DB) *Registry {
	var r Registry
	r.handlers = make(map[string]*EvHandler)
	r.config = c
	r.db = db
	return &r
}

type handlerClosure func(config *bot.Configuration, db *sql.DB) *EvHandler

// Register an handler. This is the basic interface you should use if you're not crating
// a proper command, but rather a trigger.
// For interactive commands, please use RegisterCommand below.
func (r *Registry) Register(id string, handler handlerClosure) error {
	if _, ok := r.handlers[id]; ok {
		msg := fmt.Sprintf("Cannot register handler with id '%s' twice", id)
		return errors.New(msg)
	}
	r.handlers[id] = handler(r.config, r.db)
	return nil
}

// RegisterCommand allows to register a full-featured IRC command.
func (r *Registry) RegisterCommand(command *Command) error {
	return r.Register(command.ID, command.getHandler())
}

func (r *Registry) RegisterCommands(commands []*Command) error {
	for _, command := range commands {
		err := r.RegisterCommand(command)
		if err != nil {
			return err
		}
	}
	return nil
}

// Deregister removes one handler from the system.
func (r *Registry) Deregister(id string) {
	delete(r.handlers, id)
}

func (r *Registry) AddAll(b *bot.Bot) {
	for id, EvHandler := range r.handlers {
		log.Info("Registering handler", "id", id)
		b.Irc.AddTrigger(EvHandler.getTrigger())
	}
	r.addHelp(b)
}

// Help prints out the help for the registered commands
func (r *Registry) addHelp(b *bot.Bot) {
	b.Irc.AddTrigger(hbot.Trigger{
		func(bot *hbot.Bot, m *hbot.Message) bool {

			if m.Command == "PRIVMSG" && m.Content == "!help" {
				return true
			} else if m.Content == fmt.Sprintf("%s: !help", r.config.NickName) {
				return true
			}
			return false
		},
		func(bot *hbot.Bot, m *hbot.Message) bool {
			bot.Reply(m, fmt.Sprintf("%s - irc bot for handling outages", r.config.NickName))
			bot.Reply(m, "")
			bot.Reply(m, "Available commands:")
			bot.Reply(m, fmt.Sprintf("%-16s%s\n", "!help", "Prints this message"))
			var handlers = make([]string, 0, len(r.handlers))
			for id := range r.handlers {
				handlers = append(handlers, id)
			}
			sort.Strings(handlers)
			for _, id := range handlers {
				EvHandler, ok := r.handlers[id]
				if !ok {
					// TODO: log something
					continue
				}
				if EvHandler.HelpMsg != "" {
					bot.Reply(m, fmt.Sprintf("%-16s%s\n", id, EvHandler.HelpMsg))
				}
			}
			return true
		},
	})
}
