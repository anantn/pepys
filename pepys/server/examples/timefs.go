// timefs example
package main

import "os"
import "fmt"
import "time"
import "pepys"
import "bytes"
import "pepys/server"

const IOUNIT uint32 = 1024

type TimeOps struct {}

// A FidMap is generally useful, move to Client utility library?
type FidMap struct {
	fids map[int]string
}
func NewFidMap() *FidMap {
	fm := new(FidMap)
	fm.fids = make(map[int]string, 10)
	return fm
}
func (fm *FidMap) Exists(f uint32) bool {
	_, present := fm.fids[int(f)]
	return present
}
func (fm *FidMap) AddFid(f uint32, p string) bool {
	if fm.Exists(f) {
		return false
	}
	fm.fids[int(f)] = p
	return true
}
func (fm *FidMap) DelFid(f uint32) bool {
	if !fm.Exists(f) {
		return false
	}
	fm.fids[int(f)] = "", false
	return true
}
func (fm *FidMap) GetFid(f uint32) string {
	if !fm.Exists(f) {
		return ""
	}
	return fm.fids[int(f)]
}

func (to *TimeOps) Proto(conn *server.Connection, arg *pepys.Tproto) (*pepys.Rproto, os.Error) {
	// no special options supported
	resp := new(pepys.Rproto)
	resp.Msize = pepys.MSIZE
	resp.Nmsgs = pepys.NMSGS
	
	fmt.Printf("%s:: Tproto received, sending back Rproto\n", conn.RemoteAddr)
	return resp, nil
}

func (to *TimeOps) Session(conn *server.Connection, arg *pepys.Tsession) (*pepys.Rsession, os.Error) {
	// FIXME: Session should actually be present in the library, not exposed to the user!!
	resp := new(pepys.Rsession)
	resp.Ssid = 0x1
	
	fmt.Printf("%s:: Tsession received, sending back Rsession\n", conn.RemoteAddr)
	return resp, nil
}

func (to *TimeOps) Attach(conn *server.Connection, arg *pepys.Tattach) (*pepys.Rattach, os.Error) {	
	resp := new(pepys.Rattach)
	
	// initialize fid map
	fmap := NewFidMap()
	conn.Aux = fmap
	
	fmt.Printf("conn.Aux set to %v\n", conn.Aux)
	fmt.Printf("%s:: Tattach received, sending back Rattach for root\n", conn.RemoteAddr)
	return resp, nil
}

func (to *TimeOps) Open(conn *server.Connection, arg *pepys.Topen) (*pepys.Ropen, os.Error) {
	if arg.Path != "/time" {
		return nil, os.NewError("File non-existent!")
	}
	if arg.Fid == 0 {
		return nil, os.NewError("0 is an invalid fid!")
	}
	
	fmap := conn.Aux.(*FidMap)
	if fmap.Exists(arg.Fid) {
		return nil, os.NewError("Fid is already in use!")
	}
	fmap.AddFid(arg.Fid, arg.Path)
	
	resp := new(pepys.Ropen)
	resp.Iounit = IOUNIT
	fmt.Printf("%s:: Topen received for %s, sending back Ropen with fid=%d\n", conn.RemoteAddr, arg.Path, arg.Fid)
	return resp, nil
}

func (to *TimeOps) Read(conn *server.Connection, arg *pepys.Tread) (*pepys.Rread, os.Error) {
	fmap := conn.Aux.(*FidMap)
	if !fmap.Exists(arg.Fid) {
		return nil, os.NewError("Fid not found!")
	}
	
	t := []byte(time.LocalTime().String() + "\n")
	rl := len(t)
	if int(arg.Count) < rl {
		rl = int(arg.Count)
	}
	
	by := new(bytes.Buffer)
	by.Write(t[0:rl])
	
	resp := new(pepys.Rread)
	resp.Dat = by.Bytes()
	
	fmt.Printf("%s:: Tread received, sending back Rread with %v\n", conn.RemoteAddr, resp.Dat)
	return resp, nil
}

func (to *TimeOps) Close(conn *server.Connection, arg *pepys.Tclose) (*pepys.Rclose, os.Error) {
	// we don't really care about this, always succeed if fid exists
	fmap := conn.Aux.(*FidMap)
	if !fmap.Exists(arg.Fid) {
		return nil, os.NewError("Fid not found!")
	}
	
	resp := new(pepys.Rclose)
	return resp, nil
}

func (to *TimeOps) Clunk(conn *server.Connection, arg *pepys.Tclunk) (*pepys.Rclunk, os.Error) {
	fmap := conn.Aux.(*FidMap)
	if !fmap.Exists(arg.Fid) {
		return nil, os.NewError("Fid not found!")
	}
	fmap.DelFid(arg.Fid)
	
	resp := new(pepys.Rclunk)
	return resp, nil
}

// operations unsupported by timefs
func (to *TimeOps) Flush(conn *server.Connection, arg *pepys.Tflush) (*pepys.Rflush, os.Error) {
	return nil, os.NewError("Flush is not supported!")
}
func (to *TimeOps) Write(conn *server.Connection, arg *pepys.Twrite) (*pepys.Rwrite, os.Error) {
	return nil, os.NewError("Write is not supported!")
}
func (to *TimeOps) Create(conn *server.Connection, arg *pepys.Tcreate) (*pepys.Rcreate, os.Error) {
	return nil, os.NewError("Create is not supported!")
}

func main() {
	// create server on localhost:5640
	to := new(TimeOps)
	srv, err := server.New(to, "tcp", "localhost:5640")
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("timefs server is ready and listening!\n")
	
	// start processing requests
	srv.Start()
}
