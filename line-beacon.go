package labbot

import (
	"fmt"
	"log"
	"net/http"

	"github.com/line/line-bot-sdk-go/linebot"
)

func fromBeacon(events []*linebot.Event, r *http.Request) {
	for _, event := range events {
		if event.Type == linebot.EventTypeBeacon {
			src := event.Source
			userID := src.UserID
			bot, err := linebot.New(channelSecret, channelToken)
			if err != nil {
				log.Printf("Error: %s", err.Error())
			}
			res, err := bot.GetProfile(userID).Do()
			if err != nil {
				log.Printf("Error: %s", err.Error())
			}
			fmt.Println(res.DisplayName)
		}
	}
}
