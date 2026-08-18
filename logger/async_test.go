package logger_test

import (
	"bytes"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-utils/logger"
)

type intervalWriter struct {
	blockingWriter
	dur time.Duration
}

func (w *intervalWriter) Write(p []byte) (int, error) {
	w.Lock()
	time.Sleep(w.dur)
	n, err := w.buf.Write(p)
	w.Unlock()
	return n, err
}

type blockingWriter struct {
	buf bytes.Buffer
	sync.Mutex
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	w.Lock()
	n, err := w.buf.Write(p)
	w.Unlock()
	return n, err
}

func (w *blockingWriter) Len() int {
	w.Lock()
	n := w.buf.Len()
	w.Unlock()
	return n
}

func (w *blockingWriter) String() string {
	w.Lock()
	s := w.buf.String()
	w.Unlock()
	return s
}

var _ = Describe("Logger", func() {
	Describe("Async Logger", func() {
		It("logs the formatted message to Logger.err at the debug level", func() {
			out := new(bytes.Buffer)
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)
			asyncWriterLogger.Debug("TAG", "some %s info to log", "awesome")
			asyncWriterLogger.Flush()

			expectedContent := expectedLogFormat("TAG", "DEBUG - some awesome info to log")
			Expect(out).To(MatchRegexp(expectedContent))
		})

		It("does not block when its writer is blocked", func() {
			out := new(blockingWriter)
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)

			out.Lock()
			ch := make(chan struct{}, 1)
			go func() {
				for range 10 {
					asyncWriterLogger.Info("TAG", "Make sure we are not just buffering bytes: %s", strings.Repeat("A", 4096))
					asyncWriterLogger.Error("TAG", "Make sure we are not just buffering bytes: %s", strings.Repeat("A", 4096))
				}
				ch <- struct{}{}
			}()
			Eventually(ch).Should(Receive())
			Expect(out.buf.Len()).To(Equal(0))
		})

		It("copies queued log messages", func() {
			const s0 = "ABCDEFGHIJ"
			const s1 = "abcdefghij"

			out := new(blockingWriter)
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)

			out.Lock()
			asyncWriterLogger.Debug("TAG", s0)
			asyncWriterLogger.Debug("TAG", s1)
			out.Unlock()

			Expect(asyncWriterLogger.Flush()).To(Succeed())

			lines := strings.Split(strings.TrimSpace(out.buf.String()), "\n")
			Expect(lines).To(HaveLen(2))
			Expect(lines[0]).To(HaveSuffix(s0))
			Expect(lines[1]).To(HaveSuffix(s1))
		})

		It("continuously flushes queued log messages", func() {
			out := new(blockingWriter)
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)

			out.Lock()
			for range 10 {
				asyncWriterLogger.Debug("TAG", "Queued log message")
			}
			Expect(out.buf.Len()).To(Equal(0))
			out.Unlock()
			Eventually(out.Len).ShouldNot(Equal(0))
		})

		It("flushes with a timeout", func() {
			out := new(blockingWriter)
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)
			asyncWriterLogger.Debug("TAG", "something")

			out.Lock()
			Expect(asyncWriterLogger.FlushTimeout(time.Millisecond * 10)).ToNot(Succeed())

			out.Unlock()
			Expect(asyncWriterLogger.FlushTimeout(time.Millisecond * 10)).To(Succeed())
			Expect(strings.TrimSpace(out.buf.String())).To(HaveSuffix("something"))
		})

		It("flush doesn't block writes", func() {
			const messageCount = 10
			const writeInterval = 10 * time.Millisecond

			out := &intervalWriter{dur: writeInterval}
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)

			// add some messages to the queue
			out.Lock()
			for range messageCount {
				asyncWriterLogger.Debug("NEW", "message")
			}
			out.Unlock()

			go asyncWriterLogger.Flush()

			ch := make(chan struct{}, 1)
			go func() {
				for range messageCount {
					asyncWriterLogger.Debug("NEW", "message")
				}
				ch <- struct{}{}
			}()
			Eventually(ch, time.Second).Should(Receive())
		})

		It("only flushes the current write queue", func() {
			const (
				MessageCount  = 10
				WriteInterval = 10 * time.Millisecond
				Timeout       = WriteInterval * MessageCount * 100
			)

			out := &intervalWriter{dur: WriteInterval}
			asyncWriterLogger := logger.NewAsyncWriterLogger(logger.LevelDebug, out)

			// add some messages to the queue
			out.Lock()
			for range MessageCount {
				asyncWriterLogger.Debug("QUEUED", "queued")
			}
			out.Unlock()

			// add messages faster than the queue can be drained
			tick := time.NewTicker(WriteInterval / 10)
			defer tick.Stop()
			go func() {
				for range tick.C {
					asyncWriterLogger.Debug("NEW", "new")
				}
			}()

			ch := make(chan struct{}, 1)
			go func() {
				asyncWriterLogger.Flush()
				ch <- struct{}{}
			}()

			// we only care that flush returns
			Eventually(ch, Timeout).Should(Receive())
		})

		It("prints the correct prefix during concurrent writes", func() {
			ch := make(chan struct{}, 1)
			go func() {
				testConcurrentPrefix(logger.NewAsyncWriterLogger)
				ch <- struct{}{}
			}()
			Eventually(ch, time.Second*5).Should(Receive())
		})
	})
})
