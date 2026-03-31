package terminal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func TestParseIncomingMessageIgnoresEmptyPayload(t *testing.T) {
	data, ttySize, isResize, err := parseIncomingMessage(websocket.MessageText, []byte{0, 0, 0})
	require.NoError(t, err)
	require.Nil(t, data)
	require.Nil(t, ttySize)
	require.False(t, isResize)
}

func TestParseIncomingMessageParsesResizePayload(t *testing.T) {
	message := append([]byte{1}, []byte("{\"cols\":120,\"rows\":40}\n\x00")...)

	data, ttySize, isResize, err := parseIncomingMessage(websocket.MessageText, message)
	require.NoError(t, err)
	require.Nil(t, data)
	require.NotNil(t, ttySize)
	require.True(t, isResize)
	require.Equal(t, uint16(120), ttySize.Cols)
	require.Equal(t, uint16(40), ttySize.Rows)
}

func TestParseIncomingMessageReturnsTerminalInput(t *testing.T) {
	data, ttySize, isResize, err := parseIncomingMessage(websocket.MessageText, []byte("ls -la"))
	require.NoError(t, err)
	require.Equal(t, []byte("ls -la"), data)
	require.Nil(t, ttySize)
	require.False(t, isResize)
}
