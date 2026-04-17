package imap

import (
	"bufio"
	"bytes"
	"net"
	"regexp"
	"strings"
	"sync"
)

var exchangeRespCodeQuotedValuePattern = regexp.MustCompile(`([A-Za-z0-9_-]+)="([^"]*)"`)

type exchangeResponseNormalizerConn struct {
	net.Conn

	reader *bufio.Reader

	mu                   sync.Mutex
	pending              []byte
	normalizationEnabled bool
	readErr              error
}

func newExchangeResponseNormalizerConn(conn net.Conn) *exchangeResponseNormalizerConn {
	return &exchangeResponseNormalizerConn{
		Conn:                 conn,
		reader:               bufio.NewReader(conn),
		normalizationEnabled: true,
	}
}

func (c *exchangeResponseNormalizerConn) DisableNormalization() {
	c.mu.Lock()
	c.normalizationEnabled = false
	c.mu.Unlock()
}

func (c *exchangeResponseNormalizerConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pending) == 0 {
		if !c.normalizationEnabled {
			return c.reader.Read(p)
		}

		line, err := c.reader.ReadBytes('\n')
		if len(line) == 0 {
			return 0, err
		}

		c.pending = normalizeExchangeResponseLine(line)
		c.readErr = err
	}

	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	if len(c.pending) > 0 {
		return n, nil
	}

	err := c.readErr
	c.readErr = nil
	return n, err
}

func normalizeExchangeResponseLine(line []byte) []byte {
	start := bytes.IndexByte(line, '[')
	if start == -1 {
		return line
	}

	end := bytes.IndexByte(line[start:], ']')
	if end == -1 {
		return line
	}
	end += start

	respCode := string(line[start+1 : end])
	if !strings.Contains(respCode, `="`) {
		return line
	}

	normalizedRespCode := exchangeRespCodeQuotedValuePattern.ReplaceAllString(respCode, `$1=$2`)
	if normalizedRespCode == respCode {
		return line
	}

	normalized := make([]byte, 0, len(line))
	normalized = append(normalized, line[:start+1]...)
	normalized = append(normalized, normalizedRespCode...)
	normalized = append(normalized, line[end:]...)
	return normalized
}
