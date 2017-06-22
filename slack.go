package labbot

import (
	"fmt"
	"sort"

	"go.uber.org/zap"

	"strings"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const botName = "chihiro"

func (l *labbot) msgEvent(rtm *slack.RTM, botID string, reply <-chan replyData) {
	mention := fmt.Sprintf("<@%s>", botID)
	for data := range reply {
		if !strings.Contains(data.Text, mention) {
			continue
		}
		// 誰がいる?
		if strings.Contains(data.Text, "誰がい") {
			list := make([]string, 0, len(timeStamp))
			for _, who := range timeStamp {
				if who.Inlab {
					list = append(list, who.Name)
				}
			}
			sort.Strings(list)
			rtm.SendMessage(rtm.NewOutgoingMessage("研究室には"+strings.Join(list, "、")+"がいます！", data.Channel))
		}
	}
}

// Not rtm
func (l *labbot) sendToSlack(channel, msg string) {
	channelID, err := l.findChannelID(channel)
	if err != nil {
		l.Error("Failed to find channel id", zap.Error(err))
		return
	}
	params := parameter()
	_, timestamp, err := l.PostMessage(channelID, msg, params)
	if err != nil {
		l.Warn(`Failed to post slack`, zap.Error(err), zap.String("message", msg))
		return
	}
	l.Info(
		"Message successfully sent to slack",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}

func (l *labbot) findUserID(username string) (string, error) {
	// Get users infomation
	users, err := l.GetUsers()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get slack users")
	}

	for _, user := range users {
		if username == user.Name {
			return user.ID, nil
		}
	}
	return "", fmt.Errorf("Could not find id for %s", username)
}

func (l *labbot) findChannelID(name string) (string, error) {
	// Get slack channnels
	channels, err := l.GetChannels(false)
	if err != nil {
		return "", errors.Wrap(err, "Failed to get slack channnel")
	}
	for _, channel := range channels {
		if channel.Name == name {
			return channel.ID, nil
		}
	}
	return "", fmt.Errorf("Could not find ChannelID of #%s", name)
}

func parameter() slack.PostMessageParameters {
	return slack.PostMessageParameters{
		Username:  botName,
		AsUser:    true,
		LinkNames: 1,
	}
}
