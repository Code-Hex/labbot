package labbot

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"syscall"

	"github.com/Code-Hex/exit"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/line/line-bot-sdk-go/linebot/httphandler"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

const (
	version = "0.0.1"
	msg     = "LabBot v" + version + ", Bot for tamaki lab\n"
)

var (
	slackToken    = os.Getenv("SLACK_TOKEN")
	channelSecret = os.Getenv("CHANNEL_SECRET")
	channelToken  = os.Getenv("CHANNEL_TOKEN")
)

type labbot struct {
	Options
	*http.Server
	*zap.Logger
	*cron.Cron
	*slack.Client
	waitSignal chan os.Signal
}

func (l *labbot) registerHandlers() (http.Handler, error) {
	mux := http.NewServeMux()

	// Normal
	mux.HandleFunc("/healthcheck", healthCheck) // healthcheck.go
	mux.HandleFunc("/whoisthere", whoIsThere)   // line-beacon.go

	// LINE Webhook
	webhook, err := httphandler.New(channelSecret, channelToken)
	if err != nil {
		return nil, err
	}
	webhook.HandleEvents(l.fromBeacon)
	webhook.HandleError(func(err error, r *http.Request) {
		l.Warn("LINEBot handler error", zap.Error(err))
	})
	mux.HandleFunc("/line", webhook.ServeHTTP)

	return mux, nil
}

func New() *labbot {
	sigch := make(chan os.Signal)
	signal.Notify(
		sigch,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	return &labbot{
		Server:     new(http.Server),
		Cron:       cron.New(),
		Client:     slack.New(slackToken),
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
	handler, err := l.registerHandlers()
	if err != nil {
		return errors.Wrap(err, "Failed to register http handlers")
	}
	l.Handler = handler

	logger, err := setupLogger(
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		errors.Wrap(err, "Failed to construct zap")
	}
	l.Logger = logger

	return nil
}

func (l *labbot) registerCronHandlers() {
	// You should to see cron.go
	l.AddFunc("0 0 18 * * *", l.isThereProgress)
}

func setupLogger(opts ...zap.Option) (*zap.Logger, error) {
	if os.Getenv("STAGE") == "production" {
		logger, err := zap.NewProduction(opts...)
		if err != nil {
			return nil, err
		}
		return logger, nil
	}
	logger, err := zap.NewDevelopment(opts...)
	if err != nil {
		return nil, err
	}
	return logger, nil
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
	go func() {
		if err := l.Serve(li); err != nil {
			l.Warn("Server is stopped", zap.Error(err))
		}
	}()
	<-l.waitSignal

	return l.Shutdown(context.Background())
}
