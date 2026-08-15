package gbrain

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewProxy(baseURL string) (http.Handler, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, errors.New("GBrain base URL must use http or https")
	}
	if target.Host == "" {
		return nil, errors.New("GBrain base URL must include a host")
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = -1
	return proxy, nil
}
