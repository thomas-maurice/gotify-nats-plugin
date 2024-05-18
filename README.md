# gotify-nats-plugin

This is a plugin to get notifications from NATS.

## How does it work ?
The plugin subscribes to a NATS subject and expects messages matching this structure:
```go
type NatsJSONMessage struct {
	Priority int     `json:"priority"`  // Optional: Priority of the message (0-10)
	Title    string  `json:"title"`     // Optional: Title of the notification
	Message  string  `json:"message"`   // Message of the notification
	Markdown *bool   `json:"markdown"`  // Optional: Format the message as markdown ?
	URL      *string `json:"url"`       // Optional: Click URL for Android notifications
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

# Supported authentication schemes

The following authentication schemes are supported when connecting to a NATS server/cluster:

* No authentication (boooooo)
* Token
* Username/Password
* NKEY
* NKEY + JWT token when the accounts are managed via an operator

Note that for now self signed certificates are not supported.

## Configuration
The plugin can be configured as so in the plugin config screen:
```yaml
---
# URL of the nats server you are going to connect to
nats_server_url: nats://localhost:4222
# Subject to listen on for messages
# Note that the subject can be a wildcard like one of these
#  - gotify
#  - gotify.*
#  - gotify.>
subject: gotify
# Should we render the messages as markdown by default ?
# you can still opt out by setting `markdown` to false in
# the NATS payload
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

# Compatibility matrix

Given the rapid development of NATS compared to Gotify and the number of core libs they share in common (`x/crypto` mainly)
it is very hard and annoying (or even impossible) to release versions of the plugin for the X latest versions of Gotify. Indeed these will cause conflicts when trying to import the plugin.

This is because for example a reasonably recent version of NATS is going to require `x/crypto@v0.17.0` when Gotify will be compiled with `v0.13.0` for example.

I will try to put some effort at some point to make it better, but please assume that the released binaries are going to work for whichever version of Gotify was the stable one at the time.