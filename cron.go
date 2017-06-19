package labbot

func (l *labbot) isThereProgress() {
	l.sendToSlack("general", "<!here> 進捗どうですか！？")
}

func (l *labbot) noticeSeminar() {
	l.sendToSlack("tamaki", "<!channel> みなさんっ！今日はゼミの日ですよ！")
}

func (l *labbot) noticeDayAfterTomorrow() {
	l.sendToSlack("tamaki", "<!channel> みなさんっ！明後日はゼミの日ですよ！")
}
