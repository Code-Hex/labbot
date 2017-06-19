package labbot

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const botName = "chihiro"

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
		Username: botName,
		AsUser:   true,
	}
}
