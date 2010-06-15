package server

import "os"
import "net"
import "pepys"
import "reflect"

type Server struct {
	// set if needed, library has defaults
	Msize uint32
	Nmsgs uint32
	
	// use at will
	Aux interface{}
	
	// private
	sock net.Listener
	ops *Operations
}
type Connection struct {
	// preset
	Srv *Server
	Msize uint32
	Nmsgs uint32
	RemoteAddr string
	
	// use at will
	Aux interface{}
	
	// private
	handle net.Conn
}

// Create a pepys server with protocol "proto" at address "addr" and listen
func New(ops Operations, proto string, addr string) (*Server, os.Error) {
	srv := new(Server)
	
	srv.ops = &ops;
	srv.Nmsgs = pepys.NMSGS
	srv.Msize = pepys.MSIZE
	
	// We begin implementation with a single-threaded version because incoming
	// requests have to be handled in-order. We may extend this to a multi-
	// threaded (goroutine) version in the future.
	sock, err := net.Listen(proto, addr)
	if err != nil {
		return nil, err
	}
	
	srv.sock = sock
	return srv, nil
}

// Start accepting requests and handle them
func (srv *Server) Start() os.Error {
	for {
		handle, err := srv.sock.Accept()
		if err != nil {
			srv.sock.Close()
			return err
		}
		
		conn := newConnection(handle)
		conn.Srv = srv
		go conn.process()
	}
	
	return nil
}

// Create a new client-server connection
func newConnection(handle net.Conn) (conn *Connection) {
	conn = new(Connection)
	if addr := handle.RemoteAddr(); addr != nil {
		conn.RemoteAddr = addr.String()
	}
	
	conn.handle = handle
	return conn
}
