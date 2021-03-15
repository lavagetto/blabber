package bot

import (
	"database/sql"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
	log "gopkg.in/inconshreveable/log15.v2"
)

// Bot is the basic bot with a state storage and a database connection.
type Bot struct {
	Irc *hbot.Bot
	DB  *sql.DB
}

// NewBot returns a new bot instance
func NewBot(config *Configuration) (*Bot, error) {
	channels := func(bot *hbot.Bot) {
		bot.Channels = config.Channels
	}
	// Do not hijack the session, use TLS and SASL if requested
	botOptions := func(bot *hbot.Bot) {
		bot.HijackSession = false
		if config.UseTLS {
			bot.SSL = true
		}
		if config.UseSASL {
			bot.SASL = true
			bot.Password = config.Password
		}

	}

	irc, err := hbot.NewBot(config.GetServerString(), config.NickName, botOptions, channels)
	if err != nil {
		return nil, err
	}
	logHandler := log.LvlFilterHandler(log.LvlInfo, log.StdoutHandler)
	irc.Logger.SetHandler(logHandler)
	if err != nil {
		return nil, err
	}
	db, err := newSQL(config.DbDsn)
	if err != nil {
		return nil, err
	}
	b := Bot{irc, db}
	return &b, nil
}

func newSQL(dsn string) (*sql.DB, error) {
	parsedDsn := strings.Split(dsn, "://")
	db, err := sql.Open(parsedDsn[0], parsedDsn[1])
	if err != nil {
		return nil, err
	}
	return db, nil
}
