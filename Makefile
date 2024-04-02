MKL_RED?=	\033[031m
MKL_GREEN?=	\033[032m
MKL_YELLOW?=	\033[033m
MKL_BLUE?=	\033[034m
MKL_CLR_RESET?=	\033[0m

VERSION?=	$(shell git describe --abbrev=6 --tags HEAD --always | sed 's/-/_/g')

BIN=      k2http
prefix?=  /usr/local
bindir?=	$(prefix)/bin

all: vendor build

build:
	@printf "$(MKL_YELLOW)Building $(BIN)$(MKL_CLR_RESET)\n"
	@go build -ldflags "-X main.version=`git describe --tags --always --dirty=-dev`" -o $(BIN)
	@printf "$(MKL_YELLOW)[BUILD]$(MKL_CLR_RESET)    $(BIN) created\n"
	
get: vendor

install: build
	@printf "$(MKL_YELLOW)Install $(BIN) to $(bindir)$(MKL_CLR_RESET)\n"
	install $(BIN) $(bindir)

uninstall:
	@printf "$(MKL_RED)Uninstall $(BIN) from $(bindir)$(MKL_CLR_RESET)\n"
	rm -f $(bindir)/$(BIN)

test:
	@printf "$(MKL_YELLOW)Running tests$(MKL_CLR_RESET)\n"
	@go test -race  -v
	@printf "$(MKL_GREEN)Test passed$(MKL_CLR_RESET)\n"

coverage:
	@printf "$(MKL_YELLOW)Computing coverage$(MKL_CLR_RESET)\n"
	@go test -covermode=count -coverprofile=batch.part
	@echo "mode: count" > coverage.out
	@grep -h -v "mode: count" *.part >> coverage.out
	@go tool cover -func coverage.out

vendor:
	@go mod tidy
	@go mod vendor

clean:
	rm -f $(BIN) $(SNORT_CONTROL)
	rm -rf vendor/

rpm: clean	
	$(MAKE) -C packaging/rpm