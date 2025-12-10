package socks

import (
	"fmt"
	"io"
	"net"
	"encoding/binary"
	"strconv"
	"errors"
	"os"
	"strings"
	"context"
)


const (
	connectMethod = byte(0x01)
	atypIpv4 = byte(0x01)
	atypDomainname = byte(0x03)
	atypIpv6 = byte(0x04)

	repSuccess              = byte(0x00)
    repGeneralFailure       = byte(0x01)
    repConnectionNotAllowed = byte(0x02)
    repNetworkUnreachable   = byte(0x03)
    repHostUnreachable      = byte(0x04)
    repConnectionRefused    = byte(0x05)
    repTTLExpired           = byte(0x06)
    // repCommandNotSupported  = 0x07
    // repAddressNotSupported  = 0x08
)

type SocksRequest struct {
	Atyp byte
	Address net.IP
	Domain string
	Port uint16
}

type SocksReply struct {
	Ver byte
	Rep byte
	Rsv byte
	Atyp byte
	Address net.IP
	Port uint16
}

type BoundAddr struct {
	Atyp byte
	Address net.IP
	Port uint16
}

func ProcessRequest(conn net.Conn) (net.Conn, error) {
	req, err := parseRequest(conn)
	if err != nil {
		return nil, err
	}

	remoteConn, err := handleConnect(req)
	if err != nil {
		rep := mapDialErrorToRep(err)
		sendReply(conn, rep, nil)
		return nil, err
	}

	bnd, err := extractBoundAddr(remoteConn)
	if err != nil {
		remoteConn.Close()
		rep := repGeneralFailure
		sendReply(conn, rep, nil)
		return nil, err
	}
	if err := sendReply(conn, repSuccess, bnd); err != nil {
		remoteConn.Close()
		return nil, err
	}

	return remoteConn, nil
}

func extractBoundAddr(remoteConn net.Conn) (*BoundAddr, error) {
	tcpAddr, ok := remoteConn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("local addr is not TCP")
	}

	ip := tcpAddr.IP
	var atyp byte
	var address net.IP

	if ipv4 := ip.To4(); ipv4 != nil {
		atyp = atypIpv4
		address = ipv4
	} else if ipv6 := ip.To16(); ipv6 != nil {
		atyp = atypIpv6
		address = ipv6
	}  else {
    	return nil, fmt.Errorf("invalid IP format")
    }
	
	return &BoundAddr{
		Atyp: atyp,
		Address: address,
		Port: uint16(tcpAddr.Port),
	}, nil
}

func mapDialErrorToRep(err error) byte {
    if err == nil {
        return repSuccess
    }

    // Handle timeouts
    if errors.Is(err, os.ErrDeadlineExceeded) ||
       errors.Is(err, context.DeadlineExceeded) {
        return repTTLExpired
    }

    // Network errors
    var netErr net.Error
    if errors.As(err, &netErr) {
        if strings.Contains(err.Error(), "network is unreachable") {
            return repNetworkUnreachable
        }

        if strings.Contains(err.Error(), "no route to host") {
            return repHostUnreachable
        }

        if strings.Contains(err.Error(), "connection refused") {
            return repConnectionRefused
        }
    }

    // Permission errors ⇒ ruleset-like rejection
    if errors.Is(err, os.ErrPermission) {
        return repConnectionNotAllowed
    }

    // DNS resolution failure
    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return repHostUnreachable
    }

    return repGeneralFailure
}

func handleConnect(req *SocksRequest) (net.Conn, error) {
	var host string

	switch req.Atyp {
	case atypDomainname:
		host = req.Domain
	case atypIpv4, atypIpv6:
		host = req.Address.String()
	default:
		return nil, fmt.Errorf("unsupported ATYP: %d", req.Atyp)
	}

	addr := net.JoinHostPort(host, strconv.Itoa(int(req.Port)))
	return net.Dial("tcp", addr)
}

func sendReply(conn net.Conn, rep byte, bnd *BoundAddr) error {
	reply := []byte{socksVersion, rep, 0x00}

	if bnd == nil {
		reply = append(reply, atypIpv4)
		reply = append(reply, []byte{0,0,0,0}...)
		reply = append(reply, []byte{0,0}...)
		_, err := conn.Write(reply)
		return err
	}

	switch bnd.Atyp {
	case atypIpv4:
		reply = append(reply, atypIpv4)
		reply = append(reply, bnd.Address.To4()...)
	case atypIpv6:
		reply = append(reply, atypIpv6)
		reply = append(reply, bnd.Address.To16()...)

	default:
		return fmt.Errorf("")
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, bnd.Port)
	reply = append(reply, portBuf...)

	_, err := conn.Write(reply)

	return err
}

func parseRequest(conn net.Conn) (*SocksRequest, error) {
	buf := make([]byte, 4)

	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	ver := buf[0]
	if ver != socksVersion {
		return nil, fmt.Errorf("invalid SOCKS version: got %d, expected %d", ver, socksVersion)
	}

	cmd := buf[1]
	if cmd != connectMethod {
		return nil, fmt.Errorf("unsupported command: got %d, only CONNECT (%d) is supported", cmd, connectMethod)
	}

	rsv := buf[2]
	if rsv != 0x00 {
		return nil, fmt.Errorf("invalid reserved byte: got 0x%02x, expected 0x00", rsv)
	}

	var addr []byte
	var addrLen []byte
	var domain []byte
	atyp := buf[3]
	switch atyp {
	case atypIpv4:
		addr = make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
	case atypIpv6:
		addr = make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
	case atypDomainname:
		addrLen = make([]byte, 1)
		if _, err := io.ReadFull(conn, addrLen); err != nil {
			return nil, err
		}
		if addrLen[0] == 0 {
			return nil, fmt.Errorf("domain length is zero")
		}
		domain = make([]byte, addrLen[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("invalid address type: got 0x%02x", atyp)
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return nil, err
	}

	return &SocksRequest{
		Atyp: atyp,
		Address: net.IP(addr),
		Domain: string(domain),
		Port: binary.BigEndian.Uint16(portBytes),
	}, nil
}