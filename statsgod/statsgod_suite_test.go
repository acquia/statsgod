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
