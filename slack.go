package labbot

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const botName = "chihiro"

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
