#include "πp.h"

typedef struct Fid Fid;
typedef struct Ram Ram;

struct Fid {
	int		fid;
	Ram		*ram;
	Dirdata	dat;
	Fid		*next;
};

struct Ram {
	u64int	size;
	u32int	index;
	u64int	atime; /* mtime is stored in Dirdata */
	char	*muid;
	char	*data;
};

