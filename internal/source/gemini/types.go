package gemini

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
)

type ProxyRoundTripper struct {
	// APIKey is the API Key to set on requests.
	APIKey string

	// Transport is the underlying HTTP transport.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	ProxyURL string
}

func (t *ProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}

	if t.ProxyURL != "" {
		proxyURL, err := url.Parse(t.ProxyURL)

		if err != nil {
			return nil, err
		}
		if transport, ok := rt.(*http.Transport); ok {
			transport.Proxy = http.ProxyURL(proxyURL)
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			rt = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}
	}

	newReq := *req
	args := newReq.URL.Query()
	args.Set("key", t.APIKey)
	newReq.URL.RawQuery = args.Encode()

	resp, err := rt.RoundTrip(&newReq)
	if err != nil {
		return nil, fmt.Errorf("error during round trip: %v", err)
	}

	return resp, nil
}
