package labbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"

	"go.uber.org/zap"

	"strings"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

const botName = "chihiro"

func (l *labbot) msgEvent(rtm *slack.RTM, botID string, event <-chan *slack.MessageEvent) {
	mention := fmt.Sprintf("<@%s>", botID)
	for ev := range event {
		if !strings.Contains(ev.Text, mention) {
			continue
		}
		// 誰がいる?
		if strings.Contains(ev.Text, "誰がい") {
			list := make([]string, 0, len(timeStamp))
			for _, who := range timeStamp {
				if who.Inlab {
					list = append(list, who.Name)
				}
			}
			sort.Strings(list)
			rtm.SendMessage(rtm.NewOutgoingMessage("研究室には"+strings.Join(list, "、")+"がいます！", ev.Channel))
		}

		// よし
		if strings.Contains(ev.Text, "よし") {
			l.sendButtonMessageToSlack(ev.Channel, ev.Text)
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
		l.Warn(`Failed to post to slack`, zap.Error(err), zap.String("message", msg))
		return
	}
	l.Info(
		"Message successfully sent to slack",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}

func (l *labbot) sendButtonMessageToSlack(channelID, msg string) {
	params := joinBtnParam(msg)
	_, timestamp, err := l.PostMessage(channelID, "", params)
	if err != nil {
		l.Warn(`Failed to post button to slack`, zap.Error(err), zap.String("message", msg))
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

const (
	actionJoin    = "参加"
	actionNotJoin = "参加しない"
)

func joinBtnParam(text string) slack.PostMessageParameters {
	attachment := slack.Attachment{
		Text:       text,
		Color:      "#27ae60",
		CallbackID: "participation",
		Actions: []slack.AttachmentAction{
			slack.AttachmentAction{
				Name:  actionJoin,
				Text:  "参加する",
				Type:  "button",
				Value: "join",
			},
			slack.AttachmentAction{
				Name:  actionNotJoin,
				Text:  "参加しない",
				Style: "danger",
				Type:  "button",
				Value: "not join",
			},
		},
	}
	return slack.PostMessageParameters{
		Username:    botName,
		AsUser:      true,
		LinkNames:   1,
		Attachments: []slack.Attachment{attachment},
	}
}

func (l *labbot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		l.Error("Invalid method", zap.String("method", r.Method), zap.String("expected", "POST"))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		l.Error("Failed to read request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	trimmed := string(buf)[8:] // trim `payload=`...
	jsonStr, err := url.QueryUnescape(trimmed)
	if err != nil {
		l.Error("Failed to unespace request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	l.Info("json", zap.String("json", jsonStr))

	var message slack.AttachmentActionCallback
	if err := json.Unmarshal([]byte(jsonStr), &message); err != nil {
		l.Error("Failed to decode json message from slack", zap.String("json", jsonStr))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Only accept message from slack with valid token
	if message.Token != verificationToken {
		l.Error("Invalid token", zap.String("token", message.Token))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	action := message.Actions[0]
	switch action.Name {
	case actionJoin:
		title := "楽しんでくださいねっ！"
		l.responseText(w, message.OriginalMessage, title, "")
	case actionNotJoin:
		title := "残念です…"
		l.responseText(w, message.OriginalMessage, title, "")
	default:
		l.Error("Invalid action was submitted", zap.String("action", action.Name))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (l *labbot) responseText(w http.ResponseWriter, original slack.Message, title, value string) {
	original.Attachments[0].Actions = []slack.AttachmentAction{} // empty buttons
	original.Attachments[0].Fields = []slack.AttachmentField{
		{
			Title: title,
			Value: value,
			Short: false,
		},
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(&original)
	if err != nil {
		l.Error("Failed to write json", zap.Error(err))
	}
}
