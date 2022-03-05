GOOS=windows GOARCH=amd64 go build -o clipped-windows.exe -ldflags "-s -w" . ; echo "windows"
GOOS=linux GOARCH=amd64 go build -o clipped-linux -ldflags "-s -w" . ; echo "linux"
GOOS=darwin GOARCH=amd64 go build -o clipped-mac-intel -ldflags "-s -w" . ; echo "mac"
mv clipped* bin
