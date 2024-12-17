package work_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWork(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Work Suite")
}
