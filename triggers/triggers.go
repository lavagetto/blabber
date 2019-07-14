package triggers

import (
	"blabber/bot"
	"database/sql"
	"errors"
	"fmt"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"

	hbot "github.com/whyrusleeping/hellabot"
)

type triggerFunc func(bot *hbot.Bot, m *hbot.Message) bool

type EvHandler struct {
	condition triggerFunc
	action    triggerFunc
	HelpMsg   string
}

// NewHandler generates a new handler for use in code.
func NewHandler(condition triggerFunc, action triggerFunc, HelpMsg string) *EvHandler {
	return &EvHandler{condition: condition, action: action, HelpMsg: HelpMsg}
}
func (ev *EvHandler) getTrigger() hbot.Trigger {
	return hbot.Trigger{ev.condition, ev.action}
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

// Register an handler
func (r *Registry) Register(id string, handler handlerClosure) error {
	if _, ok := r.handlers[id]; ok {
		msg := fmt.Sprintf("Cannot register handler with id '%s' twice", id)
		return errors.New(msg)
	}
	r.handlers[id] = handler(r.config, r.db)
	return nil
}

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
			bot.Reply(m, fmt.Sprintf("%s - irc bot for handling outage", r.config.NickName))
			bot.Reply(m, "")
			bot.Reply(m, "Available commands:")
			bot.Reply(m, "!help\tPrints this message")
			for id, EvHandler := range r.handlers {
				if EvHandler.HelpMsg != "" {
					bot.Reply(m, fmt.Sprintf("%-30s%s\n", id, EvHandler.HelpMsg))
				}
			}
			return true
		},
	})
}

func (r *Registry) help(bot *hbot.Bot) {
	fmt.Println("")
	fmt.Println("Available commands:")
	fmt.Println("!help\tPrints this message")

}

// Closures!
func RickRoll(c *bot.Configuration, db *sql.DB) *EvHandler {
	var lyrics = []string{
		"Never gonna give you up",
		"Never gonna let you down",
		"Never gonna run around and desert you",
		"Never gonna make you cry",
		"Never gonna say goodbye",
		"Never gonna tell a lie and hurt you",
	}
	var condition = func(bot *hbot.Bot, m *hbot.Message) bool {
		// Only public requests for singing will be accepted.
		if m.Content == fmt.Sprintf("%s: sing!", c.NickName) {
			for _, channel := range c.Channels {
				if m.To == channel {
					return true
				}
			}
		}
		return false
	}
	var action = func(bot *hbot.Bot, m *hbot.Message) bool {
		for _, line := range lyrics {
			bot.Reply(m, line)
			time.Sleep(800 * time.Millisecond)
		}
		return true
	}
	return &EvHandler{condition, action, "Sings a nice tune to you (public only). Format: sing!"}
}
