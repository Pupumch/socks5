package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

    socks "astrolabe/internal/socks5"
)


func main() {
    logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
        With().
        Timestamp().
        Str("component", "socks-server").
        Logger()

    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT,
        syscall.SIGTERM,
    )
    defer stop()

    logger.Info().
        Str("addr", ":1080").
        Msg("starting SOCKS server")

    s := socks.New(":1080", logger)

    go func() {
        <-ctx.Done()
        logger.Info().Msg("shutdown signal received")
    }()

    if err := s.ListenAndServe(ctx); err != nil {
        log.Fatal().Err(err).Msg("server error")
        os.Exit(1)
    }
}

