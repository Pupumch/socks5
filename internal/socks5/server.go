package socks

import (
	"context"
	"net"

	"github.com/rs/zerolog"
)


type Server struct {
	Addr string
	Logger zerolog.Logger
}

func New(addr string, logger zerolog.Logger) *Server {
	return &Server{
		Addr: addr,
		Logger: logger,
	}
}

func (s *Server) Serve(ctx context.Context, l net.Listener) error {

    go func() {
        <-ctx.Done()
        s.Logger.Info().Msg("shutdown: closing listener")
        l.Close()
    }()

	for {
        conn, err := l.Accept()
        if err != nil {
			select {
            case <-ctx.Done():
				return nil
			default:
				s.Logger.Error().Err(err).Msg("accept error")
				continue
			}
        }

		go s.handleConn(ctx, conn)
    }
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	return s.Serve(ctx, l)
}