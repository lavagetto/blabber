package incident

import "blabber/triggers"

// IrcCommands is a container for all commands
var IrcCommands = []*triggers.Command{
	triggers.NewCommand(
		"incident_start",
		"(?P<severity>\\d+)\\s+(?P<components_comma_sep>.+)$",
		"Start an incident",
		true,
		false,
		startIncident,
	),
	triggers.NewCommand(
		"incident_update",
		`(?P<id>\d+)\s+(?P<what>severity|description)\s+(?P<value>.+)$`,
		"Update an incident. You can update either severity or the incident description",
		true,
		false,
		updateIncident,
	),
	triggers.NewCommand(
		"incident_close",
		"(?P<id>\\d+)$",
		"Closes an incident",
		true,
		false,
		stopIncident,
	),
}
