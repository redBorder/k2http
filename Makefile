MKL_RED?=	\033[031m
MKL_GREEN?=	\033[032m
MKL_YELLOW?=	\033[033m
MKL_BLUE?=	\033[034m
MKL_CLR_RESET?=	\033[0m

BIN=      k2http
prefix?=  /usr/local
bindir?=	$(prefix)/bin

build:
	@printf "$(MKL_YELLOW)Building $(BIN)$(MKL_CLR_RESET)\n"
	go build -ldflags "-X main.githash=`git rev-parse HEAD` -X main.version=`git describe --tags --always`" -o $(BIN)

install: build
	@printf "$(MKL_YELLOW)Install $(BIN) to $(bindir)$(MKL_CLR_RESET)\n"
	install $(BIN) $(bindir)

uninstall:
	@printf "$(MKL_RED)Uninstall $(BIN) from $(bindir)$(MKL_CLR_RESET)\n"
	rm -f $(bindir)/$(BIN)

fmt:
	@if [ -n "$$(go fmt)" ]; then echo 'Please run go fmt on your code.' && exit 1; fi

vet:
	@printf "$(MKL_YELLOW)Runing go vet$(MKL_CLR_RESET)\n"
	go vet

test:
	@printf "$(MKL_YELLOW)Runing tests$(MKL_CLR_RESET)\n"
	go test -cover
	@printf "$(MKL_GREEN)Test passed$(MKL_CLR_RESET)\n"

get_dev:
	@printf "$(MKL_YELLOW)Installing deps$(MKL_CLR_RESET)\n"
	go get golang.org/x/tools/cmd/cover
	go get github.com/axw/gocov/gocov
	go get github.com/go-playground/overalls

get:
	@printf "$(MKL_YELLOW)Installing deps$(MKL_CLR_RESET)\n"
	go get -t -v
