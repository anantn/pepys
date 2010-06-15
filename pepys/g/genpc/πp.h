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

// ACL & Directory info (respresentations subject to future change)
typedef struct Perm Perm;
typedef struct Dirdata Dirdata;
struct Perm {
	u32int	perm;	/* permission bits */
	char*	uid;	/* owner user */
	char*	gid;	/* owner group */
};
struct Dirdata {
	u32int	fref;	/* unique id from server */
	u32int	ftype;	/* file type */
	u64int	vers;	/* version */
	Perm	perm;	/* permissions */
	char	*name;	/* last element of path */
};

// Error messages
char Eperm[] 	= "permission denied";
char Enotdir[] 	= "not a directory";
char Enoauth[] 	= "ramfs: authentication not required";
char Enotexist[]= "file does not exist";
char Einuse[]	= "file in use";
char Eexist[]	= "file exists";
char Eisdir[]	= "file is a directory";
char Enotowner[]= "not owner";
char Eisopen[]	= "file already open for I/O";
char Excl[]		= "exclusive use file already open";
char Ename[]	= "illegal name";
char Eversion[]	= "unknown protocol version";
char Enotempty[]= "directory not empty";
char Ebadfid[]	= "bad fid";
char Enotimpl[] = "not implemented";
char Enomem[]	= "out of memory";
char Ebadrune[] = "bad rune encountered";
char Ebadcode[] = "bad message code";

typedef struct Data	Data;
struct Data {
	u32int	len;
	uchar*	dat;
};

void
error(char *s)
{
	fprint(2, "%s: %s: %r\n", argv0, s);
	exits(s);
}
