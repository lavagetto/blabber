package incident

import "blabber/triggers"

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
		"incident_close",
		"(?P<id>\\d+)$",
		"Closes an incident",
		true,
		false,
		stopIncident,
	),
}
