# gotify-nats-plugin

This is a plugin to get notifications from NATS.

## How does it work ?
The plugin subscribes to a NATS subject and expects messages matching this structure:
```go
type NatsJSONMessage struct {
	Priority int     `json:"priority"`  # Optional: Priority of the message (0-10)
	Title    string  `json:"title"`     # Optional: Title of the notification
	Message  string  `json:"message"`   # Message of the notification
	Markdown *bool   `json:"markdown"`  # Optional: Format the message as markdown ?
	URL      *string `json:"url"`       # Optional: Click URL for android notifications
}
```

An example payload would look like this
```json
{
  "title": "hello **WORLD** 3",
  "message": "this is **the message**",
  "priority": 10,
  "markdown": false,
  "url": "https://google.fr"
}
```

To send it via the nats-cli:
```bash
$ nats pub gotify '{"title": "hello world", "message": "this is **the message**", "priority": 10, "markdown": false, "url": "https://google.fr"}'
```

## Configuration
The plugin can be configured as so in the plugin config screen:
```yaml
---
# URL of the nats server you are going to connect to
nats_server_url: nats://localhost:4222
# Subject to listen on for messages
subject: gotify
# Should we render the messages as markdown by default ?
markdown: true
# Default priority if none is specified
default_message_priority: 5
# Authentication config -- delete if no authentication is needed
auth:
  # token is used *only* for token auth
  token: foobar

  # username/password are used solely for username/password auth
  username: foo
  password: bar

  # nkey can be used on it's own, or in conjunction with a user JWT
  nkey: SUALLJRV33UOFX7TNHYFXP43YVQDAM4N3FEGP2W62EOHKYA7UFI7GCU6JU

  # jwt *must* be used in conjunction with an nkey, this is the case
  # where you are authenticating with an identity created by a NATS
  # operator
  jwt: ey..........[long jwt]......
# do we print debug messages ?
debug: false
```