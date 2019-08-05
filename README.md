# blabber
Simple IRC bot written in golang. Mostly a toy.

It can be built as any common go application.

## Running blabber

It accepts a single command-line parameter, `-config`, allowing to pass the name of the config file to read (which is `config.json` by default).

A typical configuration file will look as follows:

```json
{
    "server": "irc.mynetwork.com",
    "port": 6697,
    "use_tls": true,
    "nick": "BlabberBot",
    "password": "mysecretpassword",
    "use_sasl": true,
    "channels": ["#channel1", "#channel2"],
    "db_dsn": "sqlite:///srv/blabber/blabber.db
}
```
To generate the schema of the database, run:
```bash
sqlite3 blabber.db < schema.sql
```

## Available Commands.

You can list the implemented commands using `!help`

## ACLs

Only people listed as admins in the configuration will have free access to all commands.

You can grant one user, or a channel the right to use a command as follows:

```
# Allow a user to use a command
you > !acl_add contact_add SomeFriend
BlabberBot>	The ACL was saved.
# See the acl
you > !acl_get contact_add
BlabberBot>	ACL for contact_add
BlabberBot>	Users:
BlabberBot>		you
BlabberBot>		SomeFriend
BlabberBot>	Channels:
# Allow all users in a channel to use a command
you > !acl_add contact_add #thischan
BlabberBot>	The ACL was saved.
# Remove the authorization to a user
you > !acl_remove contact_add SomeFriend
BlabberBot>	The ACL was succesfully removed.
```
## Implemented commands

We have a couple set of commands implemented right now: incident-related commands and contact-list related commands.

### Incidents

An incident is initiated by the command `!incident_start <severity> <comp1>,[comp2,comp3..]`

Severity goes from 5 (minor issue) to 1 (full outage).

When you start an incident, blabber will:
* Register the data in its database
* Create a new google doc for the incident, from a template
* Change the topic of all channels it is in, updating the status and (on channels designed private) a link to the google doc

You can open as many incidents as you like, but hopefully you just have one to manage at the same time!

Updating an incident can be done via `!incident_update <id> [severity|description] <value>`.
It allows to change the severity of the incident, and to add a new piece of text to its description.

Those data can be retrieved with `!incident_details <id>`. In the future, data passed to !incident_update will also be added to the google document.

Finally, an incident gets closed with `!incident_close <id>`.

### Contacts

Very simple interface, you add a new contact with `!contact_add`, and retrieve it with `!contact_get`.

## Implement your additional commands

You just need to generate a callback that will react to the command, in the form:

```golang
c := triggers.NewCommand(
    "name", // The name will identify how the command gets called
    "(?P<some_regex>aAsS+)", // the regexp that identifies arguments passed to the callback
    true, // set to true if you want the command to be used from a public channel
    true, // set to true if you want the command to be used in private communication with the bot
    cb, // the callback that will be excuted
)
// The callback should be of type triggers.globalClosure
```

## FAQ
Q: Is blabber useful for X?
A: No.
