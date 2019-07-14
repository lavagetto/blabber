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

## FAQ
Q: Is blabber useful for X?
A: No.
