package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gotify/plugin-api"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/sirupsen/logrus"
)

func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		ModulePath:  "github.com/thomas-maurice/gotify-nats-plugin",
		Version:     "0.0.1",
		Author:      "Thomas Maurice",
		Website:     "https://github.com/thomas-maurice/gotify-nats-plugin",
		Description: "NATS Plugin -- Send messages from Gotify through NATS",
		License:     "MIT",
		Name:        "NATS plugin",
	}
}

type NatsJSONMessage struct {
	Priority int     `json:"priority"`
	Title    string  `json:"title"`
	Message  string  `json:"message"`
	Markdown *bool   `json:"markdown"`
	URL      *string `json:"url"`
}

type NatsAuthConfig struct {
	Token    *string `json:"token" yaml:"token"`
	Username *string `json:"username" yaml:"username"`
	Password *string `json:"password" yaml:"password"`
	Nkey     *string `json:"nkey" yaml:"nkey"`
	Jwt      *string `json:"jwt" yaml:"jwt"`
}

type NatsPluginConfig struct {
	NatsServerURL          string          `json:"nats_server_url" yaml:"nats_server_url"`
	Subject                string          `json:"subject" yaml:"subject"`
	DefaultMessagePriority int             `json:"default_message_priority" yaml:"default_message_priority"`
	Auth                   *NatsAuthConfig `json:"auth" yaml:"auth"`
	Debug                  bool            `json:"debug" yaml:"debug"`
	Markdown               bool            `json:"markdown" yaml:"markdown"`
}

type GotifyClick struct {
	URL string `json:"url"`
}

type GotifyExtraNotification struct {
	Click       *GotifyClick `json:"click"`
	BigImageURL *string      `json:"bigImageUrl"`
}

type NatsPlugin struct {
	msgHandler plugin.MessageHandler
	userCtx    plugin.UserContext
	basePath   string

	logger *logrus.Entry

	config *NatsPluginConfig

	natsClient *nats.Conn
}

func (c *NatsPlugin) SetMessageHandler(h plugin.MessageHandler) {
	c.msgHandler = h
}

func (p *NatsPlugin) Enable() error {
	p.logger.Info("Initialising NATS plugin")

	if p.natsClient != nil {
		p.natsClient.Close()
	}

	conn, err := p.getNatsClient()
	if err != nil {
		p.logger.WithError(err).Error("Failed to connect to NATS")
		return err
	}

	p.natsClient = conn

	go func() {
		p.logger.Debug("Started listening on NATS messages")
		p.consumeMessages()
		p.natsClient.SetErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
			p.logger.WithError(err).Error("NATS error occured")
		})
		p.natsClient.SetDisconnectHandler(func(c *nats.Conn) {
			p.logger.Warn("NATS connection lost")
		})
		p.natsClient.SetReconnectHandler(func(c *nats.Conn) {
			p.logger.Info("Connection to NATS server restored")
		})
	}()

	return nil
}

func (p *NatsPlugin) Disable() error {
	p.logger.Debug("Disabling plugin")
	if p.natsClient != nil {
		p.natsClient.Close()
	}

	return nil
}

func (p *NatsPlugin) consumeMessages() {
	p.logger.Debugf("Listening on subject: %s", p.config.Subject)
	p.natsClient.Subscribe(p.config.Subject, func(m *nats.Msg) {
		p.logger.Debugf("recieved payload: %s", string(m.Data))

		var msg NatsJSONMessage
		err := json.Unmarshal(m.Data, &msg)
		if err != nil {
			p.logger.WithError(err).Error("Failed to unmarshal incoming message")
			return
		}

		gotifyMsg := plugin.Message{
			Message: msg.Message,
			Title:   msg.Title,
			Extras:  make(map[string]interface{}),
		}

		if msg.Priority != 0 {
			gotifyMsg.Priority = msg.Priority
		} else {
			gotifyMsg.Priority = p.config.DefaultMessagePriority
		}

		if p.config.Markdown {
			gotifyMsg.Extras["client::display"] = map[string]string{"contentType": "text/markdown"}
		}

		if msg.Markdown != nil {
			if *msg.Markdown {
				gotifyMsg.Extras["client::display"] = map[string]string{"contentType": "text/markdown"}
			} else {
				gotifyMsg.Extras["client::display"] = map[string]string{"contentType": "text/plain"}
			}
		}

		gotifyExtrasNotification := GotifyExtraNotification{}

		if msg.URL != nil {
			gotifyExtrasNotification.Click = &GotifyClick{
				URL: *msg.URL,
			}
		}

		gotifyMsg.Extras["client::notification"] = gotifyExtrasNotification

		err = p.msgHandler.SendMessage(gotifyMsg)
		if err != nil {
			p.logger.WithError(err).Error("Failed to send the message to gotify")
		}
	})
}

func (p *NatsPlugin) RegisterWebhook(basePath string, mux *gin.RouterGroup) {
	p.basePath = basePath
	mux.POST("/hook", func(c *gin.Context) {
	})
}

func (p *NatsPlugin) DefaultConfig() interface{} {
	return &NatsPluginConfig{
		NatsServerURL:          "nats://localhost:4222",
		DefaultMessagePriority: 5,
		Subject:                "gotify",
		Debug:                  false,
		Markdown:               true,
	}
}

func (p *NatsPlugin) ValidateAndSetConfig(c interface{}) error {
	config, ok := c.(*NatsPluginConfig)
	if !ok {
		return errors.New("could not cast interface{} to NatsPluginConfig")
	}
	p.config = config

	if p.config.Debug {
		p.logger.Logger.SetLevel(logrus.DebugLevel)
		p.logger.Debug("Running logger in debug mode")
	} else {
		p.logger.Logger.SetLevel(logrus.InfoLevel)
	}
	p.logger.Info("Validated configuration")

	return nil
}

func (p *NatsPlugin) getNatsClient() (*nats.Conn, error) {
	connOptions := make([]nats.Option, 0)
	if p.config.Auth != nil {
		if p.config.Auth.Token != nil {
			p.logger.Debug("Using token authentication")
			connOptions = append(connOptions, nats.Token(*p.config.Auth.Token))
		} else if p.config.Auth.Username != nil && p.config.Auth.Password != nil {
			p.logger.Debug("Using username-password authentication")
			connOptions = append(connOptions, nats.UserInfo(*p.config.Auth.Username, *p.config.Auth.Password))
		} else if p.config.Auth.Nkey != nil {
			if p.config.Auth.Jwt != nil {
				p.logger.Debug("Using jwt authentication")
				connOptions = append(connOptions, nats.UserJWTAndSeed(*p.config.Auth.Jwt, *p.config.Auth.Nkey))
			} else {
				p.logger.Debug("Using nkey authentication")
				key, err := nkeys.FromSeed([]byte(*p.config.Auth.Nkey))
				if err != nil {
					return nil, fmt.Errorf("could not derive key pair from seed: %w", err)
				}
				pubKey, err := key.PublicKey()
				if err != nil {
					return nil, fmt.Errorf("could not derive public key from keypair: %w", err)
				}
				connOptions = append(connOptions, nats.Nkey(pubKey, func(nonce []byte) ([]byte, error) {
					kp, err := nkeys.FromSeed([]byte(*p.config.Auth.Nkey))
					if err != nil {
						return nil, fmt.Errorf("unable to derive key pair from seed: %w", err)
					}
					defer kp.Wipe()

					return kp.Sign(nonce)
				}))
			}
		}
	}

	conn, err := nats.Connect(p.config.NatsServerURL, connOptions...)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func (p *NatsPlugin) GetDisplay(location *url.URL) string {
	pluginStatus := ""

	if p.natsClient == nil {
		pluginStatus = "No NATS client detected, is the plugin enabled ?"
	} else {
		pluginStatus += fmt.Sprintf("* Client closed: `%v`\n", p.natsClient.IsClosed())
		pluginStatus += fmt.Sprintf("* Connected server URL: `%v`\n", p.natsClient.ConnectedUrlRedacted())
		pluginStatus += fmt.Sprintf("* Discovered servers: `%v`\n", p.natsClient.DiscoveredServers())
		pluginStatus += fmt.Sprintf("* Last error: `%v`\n", p.natsClient.LastError())
	}

	return fmt.Sprintf(`# gotify-nats-plugin

Sends notifications to your Gotify app from a NATS subscription.

**WARNING**: Changing the configuration of the NATS connection requires a plugin restart (disable/enable)

## Plugin status

%s

## Configuration

**WARNING**: A configuration change requires restarting the plugin (disable/enable)

To configure the plugin, you need to configure it as follows:
`+"```"+`yaml
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
`+"```"+`

## Sending a message

You can send a notification via the nats-cli sending a payload like this one for example:
`+"```bash"+`
$ nats pub gotify '{"title": "hello world", "message": "this is **the message**", "priority": 10, "markdown": false, "url": "https://google.fr"}'
`+"```"+`
`, pluginStatus)
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	logger := logrus.WithFields(
		logrus.Fields{
			"user_id":   ctx.ID,
			"user_name": ctx.Name,
		},
	)
	return &NatsPlugin{
		userCtx: ctx,
		logger:  logger,
	}
}

func main() {
	panic("this should be built as go plugin")
}
