package core

import (
	"encoding/gob"
	"io"
	"net"

	"github.com/juju/loggo"
	"golang.org/x/net/websocket"
)

var logger = loggo.GetLogger("core")

type Local struct {
	LogLevel   loggo.Level
	ListenAddr *net.TCPAddr
	URL        string
	Origin     string
}

func (local *Local) Listen() error {
	logger.SetLogLevel(local.LogLevel)

	listener, err := net.ListenTCP("tcp", local.ListenAddr)
	if err != nil {
		return err
	}

	defer listener.Close()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logger.Debugf(err.Error())
			continue
		}

		go local.handleConn(conn)
	}

	return nil
}

func (local *Local) handleConn(conn *net.TCPConn) (err error) {
	defer logger.Debugf("Handle connection error: %s", err.Error())
	defer conn.Close()

	conn.SetLinger(0)

	err = handShake(conn)
	if err != nil {
		return
	}

	_, host, err := getRequest(conn)
	if err != nil {
		return
	}

	logger.Debugf("Host: %s", host)

	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		return
	}

	ws, err := websocket.Dial(local.URL, "", local.Origin)
	if err != nil {
		return
	}

	defer ws.Close()

	enc := gob.NewEncoder(ws)
	req := Request{
		Addr: host,
	}
	err = enc.Encode(req)
	if err != nil {
		return
	}

	go func() {
		_, err = io.Copy(ws, conn)
		if err != nil {
			logger.Debugf(err.Error())
			return
		}
		return
	}()

	_, err = io.Copy(conn, ws)
	if err != nil {
		return
	}

	return
}
