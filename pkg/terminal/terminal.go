package terminal

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"gaiasec-nodeagent/pkg/config"
	"gaiasec-nodeagent/pkg/util"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/runletapp/go-console"
	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

type Terminal struct {
	Cols      int
	Rows      int
	Shell     string
	Command   string
	SessionID string
}

func NewTerminal(cols int, rows int, cmd string, sessionID string) *Terminal {

	shell := "C:\\Windows\\System32\\cmd.exe"
	shells := []string{"/bin/zsh", "/bin/bash", "/bin/dash", "/bin/sh"}
	for _, s := range shells {
		if util.Exists(s) {
			shell = s
			break
		}
	}

	return &Terminal{
		Cols:      cols,
		Rows:      rows,
		Shell:     shell,
		Command:   cmd,
		SessionID: sessionID,
	}
}

func parseIncomingMessage(messageType websocket.MessageType, message []byte) ([]byte, *TTYSize, bool, error) {
	dataBuffer := bytes.Trim(message, "\x00")
	if len(dataBuffer) == 0 {
		return nil, nil, false, nil
	}

	if messageType == websocket.MessageBinary || dataBuffer[0] != 1 {
		return dataBuffer, nil, false, nil
	}

	ttySize := &TTYSize{}
	resizeMessage := bytes.Trim(dataBuffer[1:], "\n\r\t\x00\x01")
	if err := json.Unmarshal(resizeMessage, ttySize); err != nil {
		return nil, nil, true, err
	}

	return nil, ttySize, true, nil
}

func (t *Terminal) Start() {
	proc, err := console.New(t.Cols, t.Rows)
	if err != nil {
		panic(err)
	}
	defer proc.Close()

	log.Infof("Creating new terminal [%s][%s]", t.SessionID, t.Shell)
	var args []string
	if runtime.GOOS == "windows" {
		args = []string{t.Shell}
	} else {
		args = []string{t.Shell, "-i"}
	}
	err = proc.SetENV(os.Environ())
	if err != nil {
		return
	}
	err = proc.Start(args)
	if err != nil {
		log.Infof("failed to start tty: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionID := strings.Replace(t.SessionID, "client-", "server-", -1)
	cfg := config.GetInstance()
	protocol, host := util.ParseServerURL(cfg.Server)
	wsProtocol := util.GetWebSocketProtocol(protocol)
	uri := wsProtocol + "://" + host + "/ws/terminal?node_id=" + cfg.NodeID + "&client_id=" + sessionID
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	opts := &websocket.DialOptions{
		HTTPClient: httpClient,
	}
	conn, _, err := websocket.Dial(ctx, uri, opts)
	if err != nil {
		log.Infof("failed to connect to terminal: %v", err)
		return
	}
	conn.SetReadLimit(1 << 25)

	defer func() {
		log.Infof("gracefully stopping spawned tty...")
		if err := proc.Kill(); err != nil {
			log.Infof("failed to kill tty: %v", err)
		}
		if _, err := proc.Wait(); err != nil {
			log.Infof("failed to wait for tty: %v", err)
		}
		if err := proc.Close(); err != nil {
			log.Infof("failed to close tty: %v", err)
		}
		if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
			log.Infof("failed to close conn: %v", err)
		}
	}()

	var connectionErrorLimit = 10
	var connectionClosed bool
	var waiter sync.WaitGroup
	waiter.Add(1)

	// tty >>> xterm.js
	go func() {
		errorCount := 0
		for {
			if errorCount > connectionErrorLimit {
				waiter.Done()
				break
			}
			buffer := make([]byte, 8192)
			n, err := proc.Read(buffer)
			if err != nil {
				if err == io.EOF {
					continue
				}
				log.Infof("failed to read tty: %v", err)
				if err := conn.Write(ctx, websocket.MessageText, []byte("bye!")); err != nil {
					log.Infof("failed to send termination message from tty to xterm.js: %v", err)
				}
				waiter.Done()
				break
			}
			if err := conn.Write(ctx, websocket.MessageBinary, buffer[:n]); err != nil {
				log.Infof("failed to send %v bytes from tty to xterm.js: %v", n, err)
				errorCount++
				continue
			}
			log.Infof("tty >> xterm.js sent: %d", n)
			errorCount = 0
		}
	}()

	// tty << xterm.js
	go func() {
		for {
			messageType, message, err := conn.Read(ctx)
			if err != nil {
				if !connectionClosed {
					log.Infof("failed to get next reader: %s", err)
				}
				log.Infof("gracefully stopping spawned tty...")
				if err := proc.Kill(); err != nil {
					log.Infof("failed to kill tty: %v", err)
				}
				break
			}

			dataBuffer, ttySize, isResize, err := parseIncomingMessage(messageType, message)
			if err != nil {
				log.Infof("failed to parse terminal message: %v", err)
				continue
			}

			if isResize {
				log.Infof("ttySize: %v", ttySize)
				if err := proc.SetSize(int(ttySize.Cols), int(ttySize.Rows)); err != nil {
					log.Infof("failed to set ttySize: %v", err)
				}
				continue
			}

			if len(dataBuffer) == 0 {
				log.Infof("received empty terminal message, ignoring")
				continue
			}

			// write to tty
			bytesWritten, err := proc.Write(dataBuffer)
			if err != nil {
				log.Infof("failed to write tty: %v", err)
				continue
			}
			log.Infof("tty << xterm.js: %v", bytesWritten)
		}
	}()

	waiter.Wait()

	connectionClosed = true
}
