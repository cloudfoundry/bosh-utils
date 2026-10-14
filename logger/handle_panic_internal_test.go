package logger

import (
	"bytes"
	"io"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *lockedBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *lockedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

var _ = Describe("HandlePanic", func() {
	var (
		out      *lockedBuffer
		origExit func(int)
		exitCode int
		exited   bool
	)

	BeforeEach(func() {
		out = &lockedBuffer{}
		origExit = osExit
		exited = false
		exitCode = 0
		osExit = func(code int) {
			exited = true
			exitCode = code
		}
	})

	AfterEach(func() {
		osExit = origExit
	})

	// deferHandlePanic mimics a real caller: HandlePanic is the deferred
	// function. recover() only works when called directly in the deferred
	// function, so if the panic is honoured this returns normally; otherwise
	// the panic escapes and the matcher below observes it.
	deferHandlePanic := func(l Logger) {
		defer l.HandlePanic("test-tag")
		panic("boom")
	}

	DescribeTable("recovers a deferred panic, logs it, and exits 2",
		func(newLogger func(LogLevel, io.Writer) Logger) {
			l := newLogger(LevelError, out)

			Expect(func() { deferHandlePanic(l) }).ToNot(Panic())

			Expect(exited).To(BeTrue())
			Expect(exitCode).To(Equal(2))
			Expect(out.String()).To(ContainSubstring("Panic: boom"))
		},
		Entry("sync logger", NewWriterLogger),
		Entry("async logger", NewAsyncWriterLogger),
	)

	It("does nothing when the deferred function sees no panic", func() {
		l := NewWriterLogger(LevelError, out)
		func() { defer l.HandlePanic("test-tag") }()

		Expect(exited).To(BeFalse())
		Expect(out.String()).To(BeEmpty())
	})
})
