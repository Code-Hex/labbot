package labbot

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"syscall"

	"path/filepath"

	"github.com/Code-Hex/exit"
	"github.com/go-redis/redis"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/line/line-bot-sdk-go/linebot/httphandler"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	version = "0.0.2"
	msg     = "LabBot v" + version + ", Bot for tamaki lab\n"
)

var (
	slackToken        = os.Getenv("SLACK_TOKEN")
	channelSecret     = os.Getenv("CHANNEL_SECRET")
	channelToken      = os.Getenv("CHANNEL_TOKEN")
	verificationToken = os.Getenv("VERIFICATION_TOKEN")
)

type labbot struct {
	Options
	*http.Server
	*zap.Logger
	*cron.Cron
	*slack.Client
	Redis      *redis.Client
	waitSignal chan os.Signal
}

func (l *labbot) registerHandlers() (http.Handler, error) {
	mux := http.NewServeMux()

	// Normal
	mux.HandleFunc("/healthcheck", healthCheck) // healthcheck.go
	mux.HandleFunc("/whoisthere", whoIsThere)   // line-beacon.go

	// slack webhook
	mux.HandleFunc("/slack_participate", l.ServeHTTP)

	// LINE Webhook
	webhook, err := httphandler.New(channelSecret, channelToken)
	if err != nil {
		return nil, exit.MakeSoftWare(err)
	}
	webhook.HandleEvents(l.lineAPIInit())
	webhook.HandleError(func(err error, r *http.Request) {
		l.Warn("LINEBot handler error", zap.Error(err))
	})
	mux.HandleFunc("/line", webhook.ServeHTTP)

	// Static files
	dir := "public"
	ok, err := exists(dir)
	if err != nil {
		return nil, exit.MakeUnAvailable(err)
	}
	if !ok {
		os.Mkdir(dir, os.ModeDir|os.ModePerm)
	}
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", fs)

	return mux, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func New() *labbot {
	sigch := make(chan os.Signal)
	signal.Notify(
		sigch,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	return &labbot{
		Server: new(http.Server),
		Cron:   cron.New(),
		Client: slack.New(slackToken),
		Redis: redis.NewClient(&redis.Options{
			Addr: "127.0.0.1:6379",
		}),
		waitSignal: sigch,
	}
}

func (l *labbot) Run() int {
	if e := l.run(); e != nil {
		exitCode, err := UnwrapErrors(e)
		if l.StackTrace {
			fmt.Fprintf(os.Stderr, "Error:\n  %+v\n", e)
		} else {
			fmt.Fprintf(os.Stderr, "Error:\n  %v\n", err)
		}
		return exitCode
	}
	return 0
}

func (l *labbot) run() error {
	if err := l.prepare(); err != nil {
		return err
	}
	li, err := l.listen()
	if err != nil {
		return err
	}
	return l.serve(li)
}

func (l *labbot) prepare() error {
	_, err := parseOptions(&l.Options, os.Args[1:])
	if err != nil {
		return errors.Wrap(err, "Failed to parse command line args")
	}

	logger, err := setupLogger(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		errors.Wrap(err, "Failed to construct zap")
	}
	l.Logger = logger

	handler, err := l.registerHandlers()
	if err != nil {
		return errors.Wrap(err, "Failed to register http handlers")
	}
	l.Handler = handler
	l.registerCronHandlers()

	return nil
}

func (l *labbot) registerCronHandlers() {
	l.Info("register cron")
	// Please check cron.go
	l.AddFunc("0 30 18 * * *", l.isThereProgress)
	l.AddFunc("0 0 10 * * 5", l.noticeSeminar)
	l.AddFunc("0 0 15 * * 1,3,5", l.noticeClean)
	l.AddFunc("0 0 17 * * 3", l.noticeDayAfterTomorrow)

	l.Start() // start cron job
}

func setupLogger(opts ...zap.Option) (*zap.Logger, error) {
	config := genLoggerConfig()
	enc := zapcore.NewJSONEncoder(config.EncoderConfig)

	dir := "log"
	ok, err := exists(dir)
	if err != nil {
		return nil, exit.MakeUnAvailable(err)
	}
	if !ok {
		os.Mkdir(dir, os.ModeDir|os.ModePerm)
	}
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, exit.MakeUnAvailable(err)
	}
	logf, err := rotatelogs.New(
		filepath.Join(absPath, "labbot_log.%Y%m%d%H%M"),
		rotatelogs.WithLinkName(filepath.Join(absPath, "labbot_log")),
		rotatelogs.WithMaxAge(24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
	if err != nil {
		return nil, exit.MakeUnAvailable(err)
	}
	core := zapcore.NewCore(enc, zapcore.AddSync(logf), config.Level)

	return zap.New(core, opts...), nil
}

func genLoggerConfig() zap.Config {
	if os.Getenv("STAGE") == "production" {
		return zap.NewProductionConfig()
	}
	return zap.NewDevelopmentConfig()
}

func parseOptions(opts *Options, argv []string) ([]string, error) {
	o, err := opts.parse(argv)
	if err != nil {
		return nil, exit.MakeDataErr(err)
	}
	if opts.Help {
		return nil, exit.MakeUsage(errors.New(string(opts.usage())))
	}
	if opts.Version {
		return nil, exit.MakeUsage(errors.New(msg))
	}

	return o, nil
}

func (l *labbot) listen() (net.Listener, error) {
	var li net.Listener

	if os.Getenv("SERVER_STARTER_PORT") != "" {
		listeners, err := listener.ListenAll()
		if err != nil {
			return nil, errors.Wrap(err, "server-starter error")
		}
		if 0 < len(listeners) {
			li = listeners[0]
		}
	}

	if li == nil {
		var err error
		li, err = net.Listen("tcp", fmt.Sprintf(":%d", l.Port))
		if err != nil {
			return nil, errors.Wrap(err, "listen error")
		}
	}
	return li, nil
}

func (l *labbot) serve(li net.Listener) error {
	go l.rtmRun()
	go func() {
		if err := l.Serve(li); err != nil {
			l.Warn("Server is stopped", zap.Error(err))
		}
	}()
	return l.shutdown()
}

func (l *labbot) rtmRun() {
	botID, err := l.findUserID(botName)
	if err != nil {
		l.Error("Could not to get the bot id", zap.Error(err))
	}

	rtm := l.NewRTM()
	go rtm.ManageConnection()

	reply := make(chan *slack.MessageEvent)
	defer close(reply)

	go l.msgEvent(rtm, botID, reply)

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			reply <- ev
		case *slack.RTMError:
			l.Error("slack rtm error", zap.String("Error", ev.Error()))
		case *slack.InvalidAuthEvent:
			l.Error("Invalid credentials")
			return
		}
	}
}

func (l *labbot) shutdown() error {
	<-l.waitSignal
	l.Stop() // stop cron job
	return l.Shutdown(context.Background())
}
