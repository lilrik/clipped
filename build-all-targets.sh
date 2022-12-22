# only for publishing the binaries
GOOS=windows GOARCH=amd64 go build -o clipped-windows.exe -ldflags "-s -w" . ; echo "Compiled Windows"
GOOS=linux GOARCH=amd64 go build -o clipped-linux -ldflags "-s -w" . ; echo "Compiled Linux"
GOOS=darwin GOARCH=amd64 go build -o clipped-mac-intel -ldflags "-s -w" . ; echo "Compiled Mac"
mv clipped* bin