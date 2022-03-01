package postfix

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
)

const DefaultPort int = 42002

type SocketMap struct {
	be     Backend
	logger *zap.SugaredLogger
}

func NewSocketMap(be Backend, logger *zap.SugaredLogger) (*SocketMap, error) {
	h := &SocketMap{
		be:     be,
		logger: logger,
	}
	return h, nil
}

func (h *SocketMap) ListenAndServe(addr string) error {
	return h.ListenAndServeContext(context.Background(), addr)
}

func (h *SocketMap) ListenAndServeContext(ctx context.Context, addr string) error {
	if addr == "" {
		addr = fmt.Sprintf(":%d", DefaultPort)
	}
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return h.Serve(l)
}

func (h *SocketMap) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		c := h.newClient(conn)
		go c.handleClient()
	}
}

func (h *SocketMap) newClient(conn net.Conn) *clientImpl {
	c := &clientImpl{
		h:      h,
		conn:   conn,
		logger: h.logger,
		be:     h.be,
	}
	return c
}
