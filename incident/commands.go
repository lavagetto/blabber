package incident

import "blabber/triggers"

// IrcCommands is a container for all commands defined in this module
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
	triggers.NewCommand(
		"incidents",
		"",
		"Shows a list of open incidents",
		true,
		true,
		listOpenIncidents,
	),
	triggers.NewCommand(
		"incident_details",
		"(?P<id>\\d+)$",
		"Gets all the details about an incident.",
		true,
		true,
		formatIncident,
	),
}
