DEBUGLDFLAGS = -ldflags=-compressdwarf=false
FINALLDFLAGS = -ldflags "-s -w"
GCFLAGS = -gcflags="all=-N -l"
SRCDIR = .

all: build

debug: $(SRCDIR)/*.go
	go build $(DEBUGLDFLAGS) $(GCFLAGS) $(SRCDIR)

build: $(SRCDIR)/*.go
	go build $(SRCDIR)

test: build
	cp config/real.json config/user.json && \
	./clipped ia 22 && \
	cp config/def.json config/user.json && \
	rm -rf ia

release:
	GOOS=windows GOARCH=amd64 go build -o clipped-windows.exe $(FINALLDFLAGS) $(SRCDIR)
	GOOS=linux GOARCH=amd64 go build -o clipped-linux $(FINALLDFLAGS) $(SRCDIR)
	GOOS=darwin GOARCH=amd64 go build -o clipped-mac-intel $(FINALLDFLAGS) $(SRCDIR)
	mv clipped* bin
