package labbot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

type timezone int

const (
	Morning timezone = iota
	Daytime
	Night
	MidNight
)

var timeStamp map[string]*time.Time

func init() {
	timeStamp = make(map[string]*time.Time)
}

func (l *labbot) fromBeacon(events []*linebot.Event, r *http.Request) {
	api := slack.New(slackToken)
	channels, err := api.GetChannels(false)
	if err != nil {
		l.Warn("Failed to get slack channnel", zap.Error(err))
		return
	}

	channelID, err := findChannelID(channels, "timestamp")
	if err != nil {
		l.Warn("Failed to find channel id", zap.Error(err))
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			bot, err := linebot.New(channelSecret, channelToken)
			if err != nil {
				l.Error("Failed to construct linebot", zap.Error(err))
				return
			}
			_, err = bot.ReplyMessage(
				event.ReplyToken,
				linebot.NewTextMessage("こんにちは！"),
			).Do()
			if err != nil {
				l.Error("Failed to reply message", zap.Error(err))
				return
			}
		}
		if event.Type == linebot.EventTypeBeacon {
			src := event.Source
			userID := src.UserID
			bot, err := linebot.New(channelSecret, channelToken)
			if err != nil {
				l.Error("Failed to construct linebot", zap.Error(err))
				return
			}
			res, err := bot.GetProfile(userID).Do()
			if err != nil {
				l.Error("Failed to get user profile", zap.Error(err))
				return
			}

			switch event.Beacon.Type {
			case linebot.BeaconEventTypeEnter:
				// When already in the laboratory
				if isAlready(res.DisplayName) {
					return
				}
				_, err := bot.ReplyMessage(
					event.ReplyToken,
					linebot.NewTextMessage(fmt.Sprintf("%sさん%s♡", res.DisplayName, greeting())),
				).Do()
				if err != nil {
					l.Error("Failed to reply message", zap.Error(err))
					return
				}
				l.welcomeToLab(api, res.DisplayName, channelID)
			case linebot.BeaconEventTypeLeave:
				_, err := bot.ReplyMessage(
					event.ReplyToken,
					linebot.NewTextMessage(fmt.Sprintf("%sさん、%s", res.DisplayName, getMessageWorkingTime(res.DisplayName))),
				).Do()
				if err != nil {
					l.Error("Failed to reply message", zap.Error(err))
					return
				}
				l.seeyouFromLab(api, res.DisplayName, channelID)
			}
		}
	}
}

func (l *labbot) welcomeToLab(api *slack.Client, name, channelID string) {
	now := time.Now()
	timeStamp[name] = &now
	formatted := now.Format("2006年01月02日 15時04分")
	msg := fmt.Sprintf("%sさんが%sに来ました♡", name, formatted)
	params := parameter()
	attachment := slack.Attachment{
		Color: "#e67e22",
		Text:  msg,
	}
	params.Attachments = []slack.Attachment{attachment}
	_, timestamp, err := api.PostMessage(channelID, "", params)
	if err != nil {
		l.Warn(`Failed to post "welcome message" to slack`, zap.Error(err))
		return
	}
	l.Info(
		"Message successfully sent to slack",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}

func (l *labbot) seeyouFromLab(api *slack.Client, name, channelID string) {
	now := time.Now()
	formatted := now.Format("2006年01月02日 15時04分")
	msg := fmt.Sprintf("%sさんが%sに帰りました♡", name, formatted)
	params := parameter()
	attachment := slack.Attachment{
		Color: "#3498db",
		Text:  msg,
	}
	params.Attachments = []slack.Attachment{attachment}
	_, timestamp, err := api.PostMessage(channelID, "", params)
	if err != nil {
		l.Warn(`Failed to post "seeyou message" to slack`, zap.Error(err))
		return
	}
	l.Info(
		"Message successfully sent to slack",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}

func isAlready(name string) bool {
	coming, ok := timeStamp[name]
	return ok && coming != nil
}

func findChannelID(channels []slack.Channel, name string) (string, error) {
	for _, channel := range channels {
		if channel.Name == name {
			return channel.ID, nil
		}
	}
	return "", fmt.Errorf("Could not find ChannelID of #%s", name)
}

func parameter() slack.PostMessageParameters {
	return slack.PostMessageParameters{
		Username: "kirari",
		AsUser:   true,
	}
}

func greeting() string {
	switch getTimeZone() {
	case Morning:
		return "おはようございます"
	case Daytime:
		return "こんにちは"
	case Night:
		return "こんばんは"
	}
	return "夜遅くまでお疲れ様です"
}

func getTimeZone() timezone {
	hour := time.Now().Hour()

	if 11 <= hour && hour < 17 {
		return Daytime
	}
	if 17 <= hour && hour < 23 {
		return Night
	}
	if 5 <= hour && hour < 11 {
		return Morning
	}

	return MidNight
}

func getMessageWorkingTime(name string) string {
	came := timeStamp[name]
	sub := int(time.Now().Sub(*came).Hours())

	if 0 <= sub && sub <= 4 {
		return "お疲れ様です！"
	}

	if 4 <= sub && sub <= 8 {
		return "とっても頑張ったね！偉い！"
	}

	return "お願い！死なないでね！"
}

// "/whoisthere" handler
func whoIsThere(w http.ResponseWriter, r *http.Request) {
	list := make([]string, 0, len(timeStamp))
	for who := range timeStamp {
		list = append(list, who)
	}
	sort.Slice(list, func(i, j int) bool {
		for x := 0; x < len(list[i]); x++ {
			if list[i][x] != list[j][x] {
				return list[i][x] < list[j][x]
			}
		}
		return len(list[i]) < len(list[j])
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		People []string `json:"people"`
	}{
		People: list,
	})
}
