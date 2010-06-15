# After editing the DIRS= list or adding imports to any Go files
# in any of those directories, run:
#
#	./deps.bash
#
# to rebuild the dependency information in Make.deps.

all: install

DIRS=\
	pepys\
	pepys/server\
	pepys/client\

NOTEST=\
	pepys\
	pepys/server\
	pepys/client\

EXAMPLES=\
	pepys/server/examples\
	pepys/client/examples\

clean.dirs: $(addsuffix .clean, $(DIRS))
clean.dirs: $(addsuffix .clean, $(EXAMPLES))
install.dirs: $(addsuffix .install, $(DIRS))
nuke.dirs: $(addsuffix .nuke, $(DIRS))
examples.dirs: $(addsuffix .examples, $(EXAMPLES))

%.clean:
	+cd $* && gomake clean

%.install:
	+cd $* && gomake install

%.nuke:
	+cd $* && gomake nuke

%.test:
	+cd $* && gomake test

%.examples:
	+cd $* && gomake

clean: clean.dirs

install: install.dirs

nuke: nuke.dirs

examples: examples.dirs
