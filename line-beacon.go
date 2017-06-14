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

type jsonTime time.Time

func (t jsonTime) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format("2006年01月02日 15時04分"))
	return []byte(stamp), nil
}

type Person struct {
	Name       string   `json:"name"`
	Inlab      bool     `json:"in_lab"`
	UpdateTime jsonTime `json:"updated_at"`
}

var timeStamp map[string]*Person

func init() {
	timeStamp = make(map[string]*Person)
}

func (l *labbot) fromBeacon(events []*linebot.Event, r *http.Request) {
	// Find the slack channel
	channelID, err := l.findChannelID("timestamp")
	if err != nil {
		l.Warn("Failed to find channel id", zap.Error(err))
		return
	}

	for _, event := range events {
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
				l.welcomeToLab(res.DisplayName, channelID)
			case linebot.BeaconEventTypeLeave:
				_, err := bot.ReplyMessage(
					event.ReplyToken,
					linebot.NewTextMessage(fmt.Sprintf("%sさん、%s", res.DisplayName, getMessageWorkingTime(res.DisplayName))),
				).Do()
				if err != nil {
					l.Error("Failed to reply message", zap.Error(err))
					return
				}
				l.seeyouFromLab(res.DisplayName, channelID)
			}
		}
	}
}

func (l *labbot) welcomeToLab(name, channelID string) {
	now := time.Now()
	setCameTimeStamp(name, now)
	formatted := now.Format("2006年01月02日 15時04分")

	msg := fmt.Sprintf("%sさんが%sに来ました♡", name, formatted)
	params := parameter()
	attachment := slack.Attachment{
		Color: "#e67e22",
		Text:  msg,
	}
	params.Attachments = []slack.Attachment{attachment}
	_, timestamp, err := l.PostMessage(channelID, "", params)
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

func (l *labbot) seeyouFromLab(name, channelID string) {
	now := time.Now()
	setLeaveTimeStamp(name, now)
	formatted := now.Format("2006年01月02日 15時04分")

	msg := fmt.Sprintf("%sさんが%sに帰りました♡", name, formatted)
	params := parameter()
	attachment := slack.Attachment{
		Color: "#3498db",
		Text:  msg,
	}
	params.Attachments = []slack.Attachment{attachment}
	_, timestamp, err := l.PostMessage(channelID, "", params)
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

func setCameTimeStamp(name string, now time.Time) {
	setTimeStamp(name, now, true)
}

func setLeaveTimeStamp(name string, now time.Time) {
	setTimeStamp(name, now, false)
}

func setTimeStamp(name string, now time.Time, inlab bool) {
	_, ok := timeStamp[name]
	if !ok {
		timeStamp[name] = &Person{
			Name:       name,
			Inlab:      inlab,
			UpdateTime: jsonTime(now),
		}
	} else {
		timeStamp[name].Inlab = inlab
		timeStamp[name].UpdateTime = jsonTime(now)
	}
}

func isAlready(name string) bool {
	coming, ok := timeStamp[name]
	if ok {
		return coming.Inlab
	}
	return false
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
	t := time.Time(came.UpdateTime)
	sub := int(time.Now().Sub(t).Hours())

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
	list := make([]*Person, 0, len(timeStamp))
	for _, who := range timeStamp {
		list = append(list, who)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		People []*Person `json:"people"`
	}{
		People: list,
	})
}
