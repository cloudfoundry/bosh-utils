package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"code.cloudfoundry.org/tlsconfig"

	proxy "github.com/cloudfoundry/socks5-proxy"
)

var (
	DefaultClient = CreateDefaultClientInsecureSkipVerify()
	defaultDialer = SOCKS5DialFuncFromEnvironment((&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial, proxy.NewSocks5Proxy(proxy.NewHostKey(), log.New(ioutil.Discard, "", log.LstdFlags), 1*time.Minute))
)

type Client interface {
	Do(*http.Request) (*http.Response, error)
}

func CreateDefaultClient(certPool *x509.CertPool) *http.Client {
	insecureSkipVerify := false
	return factory{}.New(insecureSkipVerify, certPool)
}

func CreateDefaultClientInsecureSkipVerify() *http.Client {
	insecureSkipVerify := true
	return factory{}.New(insecureSkipVerify, nil)
}

type factory struct{}

func (f factory) New(insecureSkipVerify bool, certPool *x509.CertPool) *http.Client {
	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		WithInsecureSkipVerify(insecureSkipVerify),
	).Client(tlsconfig.WithAuthority(certPool))
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			//TODO: JM/SS: Dial is deprecated in favor of DialContext
			// We will need to update the socks5-proxy
			Dial: defaultDialer,

			TLSHandshakeTimeout: 30 * time.Second,
			DisableKeepAlives:   true,
		},
	}

	return client
}

func WithInsecureSkipVerify(insecureSkipVerify bool) tlsconfig.TLSOption {
	return func(config *tls.Config) error {
		config.InsecureSkipVerify = insecureSkipVerify
		return nil
	}
}
