# Copyright 2014 Acquia, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

project=statsgod

version=0.1

PACKAGES=extras statsgod

export PATH := $(abspath ./_vendor/bin):$(PATH)
export GOPATH := $(abspath ./_vendor):$(GOPATH)

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

deps: clean
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

run-container:
	docker run -v $(shell pwd)/:/usr/share/go/src/github.com/acquia/statsgod -it statsgod /bin/bash

.PHONY: all clean run deps test
