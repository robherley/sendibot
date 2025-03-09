package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/lmittmann/tint"
	"github.com/robherley/sendibot/internal/bot"
	"github.com/robherley/sendibot/internal/db"
	"github.com/robherley/sendibot/internal/looper"
	"github.com/robherley/sendibot/pkg/sendico"
)

type Config struct {
	DiscordToken string `desc:"API Token for Discord" required:"true"`
	DatabaseFile string `desc:"Path of SQLite database file" default:"sendibot.db" required:"false"`
}

func init() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
}

func main() {
	if err := run(); err != nil {
		slog.Error("sendibot failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{}

	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "Usage: sendibot [flags]\n\n")
		envconfig.Usagef("", &cfg, flag.CommandLine.Output(), envconfig.DefaultListFormat)
		fmt.Fprintln(flag.CommandLine.Output(), "\nFlags:")
		flag.PrintDefaults()
	}

	register := flag.String("register", "", "guild to register commands (or 'global')")
	unregister := flag.String("unregister", "", "guild to unregister commands (or 'global')")
	flag.Parse()

	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	db, err := db.NewSQLite(cfg.DatabaseFile)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		return err
	}

	sendico, err := sendico.New(ctx)
	if err != nil {
		return err
	}

	bot, err := bot.New(cfg.DiscordToken, db, sendico)
	if err != nil {
		return err
	}
	defer bot.Close()

	if err := bot.Start(); err != nil {
		return err
	}

	exitEarly := false

	if *register != "" {
		if err := bot.Register(*register); err != nil {
			return err
		}
		exitEarly = true
	}

	if *unregister != "" {
		if err := bot.Unregister(*unregister); err != nil {
			return err
		}
		exitEarly = true
	}

	if exitEarly {
		return nil
	}

	slog.Info("sendibot is initialized")

	l := looper.New(db, sendico, bot)
	go l.Notify(ctx)
	go l.Cleanup(ctx)
	go l.Refresh(ctx)

	wait()
	return nil
}

func wait() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-done
	slog.Warn("received signal, shutting down", "signal", sig.String())
}
