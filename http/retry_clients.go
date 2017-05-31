package http

import (
	"net/http"
	"time"

	"github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type retryClient struct {
	delegate              Client
	maxAttempts           uint
	retryDelay            time.Duration
	logger                boshlog.Logger
	isResponseAttemptable func(*http.Response, error) (bool, error)
}

type RetryClient interface {
	Client
	GetWithHeaders(url string, headers map[string]string) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}

func NewRetryClient(
	delegate Client,
	maxAttempts uint,
	retryDelay time.Duration,
	logger boshlog.Logger,
) RetryClient {
	return &retryClient{
		delegate:              delegate,
		maxAttempts:           maxAttempts,
		retryDelay:            retryDelay,
		logger:                logger,
		isResponseAttemptable: nil,
	}
}

func NewNetworkSafeRetryClient(
	delegate Client,
	maxAttempts uint,
	retryDelay time.Duration,
	logger boshlog.Logger,
) RetryClient {
	return &retryClient{
		delegate:    delegate,
		maxAttempts: maxAttempts,
		retryDelay:  retryDelay,
		logger:      logger,
		isResponseAttemptable: func(resp *http.Response, err error) (bool, error) {
			isSafeMethod := resp.Request.Method == "GET" || resp.Request.Method == "HEAD"

			if err != nil || (isSafeMethod && (resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusServiceUnavailable)) {
				return true, errors.WrapError(err, "Retry")
			}

			return false, nil
		},
	}
}

func NewDefaultRetryClient(
	maxAttempts uint,
	retryDelay time.Duration,
	logger boshlog.Logger,
) RetryClient {
	return NewRetryClient(&http.Client{}, maxAttempts, retryDelay, logger)
}

func (r *retryClient) Do(req *http.Request) (*http.Response, error) {
	requestRetryable := NewRequestRetryable(req, r.delegate, r.logger, r.isResponseAttemptable)
	retryStrategy := boshretry.NewAttemptRetryStrategy(int(r.maxAttempts), r.retryDelay, requestRetryable, r.logger)
	err := retryStrategy.Try()

	return requestRetryable.Response(), err
}

func (r *retryClient) GetWithHeaders(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	return r.Do(req)
}

func (r *retryClient) Get(url string) (*http.Response, error) {
	return r.GetWithHeaders(url, map[string]string{})
}
