project=statsgod

version=0.1

PACKAGES=extras statsgod

export PATH := $(abspath ./_vendor/bin):$(PATH)

# Versions of golang prior to 1.4 use a different package URL.
GOM_GROUPS_FLAG="test,test_tip"
ifneq (,$(findstring $(shell go version | grep -e "go[0-9]\+\.[0-9]\+" -o),go1.2 go1.3))
  GOM_GROUPS_FLAG="test,test_legacy"
endif

GOM=$(if $(TRAVIS),$(HOME)/gopath/bin/gom,gom)

all: ${project}

clean:
	rm -rf _vendor

run: ${project}
	go run -race ${project}.go

$(project): deps
	$(GOM) build -o $(GOPATH)/bin/statsgod

deps:
	$(GOM) -groups=$(GOM_GROUPS_FLAG) install

lint: deps
	$(GOM) exec go fmt ./...
	$(GOM) exec go vet -x ./...
	$(GOM) exec golint .
	$(foreach p, $(PACKAGES), $(GOM) exec golint ./$(p)/.; )

test: deps lint
	(test -f coverage.out && "$(TRAVIS)" == "true") || \
		$(GOM) exec go test -covermode=count -coverprofile=coverage.out .
	(test -f statsgod/statsgod.coverprofile && "$(TRAVIS)" == "true") || \
		$(GOM) exec ginkgo -cover=true ./statsgod/.

deb:
	dpkg-buildpackage -uc -b -d -tc

recv:
	go run -race extras/receiver/test_receiver.go

.PHONY: all clean run deps test
