package socks

import (
	"slices"
	"fmt"
	"net"
    "io"
)

const (
    socksVersion = byte(0x05)
    methodNone   = byte(0x00)
    methodNoAccept = byte(0xFF)
)

func NegotiateMethods(conn net.Conn) (byte, error) { // perhaps return accepted method also ?
    // SOCKS HANDSHAKE
    /*
    1. read client greeting (socks version, number of auth methods, list of methods supported by the client)
    2. Choose a method (you choose “no authentication” for now)
    3. Send back a reply saying: Version = 5; Method = no-auth
    */

    buf := make([]byte, 2)
    if _, err := io.ReadFull(conn, buf); err != nil {
        return 0, err
    }

    ver := buf[0]
    nmethods := buf[1]

    if ver != socksVersion {
        return 0, fmt.Errorf("invalid SOCKS version %d", ver)
    }

    if nmethods == 0 {
        return 0, fmt.Errorf("client offered no authentication methods")
    }

    methods := make([]byte, nmethods)
    if _, err := io.ReadFull(conn, methods); err != nil {
        return 0, err
    }

    // only noauth is supported for now
    res := methodNoAccept
    if slices.Contains(methods, methodNone) {
		res = methodNone
	}

    // response
    n, err := conn.Write([]byte{socksVersion, res})
    if err != nil {
        return 0, fmt.Errorf("failed to write handshake response: %w", err)
    }
    if n != 2 {
        return 0, fmt.Errorf("partial write in handshake: wrote %d bytes", n)
    }
    if res == methodNoAccept {
        return 0, fmt.Errorf("no acceptable authentication methods")
    }
    return res, nil
}
