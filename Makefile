build:
	GOOS=linux CGO_ENABLED=0 go build -a -o wget .