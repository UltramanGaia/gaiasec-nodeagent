package terminal

import (
	"bytes"
	"encoding/json"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sothoth-nodeagent/pkg/config"
	"strings"
	"sync"
)

func CreateNewTerminalSocket(cols uint16, rows uint16, cmd string, sessionID string) {
	log.Printf("Creating new terminal [%s][%s]", sessionID, cmd)
	c := exec.Command(cmd, "-i")
	c.Env = os.Environ()
	tty, err := pty.Start(c)
	if err != nil {
		log.Printf("failed to start tty: %v", err)
		return
	}

	pty.Setsize(tty, &pty.Winsize{Rows: rows, Cols: cols})

	dialer := websocket.Dialer{}
	sessionID = strings.Replace(sessionID, "client-", "server-", -1)
	cfg := config.GetInstance()
	uri := "ws://" + cfg.ServerURL + "/ws/terminal?nodeId=" + cfg.NodeID + "&clientId=" + sessionID
	connection, _, err := dialer.Dial(uri, http.Header{})
	if err != nil {
		log.Printf("failed to connect to terminal: %v", err)
		return
	}

	defer func() {
		log.Printf("gracefully stopping spawned tty...")
		if err := c.Process.Kill(); err != nil {
			log.Printf("failed to kill tty: %v", err)
		}
		if _, err := c.Process.Wait(); err != nil {
			log.Printf("failed to wait for tty: %v", err)
		}
		if err := tty.Close(); err != nil {
			log.Printf("failed to close tty: %v", err)
		}
		if err := connection.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
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
				log.Printf("failed to read tty: %v", err)
				if err := connection.WriteMessage(websocket.TextMessage, []byte("bye!")); err != nil {
					log.Printf("failed to send termination message from tty to xterm.js: %v", err)
				}
				waiter.Done()
				break
			}
			if err := connection.WriteMessage(websocket.BinaryMessage, buffer[:n]); err != nil {
				log.Printf("failed to send %v bytes from tty to xterm.js: %v", n, err)
				errorCount++
				continue
			}
			log.Printf("tty >> xterm.js sent: %d", n)
			errorCount = 0
		}
	}()

	// tty << xterm.js
	go func() {
		for {
			messageType, message, err := connection.ReadMessage()
			if err != nil {
				if !connectionClosed {
					log.Printf("failed to get next reader: %s", err)
				}
				log.Printf("gracefully stopping spawned tty...")
				if err := c.Process.Kill(); err != nil {
					log.Printf("failed to kill tty: %v", err)
				}
				break
			}
			dataLength := len(message)
			dataBuffer := bytes.Trim(message, "\x00")
			if dataLength == -1 {
				log.Printf("failed to get the correct data length, ignoring")
				continue
			}
			// handle resizing
			if messageType != websocket.BinaryMessage {
				if dataBuffer[0] == 1 {
					ttySize := &TTYSize{}
					resizeMessage := bytes.Trim(dataBuffer[1:], "\n\r\t\x00\x01")
					if err := json.Unmarshal(resizeMessage, &ttySize); err != nil {
						log.Printf("failed to unmarshal ttySize: %v", err)
						continue
					}
					log.Printf("ttySize: %v", ttySize)
					if err := pty.Setsize(tty, &pty.Winsize{
						Rows: ttySize.Rows,
						Cols: ttySize.Cols,
					}); err != nil {
						log.Printf("failed to set ttySize: %v", err)
					}
					continue
				}
			}

			// write to tty
			bytesWritten, err := tty.Write(dataBuffer)
			if err != nil {
				log.Printf("failed to write tty: %v", err)
				continue
			}
			log.Printf("tty << xterm.js: %v", bytesWritten)
		}
	}()

	waiter.Wait()

	connectionClosed = true
	return
}
