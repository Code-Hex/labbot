package labbot

import (
	"bytes"
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
)

// Options struct for parse command line arguments
type Options struct {
	Help       bool `short:"h" long:"help"`
	Version    bool `short:"v" long:"version"`
	Port       int  `short:"p" long:"port" default:"8080"`
	StackTrace bool `long:"trace"`
}

func (opts *Options) parse(argv []string) ([]string, error) {
	p := flags.NewParser(opts, flags.PrintErrors)
	args, err := p.ParseArgs(argv)
	if err != nil {
		os.Stderr.Write(opts.usage())
		return nil, errors.Wrap(err, "invalid command line options")
	}

	return args, nil
}

func (opts Options) usage() []byte {
	buf := bytes.Buffer{}

	fmt.Fprintf(&buf, msg+
		`Usage: labbot [options]
  Options:
  -h,  --help                print usage and exit
  -v,  --version             display the version of labbot and exit
  -p,  --port <num>          port number to run server
  --trace                    display detail error messages
`)
	return buf.Bytes()
}
