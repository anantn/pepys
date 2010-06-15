#include <u.h>
#include <libc.h>
#include "πp.h"

#define MEM 8192

// Test
void main()
{
	Block blk;
	Message msg;
	
	// Allocate some memory for Packet
 	uchar* ptr = (uchar*) calloc(MEM, 1);

	// Set locations
	blk.beg = ptr;
	blk.rdp = blk.beg;
	blk.wrp = blk.beg;
	blk.lim = blk.beg + MEM;
	
	// Normally you wouldn't send proto/session/attach in one packet!
	
	// Tproto
	msg.code = Tproto_code;
	msg.tproto.msize = Msize;
	msg.tproto.nmsgs = Nmsgs;
	msg.tproto.options = "9p+lease";
	M2B(&blk, &msg);
	
	// Tsession
	msg.code = Tsession_code;
	msg.tsession.csid = 0x123456;
	msg.tsession.uname = "testuser";
	msg.tsession.afid = Nofid;
	M2B(&blk, &msg);
	
	// Tattach
	msg.code = Tattach_code;
	msg.tattach.fid = 1;
	msg.tattach.afid = Nofid;
	msg.tattach.uname = "testuser";
	msg.tattach.aname = "/";
	M2B(&blk, &msg);
	
	// Pretend to send over network
	print("Sending Tproto/Tsession/Tattach... sent!\n");
	// send_blk_mem(blk, memory);
	// Pretend to recieve from network (read from memory instead)
	// blk = recv_blk_mem(memory);
	blk.lim = blk.wrp;
	
	// Read messages back one by one
	print("Received messages... parsing!\n");

	memset(&msg, 0, sizeof(Message));
	while (B2M(&blk, &msg)) {
		switch (msg.code) {
			case Tproto_code:
				print("Tproto - %d %d %s\n", msg.tproto.msize, msg.tproto.nmsgs, msg.tproto.options);
				break;
			case Tsession_code:
				print("Tsession - %d %s %d\n", msg.tsession.csid, msg.tsession.uname, msg.tsession.afid);
				break;
			case Tattach_code:
				print("Tattach - %d %d %s %s\n", msg.tattach.fid, msg.tattach.afid, msg.tattach.uname, msg.tattach.aname);
				break;
		}
		memset(&msg, 0, sizeof(Message));
	}
	print("Test finished!\n");
}
