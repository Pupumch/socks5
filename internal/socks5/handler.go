package socks

import (
	"context"
	"net"
)


func (s *Server) handleConn(parentCtx context.Context, conn net.Conn) {
	// ctx, cancel := context.WithCancel(parentCtx)
	// defer cancel()
	
	// // Do something with ctx 
	// ctx.Done() // remove

	log := s.Logger.With().
        Str("client", conn.RemoteAddr().String()).
        Logger()

    authMethod, err := NegotiateMethods(conn); 
    if err != nil {
        log.Error().
            Err(err).
            Str("stage", "negotiate").
            Msg("failed")
		conn.Close()
        return 
    }

    if err = Authenticate(conn, authMethod); err != nil {
        log.Error().
            Err(err).
            Str("stage", "authenticate").
            Msg("failed")
		conn.Close()
        return
    }
    
    remoteConn, err := ProcessRequest(conn)
    if err != nil {
        log.Error().
            Err(err).
            Str("stage", "request").
            Msg("failed")
		conn.Close()
        return
    }

    log.Info().
        Str("stage", "relay").
        Msg("begin")

    Relay(conn, remoteConn)

    log.Info().
        Str("stage", "relay").
        Msg("end")
}


