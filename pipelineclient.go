package rawhttp

import (
	"io"
	"io/ioutil"
	"net/http"
	stdurl "net/url"
	"strings"

	"github.com/projectdiscovery/rawhttp/clientpipeline"
	retryablehttp "github.com/projectdiscovery/retryablehttp-go"
)

type PipelineClient struct {
	client  *clientpipeline.PipelineClient
	options PipelineOptions
}

func NewPipelineClient(options PipelineOptions) *PipelineClient {
	client := &PipelineClient{
		client: &clientpipeline.PipelineClient{
			Addr:               options.Host,
			MaxConns:           options.MaxConnections,
			MaxPendingRequests: options.MaxPendingRequests,
			ReadTimeout:        options.Timeout,
		},
		options: options,
	}
	return client
}

func (c *PipelineClient) Head(url string) (*http.Response, error) {
	return c.DoRaw("HEAD", url, "", nil, nil)
}

func (c *PipelineClient) Get(url string) (*http.Response, error) {
	return c.DoRaw("GET", url, "", nil, nil)
}

func (c *PipelineClient) Post(url string, mimetype string, body io.Reader) (*http.Response, error) {
	headers := make(map[string][]string)
	headers["Content-Type"] = []string{mimetype}
	return c.DoRaw("POST", url, "", headers, body)
}

func (c *PipelineClient) Do(req *http.Request) (*http.Response, error) {
	method := req.Method
	headers := req.Header
	url := req.URL.String()
	body := req.Body
	return c.DoRaw(method, url, "", headers, body)
}

func (c *PipelineClient) Dor(req *retryablehttp.Request) (*http.Response, error) {
	method := req.Method
	headers := req.Header
	url := req.RequestURI
	body := req.Body

	return c.do(method, url, "", headers, body)
}

func (c *PipelineClient) DoRaw(method, url, uripath string, headers map[string][]string, body io.Reader) (*http.Response, error) {
	return c.do(method, url, uripath, headers, body)
}

func (c *PipelineClient) do(method, url, uripath string, headers map[string][]string, body io.Reader) (*http.Response, error) {
	if headers == nil {
		headers = make(map[string][]string)
	}
	u, err := stdurl.ParseRequestURI(url)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	if len(headers["Host"]) > 0 {
		host = headers["Host"][0]
	}

	// standard path
	path := u.Path
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	// override if custom one is specified
	if uripath != "" {
		path = uripath
	}

	req := clientpipeline.ToRequest(method, path, nil, headers, body)
	resp := &clientpipeline.Response{}

	err = c.client.Do(req, resp)

	// response => net/http response
	r := http.Response{
		StatusCode:    resp.Status.Code,
		ContentLength: resp.ContentLength(),
		Header:        make(http.Header),
	}

	for _, header := range resp.Headers {
		r.Header.Set(header.Key, header.Value)
	}

	r.Body = ioutil.NopCloser(resp.Body)

	return &r, err
}