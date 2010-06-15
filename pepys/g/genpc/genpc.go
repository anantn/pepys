package genpc

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


const AUTO_GEN = "/* THIS IS AN AUTOMATICALLY GENERATED FILE, DO NOT EDIT! */\n\n"

// Basic C encoding routines
func opBasic() []byte {
	basic := new(bytes.Buffer)
	types := map[string]int{"s":16, "l":32, "v":64}
	
	basic.WriteString(`#include <u.h>
#include <libc.h>
#include <ip.h>
#include "πp.h"

// FIXME: All following methods should have more robust error checks
static void check_enc(Block*, int);
static void check_dec(Block*, int);

`)

	for c, sz := range types {
		b := strconv.Itoa(sz / 8)
		s := strconv.Itoa(sz)
		
		// encoding routine
		basic.WriteString("static void\nencode_u" + s + "int(Block* blk, u" + s + "int val)\n{\n")
		basic.WriteString("\tcheck_enc(blk, " + b + ");\n")
		basic.WriteString("\thnput" + c + "((void*) blk->wrp, val);\n")
		basic.WriteString("\tblk->wrp = blk->wrp + " + b + ";\n}\n")
		
		// decoding routine
		basic.WriteString("static void\ndecode_u" + s + "int(Block *blk, u" + s + "int* val)\n{\n")
		basic.WriteString("\tcheck_dec(blk, " + b + ");\n")
		basic.WriteString("\t*val = nhget" + c + "((void*) blk->rdp);\n")
		basic.WriteString("\tblk->rdp = blk->rdp + " + b + ";\n}\n")
	}
	
	basic.WriteString("\n")
	return basic.Bytes()
}

func opTypes(desc Description) []byte {
	types := new(bytes.Buffer)
	
	// Start with all the typedefs
	for _, op := range desc {
		types.WriteString("typedef struct " + op.Name + " " + op.Name + ";\n")
	}
	types.WriteString("\ntypedef struct Group Group;\n")
	types.WriteString("typedef struct Block Block;\n")
	types.WriteString("typedef struct Message Message;\n\n")
	
	// Individual messages
	for _, op := range desc {
		types.WriteString("struct " + op.Name + " {\n")
		
		// Dummy value for messages with no parameters
		if len(op.Args) <= 1 {
			types.WriteString("\tuchar\t_unused;\n")
		} else {
			for _, arg := range op.Args {
				for argname, argtype := range arg {
					an := strings.ToLower(argname)
					types.WriteString("\t" + argtype + "\t" + an + ";\n")
				}
			}
		}
		
		types.WriteString("};\n\n")
	}
	
	// Generic message struct
	types.WriteString("struct Message {\n")
	types.WriteString("\tu16int code;\n\tunion {\n")
	for _, op := range desc {
		types.WriteString("\t\t" + op.Name + " " + strings.ToLower(op.Name) + ";\n")
	}
	types.WriteString("\t};\n};\n")
	
	types.WriteString(`
struct Block {
	uchar*	beg;	// pointer to beginning of buffer
	uchar*	rdp;	// read pointer
	uchar*	wrp;	// write pointer
	uchar*	lim;	// pointer to end of buffer
};

struct Group {
	u32int	id;		// client or server session id
	u32int	tag;	// group tag
	Block*	blk;	// block associated with this group
};

// Public methods
int		B2M(Block*, Message*);
void	M2B(Block*, Message*);
`)
	return types.Bytes()
}

// Generates functions for encoding and decoding each message type
func opMethods(desc Description) []byte {
	methods := new(bytes.Buffer)
	for _, op := range desc {
		lop := strings.ToLower(op.Name)
		// encode method
		methods.WriteString("static void\nencode_" + lop + "(")
		if len(op.Args) > 1 {
			methods.WriteString("Block* blk, Message* msg)\n{\n")
			methods.WriteString("\t" + op.Name + "* arg = &(msg->" + lop + ");\n")
		} else {
			methods.WriteString("Block* blk, Message* msg)\n{")
		}
		
		for _, arg := range op.Args {
		for argname, argtype := range arg {
			carg := "(blk, arg->" + strings.ToLower(argname) + ")"
			if argtype == "char*" {
				argtype = "string"
			}
			if argtype == "Data" {
				argtype = "data"
				carg = "(blk, &(arg->" + strings.ToLower(argname) + "))"
			}
			methods.WriteString("\tencode_" + argtype + carg + ";\n")
		}
		}
		methods.WriteString("}\n")
		
		// decode method
		if len(op.Args) > 1 {
			methods.WriteString("static void\ndecode_" + lop + "(Block* blk, Message* msg)\n{\n")
			methods.WriteString("\t" + op.Name + "* arg = &(msg->" + lop + ");\n")
		} else {
			methods.WriteString("static void\ndecode_" + lop + "(Block* blk, Message* msg)\n{")
		}
		
		for _, arg := range op.Args {
		for argname, argtype := range arg {
			carg := "&(arg->" + strings.ToLower(argname) + ")"
			if argtype == "char*" {
				argtype = "string"
			}
			if argtype == "Data" {
				argtype = "data"
			}
			methods.WriteString("\tdecode_" + argtype + "(blk, " + carg + ");\n")
		}
		}
		methods.WriteString("}\n\n")
	}
	
	return methods.Bytes()
}

// Generates the public methods for this module. Break into more functions?
func opPublicMethods (desc Description) []byte {
	methods := new(bytes.Buffer)
	methods.WriteString("/* Public methods begin here */\n");
	
	// offset is
	offset := desc[0].Code
	codemap := make(map[int]string, len(desc))
	for _, op := range desc {
		codemap[op.Code] = strings.ToLower(op.Name)
	}
	firop := desc[0]
	lasop := desc[len(desc)-1]
	num := lasop.Code - firop.Code
	methods.WriteString("#define MNELEM\t" + strconv.Itoa(num + 1) + "\n")
	methods.WriteString("#define OFFSET\t" + strconv.Itoa(offset) + "\n\n")
	
	// tables
	encoder := new(bytes.Buffer)
	decoder := new(bytes.Buffer)
	
	encoder.WriteString("static void\n(*encode_table[])(Block*, Message*) = {\n")
	decoder.WriteString("static void\n(*decode_table[])(Block*, Message*) = {\n")
	for i := 0; i <= num; i = i + 1 {
		_, present := codemap[i + offset]
		if present {
			encoder.WriteString("\tencode_" + codemap[i + offset])
			decoder.WriteString("\tdecode_" + codemap[i + offset])
		} else {
			encoder.WriteString("\tnil")
			decoder.WriteString("\tnil")
		}
		
		if i != num {
			encoder.WriteString(",\n")
			decoder.WriteString(",\n")
		}
	}
	encoder.WriteString("\n};\n")
	decoder.WriteString("\n};\n")
	
	methods.Write(encoder.Bytes())
	methods.Write(decoder.Bytes())
	methods.WriteString(`
void
M2B(Block* blk, Message* msg)
{
	int code = msg->code - OFFSET;
	
	if (code < 0 || code >= MNELEM)
		error(Ebadcode);
	if (encode_table[code] == nil)
		error(Enotimpl);
	
	encode_u16int(blk, msg->code);
	encode_table[code](blk, msg);
}

int
B2M(Block* blk, Message* msg)
{
	int code;
	if (blk->lim == blk->rdp)
		return 0;
	
	decode_u16int(blk, &(msg->code));
	code = msg->code - OFFSET;
	
	if (code < 0 || code >= MNELEM)
		error(Ebadcode);
	if (encode_table[code] == nil)
		error(Enotimpl);
	
	decode_table[msg->code - OFFSET](blk, msg);
	return 1;
}
`)

	return methods.Bytes()
}

/*
func srvInterface(desc Description) string {
	inter := new(bytes.Buffer)
	inter.WriteString("type Operations interface {\n")
	for op, _ := range desc {
		// Server callbacks are only for T messages
		if op[0] == 'T' {
			inter.WriteString("\t" + strings.ToUpper(op[1:2]) + op[2:])
			inter.WriteString("(conn *Connection, arg *pepys." + op + ") ")
			inter.WriteString("(*pepys.R" + op[1:] + ", os.Error)\n")
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

	for op, _ := range desc {
		// Server callbacks are only for T messages
		if op[0] == 'T' {
			proc.WriteString("\t\t\tcase \"*pepys." + op + "\":\n")
			proc.WriteString("\t\t\t\tcresp, err = conn.Srv.ops." + strings.ToUpper(op[1:2]) + op[2:])
			proc.WriteString("(conn, op.(*pepys." + op + "))\n")
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
*/

func Generate(desc Description, dir string, rel bool) []string {
	// Generate message constants
	opCodes := "enum {\n"
	for i, op := range desc {
		opCodes = opCodes + "\t" + op.Name + "_code = " + strconv.Itoa(op.Code) + ",\n"
		for j, arg := range op.Args {
			for k, argtype := range arg {				
				switch (argtype) {
					case "string": desc[i].Args[j][k] = "char*";
					case "data": desc[i].Args[j][k] = "Data";
				}
			}
		}
	}
	opCodes = opCodes + "};\n\n"
	
	// Extract headers
	pepysBody, _ := ioutil.ReadFile(dir + "/πp.c")
	pepys := bytes.NewBuffer([]byte(AUTO_GEN))
	
	// Collect parts of pepys main file and write it
	pepys.Write(opBasic())
	pepys.Write(pepysBody)
	pepys.Write(opMethods(desc))
	pepys.Write(opPublicMethods(desc))
	ioutil.WriteFile("πp.c", pepys.Bytes(), 0644)
	
	// Collect parts of pepys header file and write it
	pepysHeader := bytes.NewBuffer([]byte(AUTO_GEN))
	pHF, _ := ioutil.ReadFile(dir + "/πp.h")
	pepysHeader.Write(pHF)
	pepysHeader.WriteString(opCodes)
	pepysHeader.Write(opTypes(desc))
	ioutil.WriteFile("πp.h", pepysHeader.Bytes(), 0644)
	return []string{"πp.h", "πp.c"}
}
