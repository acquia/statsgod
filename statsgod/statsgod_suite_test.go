package statsgod_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStatsgod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statsgod Suite")
}
