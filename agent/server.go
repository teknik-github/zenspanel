package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
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
	socketPath  string
	socketGroup string
	handlers    map[string]HandlerFunc
}

func NewServer(socketPath string) *Server {
	return &Server{
		socketPath: socketPath,
		handlers:   make(map[string]HandlerFunc),
	}
}

// SetSocketGroup configures the unix group that should own the socket along
// with root. The agent runs as root; the API runs as a non-root account
// (typically www-data). With group ownership and mode 0660, the API can
// connect while no other unprivileged account can.
func (s *Server) SetSocketGroup(group string) {
	s.socketGroup = group
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

	mode := os.FileMode(0600)
	if s.socketGroup != "" {
		g, err := user.LookupGroup(s.socketGroup)
		if err != nil {
			return fmt.Errorf("lookup socket group %q: %w", s.socketGroup, err)
		}
		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			return fmt.Errorf("parse gid for %q: %w", s.socketGroup, err)
		}
		if err := os.Chown(s.socketPath, 0, gid); err != nil {
			return fmt.Errorf("chown socket to root:%s: %w", s.socketGroup, err)
		}
		mode = 0660
	}
	if err := os.Chmod(s.socketPath, mode); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	log.Printf("Agent listening on %s (mode %#o, group %q)", s.socketPath, mode, s.socketGroup)
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
