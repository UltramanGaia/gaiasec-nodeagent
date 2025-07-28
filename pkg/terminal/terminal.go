package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/creack/pty"
	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"os"
	"os/exec"
	"sothoth-nodeagent/pkg/config"
	"strings"
	"sync"
)

func CreateNewTerminalSocket(cols uint16, rows uint16, cmd string, sessionID string) {
	log.Infof("Creating new terminal [%s][%s]", sessionID, cmd)
	c := exec.Command(cmd, "-i")
	c.Env = os.Environ()
	tty, err := pty.Start(c)
	if err != nil {
		log.Infof("failed to start tty: %v", err)
		return
	}

	pty.Setsize(tty, &pty.Winsize{Rows: rows, Cols: cols})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessionID = strings.Replace(sessionID, "client-", "server-", -1)
	cfg := config.GetInstance()
	uri := "ws://" + cfg.ServerURL + "/ws/terminal?nodeId=" + cfg.NodeID + "&clientId=" + sessionID
	connection, _, err := websocket.Dial(ctx, uri, nil)
	if err != nil {
		log.Infof("failed to connect to terminal: %v", err)
		return
	}

	defer func() {
		log.Infof("gracefully stopping spawned tty...")
		if err := c.Process.Kill(); err != nil {
			log.Infof("failed to kill tty: %v", err)
		}
		if _, err := c.Process.Wait(); err != nil {
			log.Infof("failed to wait for tty: %v", err)
		}
		if err := tty.Close(); err != nil {
			log.Infof("failed to close tty: %v", err)
		}
		if err := connection.Close(websocket.StatusNormalClosure, ""); err != nil {
			log.Infof("failed to close connection: %v", err)
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
			n, err := tty.Read(buffer)
			if err != nil {
				log.Infof("failed to read tty: %v", err)
				if err := connection.Write(ctx, websocket.MessageText, []byte("bye!")); err != nil {
					log.Infof("failed to send termination message from tty to xterm.js: %v", err)
				}
				waiter.Done()
				break
			}
			if err := connection.Write(ctx, websocket.MessageBinary, buffer[:n]); err != nil {
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
			messageType, message, err := connection.Read(ctx)
			if err != nil {
				if !connectionClosed {
					log.Infof("failed to get next reader: %s", err)
				}
				log.Infof("gracefully stopping spawned tty...")
				if err := c.Process.Kill(); err != nil {
					log.Infof("failed to kill tty: %v", err)
				}
				break
			}
			dataLength := len(message)
			dataBuffer := bytes.Trim(message, "\x00")
			if dataLength == -1 {
				log.Infof("failed to get the correct data length, ignoring")
				continue
			}
			// handle resizing
			if messageType != websocket.MessageBinary {
				if dataBuffer[0] == 1 {
					ttySize := &TTYSize{}
					resizeMessage := bytes.Trim(dataBuffer[1:], "\n\r\t\x00\x01")
					if err := json.Unmarshal(resizeMessage, &ttySize); err != nil {
						log.Infof("failed to unmarshal ttySize: %v", err)
						continue
					}
					log.Infof("ttySize: %v", ttySize)
					if err := pty.Setsize(tty, &pty.Winsize{
						Rows: ttySize.Rows,
						Cols: ttySize.Cols,
					}); err != nil {
						log.Infof("failed to set ttySize: %v", err)
					}
					continue
				}
			}

			// write to tty
			bytesWritten, err := tty.Write(dataBuffer)
			if err != nil {
				log.Infof("failed to write tty: %v", err)
				continue
			}
			log.Infof("tty << xterm.js: %v", bytesWritten)
		}
	}()

	waiter.Wait()

	connectionClosed = true
	return
}
