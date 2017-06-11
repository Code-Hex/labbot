package labbot

import (
	"fmt"
	"net/http"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

func (l *labbot) fromBeacon(events []*linebot.Event, r *http.Request) {
	api := slack.New(slackToken)
	channels, err := api.GetChannels(false)
	if err != nil {
		l.Warn("Failed to get slack channnel", zap.Error(err))
		return
	}

	var channelID string
	for _, channel := range channels {
		if channel.Name == "tamaki" {
			channelID = channel.ID
			break
		}
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeBeacon {
			src := event.Source
			userID := src.UserID
			bot, err := linebot.New(channelSecret, channelToken)
			if err != nil {
				l.Error("Failed to construct linebot", zap.Error(err))
			}
			res, err := bot.GetProfile(userID).Do()
			if err != nil {
				l.Error("Failed to get user profile", zap.Error(err))
			}

			switch event.Beacon.Type {
			case linebot.BeaconEventTypeEnter:
				l.welcomeToLab(api, res.DisplayName, channelID)
			case linebot.BeaconEventTypeLeave:
				l.seeyouFromLab(api, res.DisplayName, channelID)
			}
		}
	}
}

func (l *labbot) welcomeToLab(api *slack.Client, name, channelID string) {
	params := slack.PostMessageParameters{}
	attachment := slack.Attachment{
		Color:   "#e74c3c",
		Pretext: "some pretext",
		Text:    "some text",
		// Uncomment the following part to send a field too
		/*
			Fields: []slack.AttachmentField{
				slack.AttachmentField{
					Title: "a",
					Value: "no",
				},
			},
		*/
	}
	params.Attachments = []slack.Attachment{attachment}
	msg := fmt.Sprintf("こんにちは!!, %sさん♡", name)
	_, timestamp, err := api.PostMessage(channelID, msg, params)
	if err != nil {
		l.Warn(`Failed to post "welcome message" to slack`, zap.Error(err))
		return
	}
	l.Info(
		"Message successfully sent to channel",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}

func (l *labbot) seeyouFromLab(api *slack.Client, name, channelID string) {
	params := slack.PostMessageParameters{}
	attachment := slack.Attachment{
		Color:   "#e74c3c",
		Pretext: "some pretext",
		Text:    "some text",
		// Uncomment the following part to send a field too
		/*
			Fields: []slack.AttachmentField{
				slack.AttachmentField{
					Title: "a",
					Value: "no",
				},
			},
		*/
	}
	params.Attachments = []slack.Attachment{attachment}
	msg := fmt.Sprintf("また来てくださいね!!, %sさん♡", name)
	_, timestamp, err := api.PostMessage(channelID, msg, params)
	if err != nil {
		l.Warn(`Failed to post "seeyou message" to slack`, zap.Error(err))
		return
	}
	l.Info(
		"Message successfully sent to channel",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}
