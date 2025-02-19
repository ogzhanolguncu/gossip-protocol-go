package transport

import (
	"context"
	"net"
	"sync"

	"github.com/ogzhanolguncu/gossip-protocol/internal/protocol"
)

type TCPTransport struct {
	addr     string
	listener net.Listener
	handler  func(string, *protocol.Message) error
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewTCPTransport(addr string) (*TCPTransport, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TCPTransport{
		addr:     addr,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (t *TCPTransport) Send(addr string, msg *protocol.Message) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(msg.Encode())
	return err
}

func (t *TCPTransport) Listen(handler func(string, *protocol.Message) error) error {
	t.handler = handler
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
				conn, err := t.listener.Accept()
				if err != nil {
					continue
				}
				go t.handleConn(conn)
			}
		}
	}()

	return nil
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	msg, err := protocol.DecodeMessage(buf[:n])
	if err != nil {
		return
	}

	if t.handler != nil {
		t.handler(conn.RemoteAddr().String(), msg)
	}
}

func (t *TCPTransport) Close() error {
	t.cancel()
	if t.listener != nil {
		t.listener.Close()
	}
	t.wg.Wait()
	return nil
}
