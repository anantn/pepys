// timefs client
package main

import "os"
import "fmt"
import "net"
import "pepys"

const UNAME string = "testuser"

func main() {
	// create connection to localhost 5604 on tcp
	conn, err := net.Dial("tcp", "", "localhost:5640")
	if err != nil {
		fmt.Printf("Could not connect to timefs server!\n")
		os.Exit(1)
	}
	conn = conn.(*net.TCPConn)
	
	// first send Tproto
	pkt := new(pepys.Packet)
	proto := new(pepys.Tproto)
	proto.Msize = pepys.MSIZE
	proto.Nmsgs = pepys.NMSGS
	pkt.Add(proto)
	pkt.Send(conn)
	
	// wait for Rproto
	fmt.Printf("Waiting for Rproto from server... ")
	pkt = pepys.NewPacket(conn)
	fmt.Printf("received! %v\n", pkt)
	
	// then send Tsession
	pkt = new(pepys.Packet)
	session := new(pepys.Tsession)
	session.Csid = 0x1
	session.Uname = UNAME
	session.Afid = pepys.NOFID
	pkt.Add(session)
	pkt.Send(conn)
	
	// wait for Rsession
	fmt.Printf("Waiting for Rsession from server... ")
	pkt = pepys.NewPacket(conn)
	fmt.Printf("received! %v\n", pkt)
	
	// prepare Tattach, Topen and Tread all in one packet
	pkt = new(pepys.Packet)
	
	at := new(pepys.Tattach)
	at.Fid = uint32(1)
	at.Uname = UNAME
	at.Afid = pepys.NOFID
	at.Aname = "/"
	pkt.Add(at)
	
	op := new(pepys.Topen)
	op.Fid = uint32(2)
	op.Path = "/time"
	pkt.Add(op)
	
	re := new(pepys.Tread)
	re.Fid = uint32(2)
	re.Count = uint32(1024)
	pkt.Add(re)
	
	// send it!
	fmt.Printf("Sending attach/open/read... ")
	pkt.Send(conn)
	fmt.Printf("Sent!\n")
	
	// get response
	pkt = pepys.NewPacket(conn)
	fmt.Printf("Received response with %d messages\n", len(pkt.Msgs))
	
	if len(pkt.Msgs) != 3 {
		fmt.Printf("Ropen not received! Aborting.\n")
	}
	resp := pkt.Msgs[2].(*pepys.Rread)
	fmt.Printf("\nThe time is: %s\n", string(resp.Dat))
	
	conn.Close()
}
