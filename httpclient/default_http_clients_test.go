package httpclient_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	"net"
	"syscall"
	"os"
	"time"
)

var _ HTTPClient = &fakes.FakeHTTPClient{}

var _ = Describe("Default HTTP clients", func() {
	Describe("DefaultClient", func() {
		It("is a singleton http client", func() {
			client := DefaultClient
			Expect(client).ToNot(BeNil())
			Expect(client).To(Equal(DefaultClient))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			var client *http.Client
			client = DefaultClient

			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})

		It("enables TCP (socket) keepalive with an appropriate interval", func() {
			// to test keepalive, we need a socket. A socket is an _active_ TCP connection to a server.
			// we make our own server, connect to it, and make our assertions against the socket
			laddr := "127.0.0.1:19642" // unlikely-to-be-used port number, unprivileged (1964, Feb, my birth)
			go func() {
				ln, err := net.Listen("tcp", laddr)
				if err != nil {
					panic("I couldn't Listen() to " + laddr)
				}
				_, err = ln.Accept()
				if err != nil {
					panic("I couldn't Accept() " + laddr)
				}
			}()
			// we give the server a head-start before we try to connect to it
			time.Sleep(1 * time.Millisecond)
			client := DefaultClient
			connection, err := client.Transport.(*http.Transport).Dial("tcp", laddr)
			if err != nil {
				panic("I couldn't Dial() " + laddr)
			}

			tcpConn, ok := connection.(*net.TCPConn)
			if !ok {
				panic("I couldn't cast to TCPConn")
			}

			f, err := tcpConn.File()
			if err != nil {
				panic("I couldn't File()")
			}

			sockoptValue, err := syscall.GetsockoptInt(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
			err = os.NewSyscallError("getsockopt", err)
			if err != nil {
				panic("I couldn't GetsockoptInt()")
			}
			Expect(sockoptValue).To(Equal(syscall.SO_KEEPALIVE))

			sockoptValue, err = syscall.GetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, syscall.TCP_KEEPALIVE)
			err = os.NewSyscallError("getsockopt", err)
			if err != nil {
				panic("I couldn't GetsockoptInt()")
			}
			Expect(sockoptValue).To(Equal(30))
		})
	})

	Describe("CreateDefaultClient", func() {
		It("enforces ssl verification", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(false))
		})

		It("disables keep alive", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})

	Describe("CreateDefaultClientInsecureSkipVerify", func() {
		It("skips ssl verification", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(true))
		})

		It("disables keep alive", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})
})
