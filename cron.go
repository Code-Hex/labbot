package labbot

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (l *labbot) isThereProgress() {
	l.sendToSlack("general", "<!here> みなさん、進捗どうですか!?")
}

func (l *labbot) noticeSeminar() {
	l.sendToSlack("tamaki", "<!channel> みなさん、今日はｾﾞﾐの日ですよ!\n私も応援してますからね!")
}

func (l *labbot) noticeDayAfterTomorrow() {
	l.sendToSlack("tamaki", "<!channel> 明後日はｾﾞﾐの日ですよ!")
}

var messages = []string{
	"机の上にあるｺﾞﾐはｺﾞﾐ箱に入れましょう!",
	"たまには掃除機を使って床を掃除してあげてくださいっ!",
	"ｾﾞﾐの後は綺麗な空間でゆっくり休んで欲しいです。",
	"たまには机の上も拭きましょうねっ!",
}

func (l *labbot) noticeClean() {
	msg := messages[rand.Intn(len(messages))]
	l.sendToSlack("general", "<!channel> みなさんっ！掃除はしてますか？\n"+msg)
}
