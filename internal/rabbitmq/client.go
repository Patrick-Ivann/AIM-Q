package rabbitmq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client provides access to RabbitMQ's management HTTP API.
type Client struct {
	baseURL string
	auth    *url.Userinfo
	http    *http.Client
}

// NewClient constructs a new RabbitMQ management API client.
func NewClient(uri string) (*Client, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}
	return &Client{
		baseURL: fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host),
		auth:    parsed.User,
		http:    http.DefaultClient,
	}, nil
}

// get fetches and decodes JSON from the specified RabbitMQ API path.
func (c *Client) get(path string, out interface{}) error {
	reqURL := fmt.Sprintf("%s/api/%s", c.baseURL, path)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	if c.auth != nil {
		pass, _ := c.auth.Password()
		req.SetBasicAuth(c.auth.Username(), pass)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// FetchTopology retrieves exchanges, queues, bindings, and consumers from RabbitMQ.
func (c *Client) FetchTopology() (*Topology, error) {
	var (
		exchanges []Exchange
		queues    []Queue
		bindings  []Binding
		consumers []Consumer
	)

	if err := c.get("exchanges", &exchanges); err != nil {
		return nil, err
	}
	if err := c.get("queues", &queues); err != nil {
		return nil, err
	}
	if err := c.get("bindings", &bindings); err != nil {
		return nil, err
	}
	if err := c.get("consumers", &consumers); err != nil {
		return nil, err
	}

	return &Topology{
		Exchanges: exchanges,
		Queues:    queues,
		Bindings:  bindings,
		Consumers: consumers,
	}, nil
}
