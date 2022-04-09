package postfix

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type Client interface {
}

type clientImpl struct {
	h      *SocketMap
	conn   net.Conn
	logger *zap.SugaredLogger
	be     Backend

	buffer []byte
}

var _ Client = (*clientImpl)(nil)

const (
	// Length of the smallest allocated buffer, larger buffers will be allocated dynamically
	BaseBuffer = 1024
	// Maximum length of a buffer, will crash when exceeded
	MaxBuffer = 65535
)

func (c *clientImpl) initBuffer() {
	c.buffer = make([]byte, 0, MaxBuffer)
}

// Handles incoming requests.
func (c *clientImpl) handleClient() {
	c.logger.Debug("postfix client connected")
	defer c.conn.Close()

	c.initBuffer()

	// Make a buffer to hold incoming data.
	buf := make([]byte, 10240)
	for {
		// Read the incoming connection into the buffer.
		reqLen, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.logger.Debug("postfix client disconnected")
				return
			}
			c.logger.Error("Error reading:", err.Error())
			return
		}

		data := buf[0:reqLen]
		if err := c.dataReceived(data); err != nil {
			c.logger.Errorf("data error: %v", err)
			return
		}
	}
}

func (c *clientImpl) dataReceived(data []byte) error {
	c.buffer = append(c.buffer, data...)
	sepIndex := bytes.IndexByte(c.buffer, 0x3a)
	if sepIndex == -1 {
		//separator not yet found
		return nil
	}
	lenStr := string(c.buffer[0:sepIndex])
	var l int
	var err error
	if l, err = strconv.Atoi(lenStr); err != nil {
		return fmt.Errorf("can't read length value '%v': %w", lenStr, err)
	}
	str := string(c.buffer[sepIndex+1 : sepIndex+1+l])
	c.buffer = c.buffer[sepIndex+1+l+1:]
	return c.stringReceived(str)
}

func (c *clientImpl) stringReceived(str string) error {
	//SocketMap protocol
	c.logger.Debugf("received string '%v'", str)

	arr := strings.SplitN(str, " ", 2)
	if len(arr) != 2 {
		c.logger.Warnf("expected 2 items, found %v", len(arr))
		return nil
	}

	name, key := arr[0], arr[1]
	return c.processRequest(name, key)
}

func (c *clientImpl) processRequest(name, key string) error {
	c.logger.Debugf("Request %v/%v", name, key)

	t, value, err := c.be.Lookup(c, name, key)
	if err == nil {
		return c.replyString(t, value)
	}
	replyerr := c.replyString(ReplyTypeTEMP, err.Error())
	if replyerr != nil {
		return replyerr
	}
	return err
}

func (c *clientImpl) replyString(t ReplyType, value string) error {
	return c.sendString(fmt.Sprintf("%v %v", t, value))
}

func (c *clientImpl) sendString(str string) error {
	c.logger.Debugf("replying '%v'", str)
	bin := []byte(str)
	len := len(bin)
	lenStr := strconv.Itoa(len)
	if _, err := c.conn.Write([]byte(lenStr)); err != nil {
		return fmt.Errorf("write len failed: %w", err)
	}
	if _, err := c.conn.Write([]byte(":")); err != nil {
		return fmt.Errorf("write sep failed: %w", err)
	}
	if _, err := c.conn.Write(bin); err != nil {
		return fmt.Errorf("write bin failed: %w", err)
	}
	if _, err := c.conn.Write([]byte(",")); err != nil {
		return fmt.Errorf("write end failed: %w", err)
	}
	return nil
}
