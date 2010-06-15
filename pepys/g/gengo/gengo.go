package gengo

import "bytes"
import "strconv"
import "strings"
import "io/ioutil"

type Description []Operation
type Operation struct {
	Code int
	Name string
	Args []map[string]string
}

const AUTO_GEN = "/* THIS IS AN AUTOMATICALLY GENERATED FILE, DO NOT EDIT! */\n"

func opTypes(desc Description) string {
	types := new(bytes.Buffer)
	for _, op := range desc {
		types.WriteString("type " + op.Name + " struct { \n")
		for _, arg := range op.Args {
			for argname, argtype := range arg {				
				types.WriteString("\t" + argname + "\t" + argtype + "\n")
			}
		}
		types.WriteString("}\n\n")
	}
	return types.String()
}

// Generates functions for encoding and decoding each message type
func opMethods(desc Description) string {
	methods := new(bytes.Buffer)
	for _, op := range desc {
		// encode method
		methods.WriteString("func encode" + op.Name)
		methods.WriteString("(arg *" + op.Name + ", buf io.Writer) os.Error {\n")
		
		for _, arg := range op.Args {
		for argname, argtype := range arg {
			switch argtype {
			case "string":
				methods.WriteString("\tencodeString(arg." + argname + ", buf)\n")
			case "data":
				methods.WriteString("\tencodeData(arg." + argname + ", buf)\n")
			default:
				methods.WriteString("\tbinary.Write(buf, binary.BigEndian, arg." + argname + ")\n")
			}
		}
		}
		methods.WriteString("\treturn nil\n}\n")
		
		// decode method
		methods.WriteString("func decode" + op.Name)
		methods.WriteString("(buf io.Reader) *" + op.Name + " {\n")
		methods.WriteString("\tval := new(" + op.Name + ")\n")
		
		for _, arg := range op.Args {
		for argname, argtype := range arg {
			switch argtype {
			case "string":
				methods.WriteString("\tval." + argname + " = " + "decodeString(buf)\n")
			case "data":
				methods.WriteString("\tval." + argname + " = " + "decodeData(buf)\n")
			default:
				methods.WriteString("\tbinary.Read(buf, binary.BigEndian, &(val." + argname + "))\n")
			}
		}
		}
		methods.WriteString("\treturn val\n}\n\n")
	}
	
	return methods.String()
}

// Generates the public methods for this module. Break into more functions?
func opPublicMethods (desc Description) string {
	methods := new(bytes.Buffer)
	methods.WriteString(`/* Public methods begin here */
func NewPacket(buf io.Reader) *Packet {
	pkt := new(Packet)
	
	// length is including the length field
	// do we really need this?
	var length uint32
	binary.Read(buf, binary.BigEndian, &length)
	
	// read out number of messages contained in packet
	var nmsgs uint32
	binary.Read(buf, binary.BigEndian, &nmsgs)
	
	// allocate message space
	pkt.Msgs = make([]interface{}, nmsgs)
	
	var mtype uint32
	for i := 0; i < int(nmsgs); i++ {
		// read message type
		binary.Read(buf, binary.BigEndian, &mtype)
		switch mtype {
`)
	
	for _, op := range desc {
		methods.WriteString("\t\tcase " + op.Name + "Code:\n")
		methods.WriteString("\t\t\tpkt.Msgs[i] = decode" + op.Name + "(buf)\n")
	}
	methods.WriteString("\t\t}\n\t}\n\treturn pkt\n}\n")

	methods.WriteString(`
func (pkt *Packet) Send(buf io.Writer) os.Error {
	nmsgs := uint32(len(pkt.Msgs))
	tmpbuf := new(bytes.Buffer)
	
	for _, op := range pkt.Msgs {
		mtype := reflect.Typeof(op).String()
		switch mtype {
`)

	for _, op := range desc {
		methods.WriteString("\t\tcase \"*pepys." + op.Name + "\":\n")
		methods.WriteString("\t\t\tbinary.Write(tmpbuf, binary.BigEndian, uint32(" + op.Name + "Code))\n")
		methods.WriteString("\t\t\tencode" + op.Name + "(op.(*" + op.Name + "), tmpbuf)\n")
	}
	
	methods.WriteString("\t\t}\n\t}\n")
	methods.WriteString(`
	// total size if size of messages + 4 (nmsgs) + 4 (length)
	total := uint32(len(tmpbuf.Bytes()) + 8)
	binary.Write(buf, binary.BigEndian, total)
	binary.Write(buf, binary.BigEndian, nmsgs)
	buf.Write(tmpbuf.Bytes())
	
	return nil
}`)

	methods.WriteString(`

func (pkt *Packet) Add(op interface{}) os.Error {
	clen := len(pkt.Msgs)
	newMsgs := make([]interface{}, clen + 1)

	for i, cur := range pkt.Msgs {
		newMsgs[i] = cur
	}
	newMsgs[clen] = op
	pkt.Msgs = newMsgs
	
	// FIXME: error checking
	return nil
}
`)
	
	return methods.String()
}

func srvInterface(desc Description) string {
	inter := new(bytes.Buffer)
	inter.WriteString("type Operations interface {\n")
	for _, op := range desc {
		// Server callbacks are only for T messages
		if op.Name[0] == 'T' {
			inter.WriteString("\t" + strings.ToUpper(op.Name[1:2]) + op.Name[2:])
			inter.WriteString("(conn *Connection, arg *pepys." + op.Name + ") ")
			inter.WriteString("(*pepys.R" + op.Name[1:] + ", os.Error)\n")
		}
	}
	inter.WriteString("}\n")
	return inter.String()
}

func srvProcess(desc Description) string {
	proc := new(bytes.Buffer)
	proc.WriteString(`
// Process incoming requests from a client
func (conn *Connection) process() {
	// Pepys messages begin with a total message size, followed by the number
	// of operation included in this "group". We parse each one and execute
	// them in-order while buffering the results. As soon as all operations
	// execute successfully, or an operation fails, the results are sent back
	// to the client.
	for {
		request := pepys.NewPacket(conn.handle)
		response := new(pepys.Packet)
	
		var err os.Error
		var cresp interface{}
		for _, op := range request.Msgs {
			mtype := reflect.Typeof(op).String()
			switch mtype {
`)

	for _, op := range desc {
		// Server callbacks are only for T messages
		if op.Name[0] == 'T' {
			proc.WriteString("\t\t\tcase \"*pepys." + op.Name + "\":\n")
			proc.WriteString("\t\t\t\tcresp, err = conn.Srv.ops." + strings.ToUpper(op.Name[1:2]) + op.Name[2:])
			proc.WriteString("(conn, op.(*pepys." + op.Name + "))\n")
		}
	}
	
	proc.WriteString(`
			}
		
			if err == nil {
				response.Add(cresp)
			} else {
				ePkt := new(pepys.Rerror)
				ePkt.Ename = err.String()
				response.Add(cresp)
				
				// Do not process any more messages
				break
			}
		}
	
		// Off you go
		response.Send(conn.handle)
	}
}

`)

	return proc.String()
}

func Generate(desc Description, dir string, rel bool) []string {
	// Generate message constants
	opCodes := "const (\n"
	
	// Iterate over protocol operations and populate codes.
	// Also change types
	for i, op := range desc {
		opCodes = opCodes + "\t" + op.Name + "Code = " + strconv.Itoa(op.Code) + "\n"
		for j, arg := range op.Args {
			for k, argtype := range arg {				
				switch (argtype) {
					case "u16int": desc[i].Args[j][k] = "uint16";
					case "u32int": desc[i].Args[j][k] = "uint32";
					case "u64int": desc[i].Args[j][k] = "uint64";
				}
			}
		}
	}
	opCodes = opCodes + ")\n\n"
	
	// Extract headers
	pepysHeader, _ := ioutil.ReadFile(dir + "/pepys.go")
	serverHeader, _ := ioutil.ReadFile(dir + "/server.go")
	
	pepys := bytes.NewBuffer([]byte(AUTO_GEN))
	pepys.Write(pepysHeader)
	
	server := bytes.NewBuffer([]byte(AUTO_GEN))
	server.Write(serverHeader)
	
	// Collect parts of pepys main file and write it
	pepys.WriteString(opCodes)
	pepys.WriteString(opTypes(desc))
	pepys.WriteString(opMethods(desc))
	pepys.WriteString(opPublicMethods(desc))
	ioutil.WriteFile("pepys.go", pepys.Bytes(), 0644)
	
	// Collect parts of server file and write it
	server.WriteString(srvProcess(desc))
	server.WriteString(srvInterface(desc))
	ioutil.WriteFile("server.go", server.Bytes(), 0644)

	return []string{"pepys.go", "server.go"}
}
