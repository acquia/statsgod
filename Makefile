project=statsgod

version=0.1

PACKAGES=extras statsgod

export PATH := $(abspath ./_vendor/bin):$(PATH)

ifeq ($(TRAVIS), true)
GOM=$(HOME)/gopath/bin/gom
else
GOM=gom
endif

all: deps ${project}

clean:
	rm -rf _vendor 

run: ${project}
	go run -race ${project}.go

$(project): deps
	$(GOM) build -o $(GOPATH)/bin/statsgod

deps:
	$(GOM) -test install

lint: deps
	$(GOM) exec go fmt ./...
	$(GOM) exec go vet -x ./...
	$(GOM) exec golint .
	$(foreach p, $(PACKAGES), $(GOM) exec golint ./$(p)/.; )

test: deps lint
	$(GOM) exec go test -covermode=count -coverprofile=coverage.out .
	$(GOM) exec ginkgo -cover=true ./statsgod/.

deb: 
	dpkg-buildpackage -b -d -tc

recv:
	go run -race extras/receiver/test_receiver.go

.PHONY: all clean run deps test
