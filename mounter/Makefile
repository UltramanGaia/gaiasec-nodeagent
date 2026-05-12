MOUNTER_EXE=mounter

CFLAGS ?= -O2 -Wall -Wextra

.PHONY: all clean build

all: build/$(MOUNTER_EXE)

build:
	mkdir -p build

build/$(MOUNTER_EXE): src/main.c
	$(CC) $(CPPFLAGS) $(CFLAGS) $(LDFLAGS) -o $@ src/main.c

clean:
	rm -rf build
