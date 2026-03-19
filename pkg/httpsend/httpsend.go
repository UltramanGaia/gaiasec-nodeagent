package httpsend

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gaiasec-nodeagent/pkg/pb"
)

const (
	CRLF           = "\r\n"
	BufferSize     = 8192
	ConnectTimeout = 30 * time.Second
	ReadTimeout    = 30 * time.Second
)

var (
	contentLengthPattern = regexp.MustCompile(`(?i)Content-Length:\s*(\d+)`)
	chunkedPattern       = regexp.MustCompile(`(?i)Transfer-Encoding:\s*chunked`)
	hostPattern          = regexp.MustCompile(`(?i)^Host:`)
)

type HttpSendService struct {
	clientCert []byte
	clientKey  []byte
}

func NewHttpSendService() *HttpSendService {
	return &HttpSendService{}
}

func (s *HttpSendService) SendRequest(request *pb.HttpSendRequestProto) *pb.HttpSendResponseProto {
	startTime := time.Now()

	connectTimeout := time.Duration(request.ConnectTimeout) * time.Millisecond
	if connectTimeout <= 0 {
		connectTimeout = ConnectTimeout
	}
	readTimeout := time.Duration(request.ReadTimeout) * time.Millisecond
	if readTimeout <= 0 {
		readTimeout = ReadTimeout
	}

	rawRequest := normalizeLineEndings(request.Data)
	rawRequest = ensureHostHeader(rawRequest, request.Host, request.Port, request.Secure)
	rawRequest = addMicroDebugHeader(rawRequest)
	rawRequest = recalculateContentLength(rawRequest)

	log.Infof("Sending HTTP request to %s:%d (secure=%v)", request.Host, request.Port, request.Secure)
	log.Debugf("Raw request:\n%s", rawRequest)

	requestBytes := []byte(rawRequest)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", request.Host, request.Port), connectTimeout)
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		return &pb.HttpSendResponseProto{
			Success: false,
			Message: fmt.Sprintf("Failed to connect: %v", err),
		}
	}
	defer conn.Close()

	if request.Secure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			log.Errorf("TLS handshake failed: %v", err)
			return &pb.HttpSendResponseProto{
				Success: false,
				Message: fmt.Sprintf("TLS handshake failed: %v", err),
			}
		}
		conn = tlsConn
	}

	conn.SetDeadline(time.Now().Add(readTimeout))

	_, err = conn.Write(requestBytes)
	if err != nil {
		log.Errorf("Failed to send request: %v", err)
		return &pb.HttpSendResponseProto{
			Success: false,
			Message: fmt.Sprintf("Failed to send request: %v", err),
		}
	}

	response, err := readResponse(conn)
	if err != nil {
		log.Errorf("Failed to read response: %v", err)
		return &pb.HttpSendResponseProto{
			Success: false,
			Message: fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	responseTime := time.Since(startTime).Milliseconds()
	log.Infof("Received response in %d ms", responseTime)
	log.Debugf("Raw response:\n%s", response)

	return &pb.HttpSendResponseProto{
		Success:      true,
		Message:      "OK",
		Response:     response,
		ResponseTime: responseTime,
	}
}

func normalizeLineEndings(request string) string {
	request = strings.ReplaceAll(request, "\r\n", "\n")
	request = strings.ReplaceAll(request, "\r", "\n")
	request = strings.ReplaceAll(request, "\n", CRLF)
	return request
}

func ensureHostHeader(request, host string, port int32, secure bool) string {
	lines := strings.Split(request, CRLF)
	if len(lines) < 1 {
		return request
	}

	for _, line := range lines {
		if hostPattern.MatchString(line) {
			return request
		}
	}

	hostValue := host
	if (secure && port != 443) || (!secure && port != 80) {
		hostValue = fmt.Sprintf("%s:%d", host, port)
	}

	if len(lines) >= 1 {
		lines[0] = lines[0] + CRLF + "Host: " + hostValue
		return strings.Join(lines, CRLF)
	}

	return request
}

func addMicroDebugHeader(request string) string {
	lines := strings.Split(request, CRLF)
	if len(lines) < 1 {
		return request
	}

	lines[0] = lines[0] + CRLF + "micro-debug: all"
	return strings.Join(lines, CRLF)
}

func recalculateContentLength(request string) string {
	headerEndIndex := strings.Index(request, CRLF+CRLF)
	if headerEndIndex < 0 {
		return request
	}

	headerPart := request[:headerEndIndex]
	body := request[headerEndIndex+4:]

	if body == "" {
		return request
	}

	if chunkedPattern.MatchString(headerPart) {
		return request
	}

	bodyLength := len(body)

	if contentLengthPattern.MatchString(headerPart) {
		newHeader := contentLengthPattern.ReplaceAllString(headerPart, fmt.Sprintf("Content-Length: %d", bodyLength))
		return newHeader + CRLF + CRLF + body
	}

	return headerPart + CRLF + fmt.Sprintf("Content-Length: %d", bodyLength) + CRLF + CRLF + body
}

func readResponse(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)

	var headerBuffer bytes.Buffer
	var headerEndPos = -1

	for headerEndPos < 0 {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		headerBuffer.WriteString(line)

		headerStr := headerBuffer.String()
		headerEndPos = strings.Index(headerStr, CRLF+CRLF)
	}

	if headerEndPos < 0 {
		return headerBuffer.String(), nil
	}

	headerPart := headerBuffer.String()[:headerEndPos]

	var bodyBuffer bytes.Buffer

	chunkedMatcher := chunkedPattern.MatchString(headerPart)
	if chunkedMatcher {
		bodyBytes, err := readChunkedBody(reader)
		if err != nil {
			return "", err
		}
		bodyBuffer.Write(bodyBytes)
	} else {
		contentLengthMatcher := contentLengthPattern.FindStringSubmatch(headerPart)
		if len(contentLengthMatcher) > 1 {
			contentLength, _ := strconv.Atoi(contentLengthMatcher[1])
			bodyBytes, err := readBodyWithContentLength(reader, contentLength)
			if err != nil {
				return "", err
			}
			bodyBuffer.Write(bodyBytes)
		} else {
			bodyBytes, err := readBodyUntilClose(reader)
			if err != nil {
				return "", err
			}
			bodyBuffer.Write(bodyBytes)
		}
	}

	return headerPart + CRLF + CRLF + bodyBuffer.String(), nil
}

func readBodyWithContentLength(reader *bufio.Reader, contentLength int) ([]byte, error) {
	bodyBuffer := make([]byte, contentLength)
	_, err := io.ReadFull(reader, bodyBuffer)
	if err != nil {
		return nil, err
	}
	return bodyBuffer, nil
}

func readChunkedBody(reader *bufio.Reader) ([]byte, error) {
	var bodyBuffer bytes.Buffer

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		semicolonIndex := strings.Index(line, ";")
		if semicolonIndex >= 0 {
			line = line[:semicolonIndex]
		}

		chunkSize, err := strconv.ParseInt(line, 16, 64)
		if err != nil {
			return nil, err
		}

		if chunkSize == 0 {
			reader.ReadString('\n')
			break
		}

		chunkData := make([]byte, chunkSize)
		_, err = io.ReadFull(reader, chunkData)
		if err != nil {
			return nil, err
		}
		bodyBuffer.Write(chunkData)

		reader.ReadString('\n')
	}

	return bodyBuffer.Bytes(), nil
}

func readBodyUntilClose(reader *bufio.Reader) ([]byte, error) {
	var bodyBuffer bytes.Buffer
	buf := make([]byte, BufferSize)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			bodyBuffer.Write(buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return bodyBuffer.Bytes(), nil
}
