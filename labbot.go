package labbot

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Code-Hex/exit"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/pkg/errors"
)

type labbot struct {
	Options
	*http.Server
}

func registerHandlers(mux *http.ServeMux) http.Handler {
	mux.HandleFunc("/healthcheck", healthCheck)
	return mux
}

func New() *labbot {
	mux := http.NewServeMux()
	return &labbot{
		Server: &http.Server{
			Handler: registerHandlers(mux),
		},
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
	if err := l.listen(); err != nil {
		return err
	}
	return nil
}

func (l *labbot) prepare() error {
	_, err := parseOptions(&l.Options, os.Args[1:])
	if err != nil {
		return errors.Wrap(err, "Failed to parse command line args")
	}
	return nil
}

func parseOptions(opts *Options, argv []string) ([]string, error) {
	if len(argv) == 0 {
		return nil, exit.MakeUsage(errors.New(string(opts.usage())))
	}

	o, err := opts.parse(argv)
	if err != nil {
		return nil, exit.MakeDataErr(errors.Wrap(err, "Failed to parse command line options"))
	}
	if opts.Help {
		return nil, exit.MakeUsage(errors.New(string(opts.usage())))
	}
	if opts.Version {
		return nil, exit.MakeUsage(errors.New(msg))
	}

	return o, nil
}

func (l *labbot) listen() error {
	var li net.Listener

	if os.Getenv("SERVER_STARTER_PORT") != "" {
		listeners, err := listener.ListenAll()
		if err != nil {
			return errors.Wrap(err, "server-starter error")
		}
		if 0 < len(listeners) {
			li = listeners[0]
		}
	}

	if li == nil {
		var err error
		li, err = net.Listen("tcp", fmt.Sprintf(":%d", l.Port))
		if err != nil {
			return errors.Wrap(err, "listen error")
		}
	}
	return nil
}
