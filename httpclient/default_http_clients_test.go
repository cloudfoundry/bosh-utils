package httpclient_test

import (
	"crypto/tls"
	"net/http"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
)

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

		It("sets a TLS Session Cache", func() {
			client := CreateDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.ClientSessionCache).To(Equal(tls.NewLRUClientSessionCache(0)))
		})
	})

	Describe("CreateKeepAliveDefaultClient", func() {
		It("enforces ssl verification", func() {
			client := CreateKeepAliveDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify).To(Equal(false))
		})

		It("disables HTTP Transport keep-alive (disables HTTP/1.[01] connection reuse)", func() {
			client := CreateKeepAliveDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).DisableKeepAlives).To(Equal(false))
		})

		It("sets a TLS Session Cache", func() {
			client := CreateKeepAliveDefaultClient(nil)
			Expect(client.Transport.(*http.Transport).TLSClientConfig.ClientSessionCache).To(Equal(tls.NewLRUClientSessionCache(0)))
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

	Describe("ResetDialerContext", func() {
		It("recreates the dialer context to pick up new PROXY config that may have been set during runtime", func() {
			client := CreateDefaultClient(nil)
			clientTransport, _ := client.Transport.(*http.Transport)

			originalProxyEnv := os.Getenv("BOSH_ALL_PROXY")
			os.Setenv("BOSH_ALL_PROXY", "socks5://127.0.0.1:22")
			ResetDialerContext()
			os.Setenv("BOSH_ALL_PROXY", originalProxyEnv)
			clientAfterReset := CreateDefaultClient(nil)
			clientAfterResetTransport, _ := clientAfterReset.Transport.(*http.Transport)
			clientDialContextPointer := reflect.ValueOf(clientTransport.DialContext).Pointer()
			clientAfterResetDialContextPointer := reflect.ValueOf(clientAfterResetTransport.DialContext).Pointer()
			Expect(clientAfterResetDialContextPointer).ToNot(Equal(clientDialContextPointer))
			// don't pollute other tests with a PROXY'd dialer
			os.Unsetenv("BOSH_ALL_PROXY")
			ResetDialerContext()
		})
	})
})
