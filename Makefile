.PHONY: help default tools clean

default: help

help:
	@printf "Available targets:\n\n"
	@printf "  make help    Print this usage message.\n"
	@printf "  make clean   Remove built binaries from ./bin.\n"
	@printf "\n"

clean:
	@rm -rf bin
