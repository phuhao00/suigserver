package game

import (
	"log"
	"net"
	"time"

	"github.com/phuhao00/suigserver/server/internal/sui"
)

type Server struct {
	listener  net.Listener
	suiClient *sui.Client
	quit      chan struct{}
}

func NewServer(suiClient *sui.Client) *Server {
	return &Server{
		suiClient: suiClient,
		quit:      make(chan struct{}),
	}
}

func (s *Server) Run() {
	var err error
	s.listener, err = net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Println("Game server started on :9000")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	// TODO: 协议解析、鉴权、消息收发
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}
		log.Printf("Received: %s", string(buf[:n]))
		// 示例：与Sui交互
		s.suiClient.CallMoveFunction("game", "on_message", []interface{}{string(buf[:n])})
		conn.Write([]byte("pong"))
	}
}

func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	time.Sleep(time.Second) // 等待连接关闭
}
