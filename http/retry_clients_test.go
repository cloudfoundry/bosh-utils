package http_test

import (
	"errors"
	"fmt"
	"net/http"

	fakehttp "github.com/cloudfoundry/bosh-utils/http/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-utils/http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryClients", func() {

	var logger boshlog.Logger

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Describe("RetryClient", func() {
		Describe("Do", func() {
			var (
				retryClient RetryClient
				maxAttempts int
				fakeClient  *fakehttp.FakeClient
			)

			BeforeEach(func() {
				fakeClient = fakehttp.NewFakeClient()
				maxAttempts = 7

				retryClient = NewRetryClient(fakeClient, uint(maxAttempts), 0, logger)
			})

			It("returns response from retryable request", func() {
				fakeClient.SetMessage("fake-response-body")
				fakeClient.StatusCode = 204

				req := &http.Request{}
				resp, err := retryClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(readString(resp.Body)).To(Equal("fake-response-body"))
				Expect(resp.StatusCode).To(Equal(204))
			})

			It("attemps once if request is successful", func() {
				fakeClient.StatusCode = 200

				req := &http.Request{}
				resp, err := retryClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				Expect(fakeClient.CallCount).To(Equal(1))
				Expect(fakeClient.Requests).To(ContainElement(req))
			})

			It("retries for maxAttempts if request is failing", func() {
				fakeClient.StatusCode = 404

				req := &http.Request{}
				resp, err := retryClient.Do(req)
				Expect(err).To(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(404))

				Expect(fakeClient.CallCount).To(Equal(maxAttempts))
				Expect(fakeClient.Requests).To(ContainElement(req))
			})
		})
	})

	Describe("NetworkSafeClient", func() {
		Describe("Do", func() {
			var (
				retryClient RetryClient
				maxAttempts int
				fakeClient  *fakehttp.FakeClient
			)

			BeforeEach(func() {
				fakeClient = fakehttp.NewFakeClient()
				maxAttempts = 7

				retryClient = NewNetworkSafeRetryClient(fakeClient, uint(maxAttempts), 0, logger)
			})

			It("returns response from retryable request", func() {
				fakeClient.SetMessage("fake-response-body")
				fakeClient.StatusCode = 204

				req := &http.Request{}
				resp, err := retryClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(readString(resp.Body)).To(Equal("fake-response-body"))
				Expect(resp.StatusCode).To(Equal(204))
			})

			directorErrorCodes := []int{400, 401, 403, 404, 500}
			for _, code := range directorErrorCodes {
				It(fmt.Sprintf("attemps once if request is %d", code), func() {
					fakeClient.StatusCode = code

					req := &http.Request{}
					resp, err := retryClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(code))

					Expect(fakeClient.CallCount).To(Equal(1))
					Expect(fakeClient.Requests).To(ContainElement(req))
				})
			}

			for code := 200; code < 400; code++ {
				successHttpCode := code
				It(fmt.Sprintf("attemps once if request is %d", code), func() {
					fakeClient.StatusCode = successHttpCode

					req := &http.Request{}
					resp, err := retryClient.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(successHttpCode))

					Expect(fakeClient.CallCount).To(Equal(1))
					Expect(fakeClient.Requests).To(ContainElement(req))
				})
			}

			timeoutCodes := []int{
				http.StatusGatewayTimeout,
				http.StatusServiceUnavailable,
			}
			for _, code := range timeoutCodes {
				code := code

				Context(fmt.Sprintf("timeout http status code '%d'", code), func() {
					It("retries for maxAttempts", func() {
						fakeClient.StatusCode = code

						req := &http.Request{}
						resp, err := retryClient.Do(req)
						Expect(err).To(HaveOccurred())

						Expect(resp.StatusCode).To(Equal(code))

						Expect(fakeClient.CallCount).To(Equal(maxAttempts))
						Expect(fakeClient.Requests).To(ContainElement(req))
					})
				})
			}

		})

	})

	Describe("Get", func() {
		var (
			retryClient RetryClient
			fakeClient  *fakehttp.FakeClient
		)

		maxAttempts := 2

		Context("when server fails more often than maxAttempts", func() {

			BeforeEach(func() {
				fakeClient = fakehttp.NewFakeClient()
				retryClient = NewRetryClient(fakeClient, uint(maxAttempts), 0, logger)
			})

			It("fails after maxAttempts", func() {
				fakeClient.StatusCode = 500
				fakeClient.Error = errors.New("Some error")

				resp, err := retryClient.Get("some url")
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(500))
				Expect(err.Error()).To(Equal("Some error"))
				Expect(fakeClient.CallCount).To(Equal(maxAttempts))
			})

		})

		Context("when server succeeds within maxAttempts", func() {

			BeforeEach(func() {
				fakeClient = fakehttp.NewFakeClient()
				fakeClient.AddDoBehavior(
					&http.Response{
						Body:       NewStringReadCloser("error"),
						StatusCode: 500,
					},
					errors.New("error"),
				)
				fakeClient.AddDoBehavior(
					&http.Response{
						Body:       NewStringReadCloser("success"),
						StatusCode: 200,
					},
					nil,
				)

				retryClient = NewRetryClient(fakeClient, uint(maxAttempts), 0, logger)
			})

			It("succeeds", func() {
				fakeClient.StatusCode = 200

				resp, err := retryClient.Get("some url")
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))
				Expect(fakeClient.CallCount).To(Equal(maxAttempts))
			})
		})
	})

	Describe("GetWithHeaders", func() {
		var (
			retryClient RetryClient
			fakeClient  *fakehttp.FakeClient
		)

		BeforeEach(func() {
			fakeClient = fakehttp.NewFakeClient()
			retryClient = NewRetryClient(fakeClient, 1, 0, logger)
		})

		It("sends headers", func() {
			headers := map[string]string{
				"foo": "bar",
			}
			retryClient.GetWithHeaders("some url", headers)
			Expect(fakeClient.Requests[0].Header).To(Equal(http.Header{
				"Foo": []string{"bar"},
			}))
		})

	})
})
