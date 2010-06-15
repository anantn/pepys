include $(GOROOT)/src/Make.$(GOARCH)

TARG=timefs
OFILES=$(TARG:%=%.$O)

all: $(TARG)

$(TARG): %: %.$O
	$(LD) -o $@ $<

$(OFILES): %.$O: %.go Makefile
	$(GC) -o $@ $<

clean:
	rm -f *.[$(OS)] $(TARG) $(CLEANFILES)
