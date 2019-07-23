package main

// Simple chatbot started to help the wikimedia SRE team during outages
/*
Copyright (C) 2019  Giuseppe Lavagetto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
*/

import (
	"blabber/bot"
	"blabber/contact"
	"blabber/incident"
	"blabber/triggers"
	"flag"

	_ "github.com/mattn/go-sqlite3"
	log "gopkg.in/inconshreveable/log15.v2"
)

var configFile = flag.String("config", "config.json", "Optional configuration file (JSON)")

func main() {
	flag.Parse()
	conf, err := bot.GetConfig(*configFile)
	if err != nil {
		log.Info("Could not open configuration file")
	}
	bbot, err := bot.NewBot(conf)
	if err != nil {
		panic(err)
	}
	defer bbot.DB.Close()

	registry := triggers.NewRegistry(conf, bbot.DB)
	// Basic bot - does rickrolling and manages ACLs
	registry.RegisterCommands(triggers.IrcCommands)
	// Incident related - the first is a simple event handler with no command associated
	registry.Register("store_topic", incident.StoreTopic, "")
	registry.RegisterCommands(incident.IrcCommands)
	// Contact list related
	registry.RegisterCommands(contact.IrcCommands)
	registry.AddAll(bbot)
	bbot.Irc.Run()
}
