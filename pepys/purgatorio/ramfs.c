#include <u.h>
#include <libc.h>
#include <ip.h>
#include "include.h"

void
ramfs_proto(Block* blk, Message* msg)
{
	
}

void
ramfs_session(Block* blk, Message* msg)
{
	
}

void
ramfs_attach(Block* blk, Message* msg)
{
	
}

void
ramfs_flush(Block* blk, Message* msg)
{
	
}

void
ramfs_open(Block* blk, Message* msg)
{
	
}

void
ramfs_create(Block* blk, Message* msg)
{
	
}

void
ramfs_read(Block* blk, Message* msg)
{
	
}

void
ramfs_write(Block* blk, Message* msg)
{
	
}

void
ramfs_remove(Block* blk, Message* msg)
{
	
}

void
ramfs_clunk(Block* blk, Message* msg)
{
	
}

/* FIXME: Move this to the generic πp library as all servers will do it.
   Then we can make the function signatures message specific instead of
   (Block*, Message*) for all. (easily done in generator)
 */
static void
(*ramfs_table[])(Block*, Message*) = {
	ramfs_proto,
	nil,
	ramfs_session,
	nil,
	ramfs_attach,
	nil,
	nil,
	nil,
	ramfs_flush,
	nil,
	ramfs_open,
	nil,
	ramfs_create,
	nil,
	ramfs_read,
	nil,
	ramfs_write,
	nil,
	ramfs_remove,
	nil,
	ramfs_clunk,
	nil
};

/* Read block from file descriptor */
void
fd2B(Block* blk, int fd)
{
	int sz;
	char size[4];
	
	/* read group size */
	if (read(fd, size, 4) != 4)
		exits("could not read group");
	sz = nhgetl((void*) size);
	
	/* allocate request */
	blk->wrp = nil;
	blk->beg = (uchar*) calloc(sz, 1);
	blk->rdp = blk->beg;
	blk->lim = blk->beg + sz;
	
	/* read group */
	if (read(fd, blk->rdp, sz) != sz)
		exits("could not read entire message");
}

/* Write block to file descriptor */
void
B2fd(Block* blk, int fd)
{
	int sz;
	char size[4];
	
	sz = (int)(blk->wrp - blk->beg);
	hnputl((void*) size, sz);
	
	if (write(fd, size, 4) != 4)
		exits("could not write response size");
	
	if (write(fd, blk->beg, sz) != sz)
		exits("could not write response");
}

/* Single threaded ramfs server. Grab a group of messages,
   process each one & send response
 */
void
main()
{
	Message msg;
	Block req, res;
	int dfd, acfd, lcfd;
	char adir[40], ldir[40];
	
	acfd = announce("tcp!*!564", adir);
	if (acfd < 0)
		exits("could not announce ramfs");
	
	for (;;) {
		lcfd = listen(adir, ldir);
		if (lcfd < 0)
			exits("could not listen");
			
		/* accept & read incoming data */
		dfd = accept(lcfd, ldir);
		if (dfd < 0)
			exits("could not accept");
		
		/* read request */
		fd2B(&req, dfd);
		
		/* allocate response */
		res.rdp = nil;
		res.beg = (uchar*) calloc(4096, 1);
		res.wrp = req.beg;
		res.lim = req.beg + 4096;
		
		/* iterate over messages & collect responses */
		while (B2M(&req, &msg)) {
			if (!ramfs_table[msg.code]) {
				/* send rerror & process no more */
				msg.code = Rerror_code;
				msg.rerror.ename = Enotimpl;
				M2B(&res, &msg);
				break;
			}
			ramfs_table[msg.code](&res, &msg);
		}
		
		/* send response */
		B2fd(&res, dfd);
	}
}