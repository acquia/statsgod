/**
 * Copyright 2015 Acquia, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package statsgod_test

import (
	. "github.com/acquia/statsgod/statsgod"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
	"time"
)

func TestStatsgod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statsgod Suite")
}

var _ = BeforeSuite(func() {
	for i, ts := range testSockets {
		socket := CreateSocket(ts.socketType, ts.socketAddr)
		go socket.Listen(parseChannel, logger, &config)
		BlockForSocket(socket, time.Second)
		sockets[i] = socket
	}
})

var _ = AfterSuite(func() {
	for i, _ := range testSockets {
		sockets[i].Close(logger)
	}
})
