static void
encode_string(Block* blk, char* val)
{
	uint sz;
	Rune len;
	char l[UTFmax];
	
	if (val == nil) {
		check_enc(blk, 1);
		*blk->wrp++ = '\0';
		return;
	}
	
	len = strlen(val);
	sz = runetochar(l, &len);
	check_enc(blk, len+sz);
	memmove(blk->wrp, l, sz);
	blk->wrp += sz;
	memmove(blk->wrp, val, len);
	blk->wrp += len;
}
static void
decode_string(Block* blk, char** val)
{
	uint sz;
	Rune len;
	
	if (*blk->rdp == '\0') {
		*val = nil;
		blk->rdp++;
		return;
	}

	sz = chartorune((Rune*) &len, (char*) blk->rdp);
	if (len == Runeerror)
		error(Ebadrune);

	check_dec(blk, sz + len);
	*val = (char*)blk->rdp;
	memmove(blk->rdp, blk->rdp+sz, len);
	blk->rdp[len] = '\0';
	blk->rdp += sz + len;
}

static void
encode_data(Block* blk, Data* dat)
{
	check_enc(blk, dat->len + 4);
	encode_u32int(blk, dat->len);
	
	/* Don't memmove data that's already in place (see prep_rread) */
	if (blk->wrp != dat->dat && dat->len > 0)
		memmove(blk->wrp, dat->dat, dat->len);
	blk->wrp = blk->wrp + dat->len;
}
static void
decode_data(Block* blk, Data* dat)
{
	decode_u32int(blk, &(dat->len));
	check_dec(blk, dat->len);
	dat->dat = blk->rdp;
	blk->rdp = blk->rdp + dat->len;
}

static void
check_enc(Block* blk, int size)
{
	uint avail = blk->lim - blk->wrp;
	if (avail < size) {
		error(Enomem);
	}
}

static void
check_dec(Block* blk, int size)
{
	uint avail = blk->wrp - blk->rdp;
	if (avail < size) {
		error(Enomem);
	}
}
