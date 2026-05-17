package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HandlerFunc func(params json.RawMessage) (interface{}, error)

type Server struct {
	socketPath string
	handlers   map[string]HandlerFunc
}

func NewServer(socketPath string) *Server {
	return &Server{
		socketPath: socketPath,
		handlers:   make(map[string]HandlerFunc),
	}
}

func (s *Server) Register(method string, handler HandlerFunc) {
	s.handlers[method] = handler
}

func (s *Server) Listen() error {
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove socket: %w", err)
	}

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	defer ln.Close()

	if err := os.Chmod(s.socketPath, 0600); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	log.Printf("Agent listening on %s", s.socketPath)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	var req Request
	if err := dec.Decode(&req); err != nil {
		return
	}

	resp := Response{JSONRPC: "2.0", ID: req.ID}
	handler, ok := s.handlers[req.Method]
	if !ok {
		resp.Error = &RPCError{Code: -32601, Message: "method not found: " + req.Method}
		enc.Encode(resp)
		return
	}

	result, err := handler(req.Params)
	if err != nil {
		resp.Error = &RPCError{Code: -32000, Message: err.Error()}
	} else {
		resp.Result = result
	}
	enc.Encode(resp)
}
