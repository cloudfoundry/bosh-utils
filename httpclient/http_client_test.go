package httpclient_test

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
)

var _ = Describe("HTTPClient", func() {
	var (
		httpClient             *HTTPClient
		fastErroringHttpClient *HTTPClient
		server                 *ghttp.Server
		logger                 loggerfakes.FakeLogger
		opts                   Opts
	)

	JustBeforeEach(func() {
		logger = loggerfakes.FakeLogger{}

		httpClient = NewHTTPClientOpts(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            nil,
					InsecureSkipVerify: true,
				},

				Proxy: http.ProxyFromEnvironment,

				Dial: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 0,
				}).Dial,

				TLSHandshakeTimeout: 10 * time.Second,
				DisableKeepAlives:   true,
			},
		}, &logger, opts)

		fastErroringHttpClient = NewHTTPClientOpts(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            nil,
					InsecureSkipVerify: true,
				},

				Proxy: http.ProxyFromEnvironment,

				Dial: (&net.Dialer{
					Timeout:   1 * time.Millisecond,
					KeepAlive: 0,
				}).Dial,

				TLSHandshakeTimeout: 1 * time.Millisecond,
				DisableKeepAlives:   true,
			},
		}, &logger, opts)

		server = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Post/PostCustomized", func() {
		It("makes a POST request with given payload", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path"),
					ghttp.VerifyBody([]byte("post-request")),
					ghttp.RespondWith(http.StatusOK, []byte("post-response")),
				),
			)

			url := server.URL() + "/path"

			response, err := httpClient.Post(url, []byte("post-request"))
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("post-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("allows to override request including payload", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/path"),
					ghttp.VerifyBody([]byte("post-request-override")),
					ghttp.VerifyHeader(http.Header{
						"X-Custom": []string{"custom"},
					}),
					ghttp.RespondWith(http.StatusOK, []byte("post-response")),
				),
			)

			url := server.URL() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
				r.Body = io.NopCloser(bytes.NewBufferString("post-request-override"))
				r.ContentLength = 21
			}

			response, err := httpClient.PostCustomized(url, []byte("post-request"), setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("post-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("redacts passwords from error message", func() {
			url := "http://foo:bar@10.10.0.0/path"

			_, err := fastErroringHttpClient.PostCustomized(url, []byte("post-request"), func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("http://foo:<redacted>@10.10.0.0/path"))
		})

		It("redacts passwords from error message for https calls", func() {
			url := "https://foo:bar@10.10.0.0/path"

			_, err := fastErroringHttpClient.PostCustomized(url, []byte("post-request"), func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("https://foo:<redacted>@10.10.0.0/path"))
		})

		Describe("httpclient opts", func() {
			BeforeEach(func() {
				opts = Opts{NoRedactUrlQuery: true}
			})

			It("does not redact every query param from endpoint for https calls", func() {

				url := "https://oauth-url?refresh_token=abc&param2=xyz"

				httpClient.PostCustomized(url, []byte("post-request"), func(r *http.Request) {}) //nolint:errcheck
				_, _, args := logger.DebugArgsForCall(0)
				Expect(args[0]).To(ContainSubstring("param2=xyz"))
				Expect(args[0]).To(ContainSubstring("refresh_token=abc"))
			})

			Context("httpclient has been configured to redact query parmas", func() {
				BeforeEach(func() {
					opts = Opts{}
				})

				It("redacts every query param from endpoint for https calls", func() {
					url := "https://oauth-url?refresh_token=abc&param2=xyz"

					httpClient.PostCustomized(url, []byte("post-request"), func(r *http.Request) {}) //nolint:errcheck
					_, _, args := logger.DebugArgsForCall(0)
					Expect(args[0]).To(ContainSubstring("param2=<redacted>"))
					Expect(args[0]).To(ContainSubstring("refresh_token=<redacted>"))
					Expect(args[0]).ToNot(ContainSubstring("abc"))
					Expect(args[0]).ToNot(ContainSubstring("xyz"))
				})
			})
		})
	})

	Describe("Delete/DeleteCustomized", func() {
		It("allows to override request", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/path"),
					ghttp.VerifyHeader(http.Header{
						"X-Custom": []string{"custom"},
					}),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			url := server.URL() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
			}

			response, err := httpClient.DeleteCustomized(url, setHeaders)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Describe("httpclient opts", func() {
			BeforeEach(func() {
				opts = Opts{NoRedactUrlQuery: true}
			})

			It("does not redact every query param from endpoint for https calls", func() {
				url := "https://oauth-url?refresh_token=abc&param2=xyz"

				httpClient.Delete(url) //nolint:errcheck
				_, _, args := logger.DebugArgsForCall(0)
				Expect(args[0]).To(ContainSubstring("param2=xyz"))
				Expect(args[0]).To(ContainSubstring("refresh_token=abc"))
			})

			Context("httpclient has been configured to redact query parmas", func() {
				BeforeEach(func() {
					opts = Opts{}
				})

				It("redacts every query param from endpoint for https calls", func() {
					url := "https://oauth-url?refresh_token=abc&param2=xyz"

					httpClient.Delete(url) //nolint:errcheck
					_, _, args := logger.DebugArgsForCall(0)
					Expect(args[0]).To(ContainSubstring("param2=<redacted>"))
					Expect(args[0]).To(ContainSubstring("refresh_token=<redacted>"))
					Expect(args[0]).ToNot(ContainSubstring("abc"))
					Expect(args[0]).ToNot(ContainSubstring("xyz"))
				})
			})
		})
	})

	Describe("Put/PutCustomized", func() {
		It("makes a PUT request with given payload", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/path"),
					ghttp.VerifyBody([]byte("put-request")),
					ghttp.RespondWith(http.StatusOK, []byte("put-response")),
				),
			)

			url := server.URL() + "/path"

			response, err := httpClient.Put(url, []byte("put-request"))
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("put-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("allows to override request including payload", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/path"),
					ghttp.VerifyBody([]byte("put-request-override")),
					ghttp.VerifyHeader(http.Header{
						"X-Custom": []string{"custom"},
					}),
					ghttp.RespondWith(http.StatusOK, []byte("put-response")),
				),
			)

			url := server.URL() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
				r.Body = io.NopCloser(bytes.NewBufferString("put-request-override"))
				r.ContentLength = 20
			}

			response, err := httpClient.PutCustomized(url, []byte("put-request"), setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("put-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("redacts passwords from error message", func() {
			url := "http://foo:bar@10.10.0.0/path"

			_, err := fastErroringHttpClient.PutCustomized(url, []byte("put-request"), func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("http://foo:<redacted>@10.10.0.0/path"))
		})

		It("redacts passwords from error message for https calls", func() {
			url := "https://foo:bar@10.10.0.0/path"

			_, err := fastErroringHttpClient.PutCustomized(url, []byte("put-request"), func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("https://foo:<redacted>@10.10.0.0/path"))
		})

		Describe("httpclient opts", func() {
			BeforeEach(func() {
				opts = Opts{NoRedactUrlQuery: true}
			})

			It("does not redact every query param from endpoint for https calls", func() {
				url := "https://oauth-url?refresh_token=abc&param2=xyz"

				httpClient.PutCustomized(url, []byte("post-request"), func(r *http.Request) {}) //nolint:errcheck
				_, _, args := logger.DebugArgsForCall(0)
				Expect(args[0]).To(ContainSubstring("param2=xyz"))
				Expect(args[0]).To(ContainSubstring("refresh_token=abc"))
			})

			Context("httpclient has been configured to redact query parmas", func() {
				BeforeEach(func() {
					opts = Opts{}
				})

				It("redacts every query param from endpoint for https calls", func() {
					url := "https://oauth-url?refresh_token=abc&param2=xyz"

					httpClient.PutCustomized(url, []byte("post-request"), func(r *http.Request) {}) //nolint:errcheck
					_, _, args := logger.DebugArgsForCall(0)
					Expect(args[0]).To(ContainSubstring("param2=<redacted>"))
					Expect(args[0]).To(ContainSubstring("refresh_token=<redacted>"))
					Expect(args[0]).ToNot(ContainSubstring("abc"))
					Expect(args[0]).ToNot(ContainSubstring("xyz"))
				})
			})
		})
	})

	Describe("Get/GetCustomized", func() {
		It("makes a get request with given payload", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.RespondWith(http.StatusOK, []byte("get-response")),
				),
			)

			url := server.URL() + "/path"

			response, err := httpClient.Get(url)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("get-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("allows to override request", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/path"),
					ghttp.VerifyHeader(http.Header{
						"X-Custom": []string{"custom"},
					}),
					ghttp.RespondWith(http.StatusOK, []byte("get-response")),
				),
			)

			url := server.URL() + "/path"

			setHeaders := func(r *http.Request) {
				r.Header.Add("X-Custom", "custom")
			}

			response, err := httpClient.GetCustomized(url, setHeaders)
			Expect(err).ToNot(HaveOccurred())

			defer response.Body.Close()

			responseBody, err := io.ReadAll(response.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(responseBody).To(Equal([]byte("get-response")))
			Expect(response.StatusCode).To(Equal(200))

			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("redacts passwords from error message", func() {
			url := "http://foo:bar@10.10.0.0/path"

			_, err := fastErroringHttpClient.GetCustomized(url, func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("http://foo:<redacted>@10.10.0.0/path"))
		})

		It("redacts passwords from error message for https calls", func() {
			url := "https://foo:bar@10.10.0.0:8080/path"

			_, err := fastErroringHttpClient.GetCustomized(url, func(r *http.Request) {})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("https://foo:<redacted>@10.10.0.0:8080/path"))
		})

		Describe("httpclient opts", func() {
			BeforeEach(func() {
				opts = Opts{NoRedactUrlQuery: true}
			})

			It("does not redact every query param from endpoint for https calls", func() {
				url := "https://oauth-url?refresh_token=abc&param2=xyz"

				httpClient.GetCustomized(url, func(r *http.Request) {}) //nolint:errcheck
				_, _, args := logger.DebugArgsForCall(0)
				Expect(args[0]).To(ContainSubstring("param2=xyz"))
				Expect(args[0]).To(ContainSubstring("refresh_token=abc"))
			})

			Context("httpclient has been configured to redact query parmas", func() {
				BeforeEach(func() {
					opts = Opts{}
				})

				It("redacts every query param from endpoint for https calls", func() {
					url := "https://oauth-url?refresh_token=abc&param2=xyz"

					httpClient.GetCustomized(url, func(r *http.Request) {}) //nolint:errcheck
					_, _, args := logger.DebugArgsForCall(0)
					Expect(args[0]).To(ContainSubstring("param2=<redacted>"))
					Expect(args[0]).To(ContainSubstring("refresh_token=<redacted>"))
					Expect(args[0]).ToNot(ContainSubstring("abc"))
					Expect(args[0]).ToNot(ContainSubstring("xyz"))
				})
			})
		})
	})

})
