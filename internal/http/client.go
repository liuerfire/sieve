package httpx

import (
	"net/http"
	"time"
)

const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

type userAgentTransport struct {
	base http.RoundTripper
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	if cloned.Header.Get("User-Agent") == "" {
		cloned.Header.Set("User-Agent", DefaultUserAgent)
	}
	return t.base.RoundTrip(cloned)
}

func NewClient() *http.Client {
	base := http.DefaultTransport
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: userAgentTransport{
			base: base,
		},
	}
}
