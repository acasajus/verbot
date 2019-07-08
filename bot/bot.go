// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
)

type Bot struct {
	client       *model.Client4
	exitChan     chan struct{}
	user         *model.User
	team         *model.Team
	debugChannel *model.Channel
	wsClient     *model.WebSocketClient
}

func Connect(conf Conf) (*Bot, error) {

	client := model.NewAPIv4Client(conf.Url)

	bot := &Bot{
		client:   client,
		exitChan: make(chan struct{}),
	}
	if err := bot.makeSureServerIsRunning(); err != nil {
		return nil, err
	}
	if err := bot.login(conf.Login, conf.Password); err != nil {
		return nil, err
	}
	if err := bot.updateBotInfo(conf.Bot); err != nil {
		return nil, err
	}
	if err := bot.findTeam(conf.Team); err != nil {
		return nil, err
	}
	if err := bot.createDebugChannel(conf.DebugChannel); err != nil {
		return nil, err
	}

	bot.sendDebugMsg("_"+bot.user.Username+" has **started** running_", "")

	wsUrl := strings.Replace(conf.Url, "http", "ws", 1)
	wsClient, err := model.NewWebSocketClient4(wsUrl, client.AuthToken)
	if err != nil {
		return nil, errors.Wrap(err, "We failed to connect to the web socket")
	}

	bot.setupGracefulShutdown()
	wsClient.Listen()

	go func() {
		for {
			select {
			case resp := <-wsClient.EventChannel:
				bot.handleWebSocketResponse(resp)
			}
		}
	}()

	return bot, nil
}

func (bot *Bot) makeSureServerIsRunning() error {
	props, resp := bot.client.GetOldClientConfig("")
	if resp.Error != nil {
		return errors.Wrap(resp.Error, "There was a problem pinging the Mattermost server.  Are you sure it's running?")
	}
	log.Printf("Server detected and is running version %s", props["Version"])
	return nil
}

func (bot *Bot) login(user, pass string) error {
	userObj, resp := bot.client.Login(user, pass)
	if resp.Error != nil {
		return errors.Wrap(resp.Error, "There was a problem logging into the Mattermost server")
	}
	bot.user = userObj
	return nil
}

func (bot *Bot) updateBotInfo(name BotNameConf) error {
	if bot.user.FirstName != name.First || bot.user.LastName != name.Last || bot.user.Username != name.Username {
		bot.user.FirstName = name.First
		bot.user.LastName = name.Last
		bot.user.Username = name.Username

		if _, resp := bot.client.UpdateUser(bot.user); resp.Error != nil {
			return errors.Wrap(resp.Error, "We failed to update the Bot user")
		}
	}
	return nil
}

func (bot *Bot) findTeam(team string) error {
	teamObj, resp := bot.client.GetTeamByName(team, "")
	if resp.Error != nil {
		return errors.Wrap(resp.Error, fmt.Sprintf("We do not appear to be a member of the team '%s'", team))
	}
	bot.team = teamObj
	return nil
}

func (bot *Bot) createDebugChannel(channelName string) error {
	if rchannel, resp := bot.client.GetChannelByName(channelName, bot.team.Id, ""); resp.Error == nil {
		bot.debugChannel = rchannel
		return nil
	}

	// Looks like we need to create the logging channel
	channel := &model.Channel{}
	channel.Name = channelName
	channel.DisplayName = fmt.Sprintf("Debugging For %s Bot", bot.user.Username)
	channel.Purpose = "This is used as a test channel for logging bot debug messages"
	channel.Type = model.CHANNEL_OPEN
	channel.TeamId = bot.team.Id
	if rchannel, resp := bot.client.CreateChannel(channel); resp.Error != nil {
		return errors.Wrap(resp.Error, "We failed to create the debug channel")
	} else {
		bot.debugChannel = rchannel
	}
	return nil
}

func (bot *Bot) sendDebugMsg(msg string, replyToId string) {
	post := &model.Post{}
	post.ChannelId = bot.debugChannel.Id
	post.Message = msg

	post.RootId = replyToId

	if _, resp := bot.client.CreatePost(post); resp.Error != nil {
		log.Printf("We failed to send a message to the logging channel: %s", resp.Error)
	}
}

func (bot *Bot) handleWebSocketResponse(event *model.WebSocketEvent) {
	bot.handleMsgFromDebuggingChannel(event)
}

func (bot *Bot) handleMsgFromDebuggingChannel(event *model.WebSocketEvent) {
	// If this isn't the debugging channel then lets ingore it
	if event.Broadcast.ChannelId != bot.debugChannel.Id {
		return
	}

	// Lets only reponded to messaged posted events
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}

	log.Println("Responding to debugging channel msg")

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
	if post != nil {
		// ignore my events
		if post.UserId == bot.user.Id {
			return
		}

		// if you see any word matching 'alive' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)alive(?:$|\W)`, post.Message); matched {
			bot.sendDebugMsg("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'up' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)up(?:$|\W)`, post.Message); matched {
			bot.sendDebugMsg("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'running' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)running(?:$|\W)`, post.Message); matched {
			bot.sendDebugMsg("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'hello' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)hello(?:$|\W)`, post.Message); matched {
			bot.sendDebugMsg("Yes I'm running", post.Id)
			return
		}
	}

	bot.sendDebugMsg("I did not understand you!", post.Id)
}

func (bot *Bot) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if bot.wsClient != nil {
				bot.wsClient.Close()
			}

			bot.sendDebugMsg("_"+bot.user.Username+" has **stopped** running_", "")
			bot.exitChan <- struct{}{}
		}
	}()
}

func (bot *Bot) Wait() {
	<-bot.exitChan
}
