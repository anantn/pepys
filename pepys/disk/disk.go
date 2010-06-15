package disk

import (
	"os"
	"fmt"
	"gob"
	"time"
	"log"
)

const K = 1 << 10
const M = 1 << 20
const G = 1 << 30
const bbits = 9
const bsize = 1 << bbits // block size
const asize = G          // arena size
const abits = 30

/*
 *	Sizes in bytes
 */
const DiskaddrSize = 8 // uint64
const VidSize = 12
const IndexelemSize = 32 // Vid+Metaaddr

/*
 * A file system that can store 10 million files needs
 * ½ GB of index space.  We can keep this in memory in a balanced
 * tree.  It's important that we're able to find a file quickly, but
 * also that we can equally quickly find that a file isn't there.
 */

type ServerID uint64
type FileID uint64 // Not to be confused with Fid
type VersionID int64

type Xid struct {
	s ServerID
	f FileID
}

type Vid struct {
	Xid
	v VersionID
}

type Indexelem struct {
	/*
	 * Used to find files on disk;  the
	 * entire index is read into memory when pepysfs
	 * starts up; it is written to disk every time
	 * a Chunk fills up.
	 * This describes a single entry of the index on disk.
	 * The in-memory data structure is Treeelem.
	 */
	Vid   int64
	daddr uint64 /* where to start reading */
	doff  int    /* where the interesting bit starts */
	dlen  int    /* disk metadata size — if 0, not on disk */
}

type Super struct {
	magic      string
	time       int64     /* Time last written/changed */
	fstcurrent bool      /* true: first or false: second copy was last written */
	size       uint64    /* size of the disk */
	vidx       [2]uint64 /* Vid index addresses */
	vids       uint64    /* Vid index size */
	atab       uint64    /* Arena tab address */
	atas       uint64    /* Arena tab size */
	arenas     uint64    /* first Arena address */
	asize      uint64    /* Arena size */
	narena     uint32    /* number of arenas */
	bsize      int       /* block size */
	rootvid    Vid       /* vid of root */

	/* Dynamic stuff */
	lastsnap uint64 /* Place of last snapshot */
}

type Disk struct {
	name  string
	f     *os.File
	size  uint64
	super *Super
	log   *log.Logger
	dec   *gob.Decoder
	enc   *gob.Encoder
}

type Metaaddr struct {
	/*
	 * Location of file metadata on disk.
	 * Many files are written in their entirety:
	 * data followed by metadata.  In those cases,
	 * it is often a good idea to read data and metadata
	 * in one fell swoop.  When this is the case, offset
	 * indicates where the metadata starts.  Length
	 * always indicates how much to read, starting at
	 * uint64, to get all of the metadata.
	 * The doff field, when non-zero, indicates that
	 * data is included in the Fileaddr.
	 */
	daddr uint64 // where to start reading
	doff  int32  // where the interesting bit starts
	dlen  uint16 // disk metadata size — if 0, not on disk
	maddr uint   // memory address
	mlen  uint16 // mem metadata size — if 0, not in memory
}

/*
 *	What if we allocated everything on disk at 512-byte
 *	boundaries?
 *	• Metadata for a file always starts at a 512-byte
 *	  boundary and occupies a multiple of 512 bytes.
 *	• File data starts at a 512-byte boundary and cooupies
 *	  a multiple of 512-byte miniblocks
 *	• In memory (not on disk!), we use block lists which,
 *	  again, consist of 512-byte entities:
 *	  A (long) length, in bytes, followed by up to 126
 *	  pointers to miniblocks, optionally followed by the
 *	  address of the next 512-byte miniblock with (up to 127)
 *	  pointers.
 *	• On the disk, we can use "ranges": pairs of {diskaddress,
 *	  length} that point to multiple blocks of 512-bytes
 *	  miniblocks.
 *	• We'll take care to make the number "512" a variable that
 *	  we can change later.  As jmk says, it's very unlikely we'll
 *	  make this number any smaller — i.e. it's likely that it
 *	  needs to be bigger).
 */

/* All data on disk will have network byte order, using the Plan 9
 * convention of
 *	uchar	1 bytes
 *	ushort	2 bytes
 *	uint32	4 bytes
 *	uint64	8 bytes
 * with alignment at multiples of the size
 *
 * Disk layout, with the following general areas:
 *
 * Config 8MB
 *	8 MB, fillable with text that describes the configuration
 *	for this file system (a la venti/conf), or with a small
 *	9fat file system, or even with a packfs thingie that allows
 *	booting off this partition
 * Super 8KB
 *	Superblock (size TBD), with where everything is and how big
 *	things are on this particular disk (the config section lists
 *	the disks used by a file server; each disk has its own
 *	superblock, freelist, etc. etc.
 *	Location of last snapshot in log.
 *	We also need to record things "to be done" to outside
 *	servers (e.g., while being disconnected).
 *	Can these just be in the log?
 *	First thing will be the pepysmagic string;
 * VidIndex 8K (256 files) to 128 MB (2M files)
 *	List of Vids and their disk addresses (ordered?) to be
 *	read into the AVL tree that Pepysfs uses to find files
 *	on disk.
 * ArenaTab 1K/arena
 *	Map of the arenas forming the log and the list of free
 *	ones.  Should we keep track here of the fill ratio of arenas?
 * Arenas
 *	The actual data on the disk
 * Second copy of VidIndex.  Alternately written with main one
 * Second copy of Super.
 */

const Configsize = 0      /* Will be 8MB, 0 for initial testing */
const Supersize = 8 * K   /* Size of the superblock */
const MinVidxsize = 8 * K /* Minimum allowable index size */
const MinAtabsize = 8 * K /* Minimum arena table size */
const MinArenas = 4       /* Minimum number of arenas */
const MinDisk = Configsize + MinAtabsize + 2*Supersize + 2*MinVidxsize

func overlap(a1 uint64, s1 uint64, a2 uint64, s2 uint64) bool {
	return a1 < a2 && a1+s1 > a2 || a2 < a1 && a2+s2 > a1 || a1 == a2 && s1 != 0 && s2 != 0
}

func (s *Super) IsSane() os.Error {
	// Check the sanity of the superblock

	if s.time < 0 || s.time > time.Nanoseconds() {
		return os.NewError(fmt.Sprintf("Bad time %d", s.time))
	}
	if s.vidx[0] < Configsize+Supersize {
		return os.NewError(fmt.Sprintf("Index overlaps conf/superblock (%#x)", s.vidx[0]))
	}
	if s.atab < Configsize+Supersize {
		return os.NewError(fmt.Sprintf("Arena table overlaps conf/superblock (%#x)", s.atab))
	}
	if s.arenas < Configsize+Supersize {
		return os.NewError(fmt.Sprintf("Arenas overlap conf/superblock (%#x)", s.arenas))
	}
	if overlap(s.vidx[0], s.vids, s.atab, s.atas) {
		return os.NewError(fmt.Sprintf("Arena table and Vid Index overlap"))
	}
	if overlap(s.vidx[0], s.vids, s.arenas, s.asize*uint64(s.narena)) {
		return os.NewError(fmt.Sprintf("Vid Index and arenas overlap"))
	}
	if overlap(s.atab, s.atas, s.arenas, s.asize*uint64(s.narena)) {
		return os.NewError(fmt.Sprintf("Vid Index and arenas overlap"))
	}
	if s.arenas+s.asize*uint64(s.narena) > s.size {
		return os.NewError(fmt.Sprintf("Arenas don't fit"))
	}
	if s.atab+s.atas > s.size {
		return os.NewError(fmt.Sprintf("Arena table doesn't fit"))
	}
	if s.vidx[0]+s.vids > s.size {
		return os.NewError(fmt.Sprintf("Vid Index doesn't fit"))
	}
	return nil
}

func New(name string) (*Disk, os.Error) {
	var err os.Error
	var size int64

	disk := new(Disk)
	disk.log = log.New(os.Stderr, nil, name, log.Lok|log.Ltime)
	disk.name = name
	if disk.f, err = os.Open(name, os.O_RDWR, 0); err != nil {
		return nil, err
	}
	// Find end of disk
	if size, err = disk.f.Seek(0, 2); err != nil {
		return nil, err
	}
	disk.size = uint64(size)
	if disk.size < MinDisk {
		err = os.NewError(fmt.Sprintf("readsuper: %s: too small %d < %d", disk.name, disk.size, MinDisk))
		return nil, err
	}
	disk.dec = gob.NewDecoder(disk.f)
	disk.enc = gob.NewEncoder(disk.f)
	return disk, nil
}

func (disk *Disk) ReadSuper() os.Error {
	disk.log.Logf("readsuper [%d (%#x)] at %d (%#x)\n",
		disk.size, disk.size, Configsize, Configsize)

	pos, err := disk.f.Seek(Configsize, 0)
	if pos != Configsize || err != nil {
		return os.NewError(fmt.Sprintf("readsuper: %s: seek %d: %s, %s", disk.name, disk.size, Configsize, err.String()))
	}

	s := make([]*Super, 2)

	if err = disk.dec.Decode(s[0]); err != nil {
		return err
	}

	if err = s[0].IsSane(); err != nil {
		return err
	}
	disk.log.Logf("first superblock IsSane\n")

	spos := int64(s[0].size-Supersize) & int64(^(s[0].bsize - 1))

	disk.log.Logf("readsuper %s[%d (%#x)] at %d (%#x)\n",
		disk.name, disk.size, disk.size, spos, spos)

	pos, err = disk.f.Seek(spos, 0)
	if spos != pos || err != nil {
		return os.NewError(fmt.Sprintf("readsuper: %s: seek %d - %d", disk.name, disk.size, pos))
	}

	if err = disk.dec.Decode(s[1]); err != nil {
		return err
	}
	if err = s[1].IsSane(); err != nil {
		return err
	}
	disk.log.Logf("second superblock IsSane\n")

	fstcurrent := s[0].time > s[1].time
	if fstcurrent {
		disk.super = s[0]
		disk.log.Logf("first superblock with time %s\n", s[0].time)
	} else {
		disk.super = s[1]
		disk.log.Logf("second superblock with time %s\n", s[1].time)
	}
	disk.super.fstcurrent = fstcurrent
	return nil
}

func (disk *Disk) WriteSuper() os.Error {
	var spos int64

	if disk.super.fstcurrent {
		// first is current, write to second:
		spos = int64(disk.size-Supersize) & int64(^(disk.super.bsize - 1))
	} else {
		// second is current, write to first:
		spos = int64(Configsize)
	}
	pos, err := disk.f.Seek(spos, 0)
	if pos != spos || err != nil {
		return os.NewError(fmt.Sprintf("writesuper: %s: seek %d: %s, %s",
			disk.name, disk.size, spos, err.String()))
	}
	if err = disk.enc.Encode(disk.super); err != nil {
		return err
	}
	return nil
}


func (disk *Disk) CreateSuper() os.Error {
 	s := new(Super)
	disk.log.Logf("createsuper %s, %d (0x%x)\n", disk.name, disk.size, disk.size)
	s.size = disk.size
	if s.size < MinDisk {
		return os.NewError(fmt.Sprintf("createsuper: %s: too small %d < %d",
			disk.name, disk.size, MinDisk))
	}
	x := disk.size - Configsize - 2*Supersize
	n := x / asize

	disk.log.Logf("disk will hold approx. %d arenas\n", n)

	x = asize * n / 20000

	disk.log.Logf("disk will hold up to %d files\n", x)

	s.vidx[0] = Configsize + Supersize

	// Size is #of files*Index size rounded up to nearest MB
	s.vids = x*IndexelemSize + M - 1&M - 1

	disk.log.Logf("Index size 0x%x = %d\n", s.vids, s.vids)

	s.atab = s.vidx[0] + s.vids
	s.atas = n*K + M - 1&M - 1

	disk.log.Logf("Arena tab 0x%x = %d\n", s.atas, s.atas)

	s.arenas = s.atab + s.atas
	x = s.size - s.vids - Supersize - s.arenas
	s.asize = asize
	s.narena = uint32(x / asize)
	s.bsize = bsize

	disk.log.Logf("configuring %d arenas of %d KB\n", s.narena, s.asize/K)

	if err := s.IsSane(); err != nil {
		return err
	}
	disk.super = s
	return disk.WriteSuper()
}

/*
 * Log structure
 *
 *	uchar	type
 *	ushort	length (length == 0: no data)
 *	---------------------------- 512-byte block boundary
 *	[data]	k*512 bytes (k = 0, ... 128)
 *	---------------------------- 512-byte block boundary
 *	metadata, k*512 - 3 bytes (k = 1, ...)
 *	(type, length)
 *	---------------------------- 512-byte block boundary
 *	...
 *
 *	Metadata:
 *		ushort		length
 *		uint64+uint32	Vid
 *		uchar		type (e.g., dir, file, tmp, ...)
 *		uchar		state (e.g., dirty, cached, ...)
 */
