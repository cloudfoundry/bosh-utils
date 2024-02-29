package httpclient_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"time"

	"github.com/cloudfoundry/bosh-utils/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/paraphernalia/secure/tlsconfig"
)

var _ = Describe("NewMutualTLSClient", func() {
	var (
		identity   tls.Certificate
		caCertPool *x509.CertPool
		serverName string

		client    *http.Client
		tlsConfig *tls.Config
		timeout   time.Duration
		err       error
	)

	BeforeEach(func() {
		// Load client cert
		identity, err = tls.LoadX509KeyPair("./assets/test_client.pem", "./assets/test_client.key")
		Expect(err).ToNot(HaveOccurred())
		// Load CA cert
		caCert, err := os.ReadFile("./assets/test_ca.pem")
		Expect(err).ToNot(HaveOccurred())
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		timeout = 10 * time.Second
		serverName = "server_name"
	})

	JustBeforeEach(func() {
		client = httpclient.NewMutualTLSClient(identity, caCertPool, serverName)
		tlsConfig = client.Transport.(*http.Transport).TLSClientConfig
	})

	It("configures a ca cert pool", func() {
		Expect(tlsConfig.RootCAs).To(Equal(caCertPool))
	})

	It("configures a client certificate", func() {
		Expect(tlsConfig.Certificates).To(ConsistOf(identity))
	})

	It("has secure tls defaults", func() {
		tlsConfigBefore := *tlsConfig //nolint:govet

		tlsconfig.WithInternalServiceDefaults()(tlsConfig)

		Expect(*tlsConfig).To(Equal(tlsConfigBefore)) //nolint:govet
	})

	It("sets up a timeout", func() {
		Expect(client.Timeout).To(Equal(timeout))
	})

	It("configures the tls server name", func() {
		Expect(tlsConfig.ServerName).To(Equal(serverName))
	})
})
