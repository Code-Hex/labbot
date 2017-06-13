package labbot

import "go.uber.org/zap"

func (l *labbot) isThereProgress() {
	// Find the slack channel
	channelID, err := l.findChannelID("tamaki")
	if err != nil {
		l.Warn("Failed to find channel id", zap.Error(err))
		return
	}
	params := parameter()
	_, timestamp, err := l.PostMessage(channelID, "<!here> 進捗はありますか？", params)
	if err != nil {
		l.Warn(`Failed to post "Is there progress?" to slack`, zap.Error(err))
		return
	}
	l.Info(
		"Message successfully sent to slack",
		zap.String("channelID", channelID),
		zap.String("timestamp", timestamp),
	)
}
