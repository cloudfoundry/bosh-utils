package httpclient_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	"syscall"
	"net"
	"os"
)

var _ HTTPClient = &fakes.FakeHTTPClient{}

var _ = Describe("Default HTTP clients", func() {
	It("enables TCP (socket) keepalive with an appropriate interval", func() {
		// to test keepalive, we need a socket. A socket is an _active_ TCP connection to a server.
		// we make our own server, connect to it, and make our assertions against the socket
		laddr := "127.0.0.1:19642" // unlikely-to-be-used port number, unprivileged (1964, Feb, my birth)
		readyToAccept := make(chan bool, 1)

		go func() {
			defer GinkgoRecover()
			defer func(){
				readyToAccept<-true
			}()

			ln, err := net.Listen("tcp", laddr)
			Expect(err).ToNot(HaveOccurred())

			readyToAccept<-true

			_, err = ln.Accept()
			Expect(err).ToNot(HaveOccurred())
		}()

		<-readyToAccept

		client := CreateDefaultClient(nil)
		connection, err := client.Transport.(*http.Transport).Dial("tcp", laddr)
		Expect(err).ToNot(HaveOccurred())

		tcpConn, ok := connection.(*net.TCPConn)
		Expect(ok).To(BeTrue())

		f, err := tcpConn.File()
		Expect(err).ToNot(HaveOccurred())

		sockoptValue, err := syscall.GetsockoptInt(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
		err = os.NewSyscallError("getsockopt", err)
		Expect(err).ToNot(HaveOccurred())
		Expect(sockoptValue).To(Equal(KEEPALIVE_GETSOCKOPT_PLATFORM_INDEPENDENT))

		sockoptValue, err = syscall.GetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, KEEPALIVE_INTERVAL_PLATFORM_INDEPENDENT)
		err = os.NewSyscallError("getsockopt", err)
		Expect(err).ToNot(HaveOccurred())
		Expect(sockoptValue).To(Equal(30))
	})

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
	})

	Describe("CreateDefaultClient", func() {
		It("enforces ssl verification", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(false))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})

	Describe("CreateDefaultClientInsecureSkipVerify", func() {
		It("skips ssl verification", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(true))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			client := CreateDefaultClientInsecureSkipVerify()
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(true))
		})
	})
})
