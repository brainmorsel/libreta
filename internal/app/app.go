package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
)

const (
	bindAddrDefault = "127.0.0.1:8899"
	bindAddrUsage   = "address and port for server"
	dataDirDefault  = "./"
	dataDirUsage    = "path to data directory"
)

type Config struct {
	BindAddr     string
	DevServerURL url.URL
	DataDir      string
}

func Run(ctx context.Context, stdout, stderr io.Writer, args []string, getenv func(string) string) error {
	config := Config{}

	var flagError = &FlagError{}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(&flagError.buf)

	flags.StringVar(&config.BindAddr, "bind", bindAddrDefault, bindAddrUsage)
	flags.StringVar(&config.BindAddr, "b", bindAddrDefault, bindAddrUsage+" (shorthand)")
	flags.StringVar(&config.DataDir, "data-dir", dataDirDefault, dataDirUsage)
	flags.StringVar(&config.DataDir, "d", dataDirDefault, dataDirUsage+" (shorthand)")
	flags.Var(&FlagURLValue{&config.DevServerURL}, "dev-server", "ui dev server url (e.g. http://localhost:5173/)")

	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage of %s [command]:\n", flags.Name())
		fmt.Fprintf(flags.Output(), `
Commands:
  server (default)
        run web server

Flags:
`)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args[1:]); err != nil {
		return flagError
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	switch {
	case flags.Arg(0) == "" || flags.Arg(0) == "server":
		return cmdServer(ctx, logger, &config)
	}
	return nil
}
