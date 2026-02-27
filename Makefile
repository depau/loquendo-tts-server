GOOS := windows
GOARCH := 386

.PHONY: all clean loqtts_server.exe loqtts_speak.exe
all: loqtts_server.exe loqtts_speak.exe

loqtts_server.exe:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o loqtts_server.exe ./cmd/loqtts_server

loqtts_speak.exe:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o loqtts_speak.exe ./cmd/loqtts_speak

loqtts_server: loqtts_server.exe

loqtts_speak: loqtts_speak.exe

clean:
	rm -f loqtts_server.exe loqtts_speak.exe

