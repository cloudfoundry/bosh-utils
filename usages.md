  # agent


  ## agent_request.go

	httpResponse, err := r.httpClient.Post(r.endpoint, agentRequestJSON)
	if err != nil {
		return bosherr.WrapErrorf(err, "Performing request to agent")
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()


## client_factory.go

func (f *agentClientFactory) NewAgentClient(directorID, mbusURL, caCert string) (agentclient.AgentClient, error) {
	client := httpclient.DefaultClient

	if caCert != "" {
		caCertPool, err := crypto.CertPoolFromPEM([]byte(caCert))
		if err != nil {
			return nil, err
		}
		client = httpclient.CreateDefaultClient(caCertPool)
	}

	httpClient := httpclient.NewHTTPClient(client, f.logger)
	return NewAgentClient(mbusURL, directorID, f.getTaskDelay, 10, httpClient, f.logger), nil
}


## data_service.go

	url := fmt.Sprintf("%s%s", ms.metadataHost, ms.sshKeysPath)
	resp, err := ms.client.GetCustomized(url, ms.addHeaders())
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Getting open ssh key from url %s", url)
	}

func createRetryClient(delay time.Duration, logger boshlog.Logger) boshhttpclient.HTTPClient {
	return boshhttpclient.NewHTTPClient(
		boshhttp.NewRetryClient(
			boshhttpclient.CreateDefaultClient(nil), 10, delay, logger),
		logger)
}

## http_registry.go

	client := boshhttp.NewRetryClient(boshhttpclient.CreateDefaultClient(nil), 10, r.retryDelay, r.logger)
	wrapperResponse, err := boshhttpclient.NewHTTPClient(client, r.logger).Get(settingsURL)
	if err != nil {
		return settings, bosherr.WrapError(err, "Getting settings from url")
	}


-----------------------

# cli

## uaa/client_request.go

	resp, err := r.httpClient.GetCustomized(url, setHeaders)
	if err != nil {
		return bosherr.WrapErrorf(err, "Performing request GET '%s'", url)
	}

	resp, err := r.httpClient.PostCustomized(url, payload, setHeaders)
	if err != nil {
		return bosherr.WrapErrorf(err, "Performing request POST '%s'", url)
	}

## uaa/factory.go

	rawClient := boshhttpclient.CreateDefaultClient(certPool)
	retryClient := boshhttp.NewNetworkSafeRetryClient(rawClient, 5, 500*time.Millisecond, f.logger)

	httpClient := boshhttpclient.NewHTTPClient(retryClient, f.logger)

	endpoint := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port)),
		Path:   config.Path,
	}

	return NewClient(endpoint.String(), config.Client, config.ClientSecret, httpClient, f.logger), nil

## cmd/{deployment_preparer,deployment_deleter}.go

	blobstore, err := c.blobstoreFactory.Create(installationMbus, bihttpclient.CreateDefaultClientInsecureSkipVerify())
	if err != nil {
		return nil, bosherr.WrapError(err, "Creating blobstore client")
	}

## cmd/env_factory.go

		httpClient := bihttpclient.NewHTTPClient(bitarball.HTTPClient, deps.Logger)
		tarballProvider := bitarball.NewProvider(
			tarballCache, deps.FS, httpClient, 3, 500*time.Millisecond, deps.Logger)


## installation/tarball/provider.go

		response, err := p.httpClient.Get(source.GetURL())
		if err != nil {
			return true, bosherr.WrapError(err, "Unable to download")
		}

## ssh/client.go --> NOT HTTP

		dialFunc = boshhttp.SOCKS5DialFuncFromEnvironment(net.Dial)


## director/client_request.go
	wrapperFunc := r.setContextIDHeader(f)

	resp, err := r.httpClient.GetCustomized(url, wrapperFunc)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Performing request GET '%s'", url)
	}


	wrapperFunc := func(req *http.Request) {
		if f != nil {
			f(req)
		}

		isArchive := req.Header.Get("content-type") == "application/x-compressed"

		if isArchive && req.ContentLength > 0 && req.Body != nil {
			req.Body = r.fileReporter.TrackUpload(req.ContentLength, req.Body)
		}
	}

	wrapperFunc = r.setContextIDHeader(wrapperFunc)

	resp, err := r.httpClient.PostCustomized(url, payload, wrapperFunc)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Performing request POST '%s'", url)
	}

	wrapperFunc := r.setContextIDHeader(f)

	resp, err := r.httpClient.PutCustomized(url, payload, wrapperFunc)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Performing request PUT '%s'", url)
	}


	resp, err := r.httpClient.Delete(url)
	if err != nil {
		return nil, nil, bosherr.WrapErrorf(err, "Performing request DELETE '%s'", url)
	}


func (r ClientRequest) setContextIDHeader(f func(*http.Request)) func(*http.Request) {
	return func(req *http.Request) {
		if f != nil {
			f(req)
		}
		if r.contextId != "" {
			req.Header.Set("X-Bosh-Context-Id", r.contextId)
		}
	}
}


# DNS

func NewHealthClient(caCert []byte, cert tls.Certificate, logger boshlog.Logger) httpclient.HTTPClient {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := boshhttp.NewMutualTLSClient(cert, caCertPool, "health.bosh-dns")
	client.Timeout = 5 * time.Second

	if tr, ok := client.Transport.(*http.Transport); ok {
		tr.TLSClientConfig.ClientSessionCache = tls.NewLRUClientSessionCache(0)
	}
	httpClient := boshhttp.NewNetworkSafeRetryClient(client, 4, 500*time.Millisecond, logger)
	return httpclient.NewHTTPClient(httpClient, logger)
}

