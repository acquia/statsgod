project=statsgod

all: deps ${project}

clean:
	rm -rf $(bindir)

run: ${project}
	go run -race ${project}.go

$(project): deps
	go build ./...

deps:
	go get github.com/kr/godep github.com/golang/lint/golint

test: deps
	golint ./

recv:
	go run -race receiver/test_receiver.go

.PHONY: all clean run deps test
