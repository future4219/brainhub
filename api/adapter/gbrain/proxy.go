package gbrain

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"brainhub/usecase/input_port"
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
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		if len(input_port.MCPWritableSources(r.Context())) > 0 {
			r.Header.Set("Accept-Encoding", "identity")
		}
	}
	proxy.ModifyResponse = appendMCPWriteTool
	proxy.FlushInterval = -1
	return proxy, nil
}
