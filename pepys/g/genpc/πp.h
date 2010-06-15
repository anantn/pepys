// General constants
enum {
	Nmsgs	= 16,	// default max number of messages per packet
	Iohdrsz	= 24,	// the non-data size of the Twrite messages
	Msize	= 8192 + Iohdrsz, // default message size
	Port	= 564	// default port for file servers
};

// Special values
enum {
	Notag	= ~0,
	Nofid	= ~0,
	Nouid	= ~0
};

// Flags for the mode field in Topen & Tcreate messages
enum {
	Oread   = 0x1,
	Owrite  = 0x2,
	Ordwr   = Oread | Owrite,
	Oexec   = 0x4 | Oread,
	Otrunc  = 0x10,
	Ocexec  = 0x20,
	Orclose = 0x40,
};

// Ftype flags
enum {
	Fdir	= 0x1,
	Fappend	= 0x2,
	Fversioned	= 0x4
};

// Permission bits
enum {
	Prmread		= 0x4,
	Prmwrite	= 0x2,
	Prmexec		= 0x1
};

// ACL - respresentation subject to future change
typedef struct Perm Perm;
struct Perm {
	u32int	perm;
	char*	uid;
	char*	gid;
};

typedef struct Data	Data;
struct Data {
	u32int	len;
	uchar*	dat;
};

