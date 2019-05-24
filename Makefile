all: fake-device

fake-device: *.go
	go build -o fake-device *.go

run: fake-device
	./fake-device

clean:
	rm -f fake-device
