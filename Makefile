DEBUGLDFLAGS = -ldflags=-compressdwarf=false
FINALLDFLAGS = -ldflags "-s -w"
GCFLAGS = -gcflags="all=-N -l"

all: build

debug: *.go
	go build $(DEBUGLDFLAGS) $(GCFLAGS) .

build: *.go
	go build .

test: build
	cp config/real.json config/user.json && \
	./clipped ia 22 && \
	cp config/def.json config/user.json && \
	rm -rf ia

release:
	GOOS=windows GOARCH=amd64 go build -o clipped-windows.exe $(FINALLDFLAGS) .
	GOOS=linux GOARCH=amd64 go build -o clipped-linux $(FINALLDFLAGS) .
	GOOS=darwin GOARCH=amd64 go build -o clipped-mac-intel $(FINALLDFLAGS) .
	mv clipped* bin
