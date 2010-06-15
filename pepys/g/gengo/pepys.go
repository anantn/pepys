package pepys

import "io"
import "os"
import "bytes"
import "reflect"
import "encoding/binary"

// General constants
const(
	Nmsgs	= 16,	// default max number of messages per packet
	Iohdrsz	= 24,	// the non-data size of the Twrite messages
	Msize	= 8192 + Iohdrsz, // default message size
	Port	= 564	// default port for file servers
)

// Special values
const (
	Notag	= ~0,
	Nofid	= ~0,
	Nouid	= ~0
)

// Flags for the mode field in Topen messages
const (
	Oread   = 0x1,
	Owrite  = 0x2,
	Ordwr   = Oread | Owrite,
	Oexec   = 0x4 | Oread,
	Otrunc  = 0x10,
	Ocexec  = 0x20,
	Orclose = 0x40,
)

type data []byte
type Packet struct {
	Msgs []interface{}
}

// FIXME: error checking for all the following methods
func encodeString(val string, buf io.Writer) os.Error {
	byt := []byte(val)
	length := uint16(len(byt))
	
	binary.Write(buf, binary.BigEndian, length)
	buf.Write(byt)
	return nil
}
func decodeString(buf io.Reader) string {
	var length uint16
	binary.Read(buf, binary.BigEndian, &length)
	
	value := make([]byte, length)
	buf.Read(value)
	return string(value)
}

func encodeData(val data, buf io.Writer) os.Error {
	length := uint32(len(val))
	
	binary.Write(buf, binary.BigEndian, length)
	buf.Write(val)
	return nil
}
func decodeData(buf io.Reader) data {
	var length uint32
	binary.Read(buf, binary.BigEndian, &length)
	
	value := make([]byte, length)
	buf.Read(value)
	return value
}
