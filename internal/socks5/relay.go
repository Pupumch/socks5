package socks

import (
	"io"
	"net"
	"sync"
)

func Relay(clientConn net.Conn, remoteConn net.Conn) {
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        io.Copy(remoteConn, clientConn)
        remoteConn.Close()
    }()

    go func() {
        defer wg.Done()
        io.Copy(clientConn, remoteConn)
        clientConn.Close()
    }()

    wg.Wait()
}