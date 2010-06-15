#include <u.h>
#include <libc.h>
#include <ip.h>

#define Enomem	-1
#define Ebadrune	-2
#define Ebadcode	-3
#define Enotimpl	-4

void
error(int err)
{
	print("Error: %d\n", err);
	exit(err);
}
