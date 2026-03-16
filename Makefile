NAME = gonn

SRCS = \
	document.go  \
	index.go  \
	roff.go  \
	server.go  \
	template.go  \
	utils.go  \
	version.go

CMD_SRCS = cmd/main.go

$(NAME): $(SRCS) $(CMD_SRCS)
	CGO_ENABLED=0 go build -o $@ $(CMD_SRCS)


$(NAME).debug: $(SRCS) $(CMD_SRCS)
	CGO_ENABLED=1 go build -race -o $@ $(CMD_SRCS)


clean:
	go clean -x -i
	rm -f $(NAME) $(NAME).debug

check:
	##go test -x --count=1 ./...
	go test --count=1 ./...
