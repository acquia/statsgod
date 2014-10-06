project=statsgod

PACKAGES=generator receiver

export PATH := $(abspath ./_vendor/bin):$(PATH)

ifeq ($(TRAVIS), true)
GOM=$(HOME)/gopath/bin/gom
else
GOM=gom
endif

all: deps ${project}

clean:
	rm -rf _vendor ${project}

run: ${project}
	go run -race ${project}.go

$(project): deps
	$(GOM) build

deps:
	$(GOM) -test install

test: deps
	$(GOM) exec go fmt ./...
	$(GOM) exec go vet -x ./...
	$(GOM) exec golint $(PACKAGES:%=./%)
	$(GOM) exec go test -covermode=count -coverprofile=coverage.out .

recv:
	go run -race receiver/test_receiver.go

.PHONY: all clean run deps test
