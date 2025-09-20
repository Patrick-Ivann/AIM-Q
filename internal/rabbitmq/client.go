// Package rabbitmq defines types representing RabbitMQ objects retrieved
//
// via the management API, as well as helper methods for topology filtering.
package rabbitmq

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client provides access to RabbitMQ's management HTTP API.
//
// A Client is safe for use by a single goroutine at a time.
// Create separate instances per connection context if you need concurrency.
type Client struct {
	baseURL string        // Base URL of the RabbitMQ management API (e.g., "http://localhost:15672")
	auth    *url.Userinfo // Authentication info for HTTP basic auth
	http    *http.Client  // HTTP client used to make API requests
}

// NewClient constructs a new RabbitMQ management API client from a full URI string.
//
// The uri argument should include protocol, host, port, and authentication (e.g., "http://user:password@rabbitmqhost:15672").
// Returns an error for invalid URIs.
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

// get fetches and decodes a JSON response from the given API path into out.
//
// The out argument must be a pointer to a Go type into which the JSON will be unmarshaled.
// Returns an error if the HTTP request fails, returns a non-200 code, or if JSON decoding fails.
func (c *Client) get(path string, out interface{}) error {
	reqURL := fmt.Sprintf("%s/api/%s", c.baseURL, path)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	// Set authentication using basic auth, if available.
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

	// Read and return an error if the response is not HTTP 200.
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// FetchTopology retrieves and returns the full topology of the RabbitMQ server.
//
// This includes exchanges, queues, bindings, and consumers, all of which are
// fetched by separate GET requests to the management API.
// Returns a filled Topology struct or an error on first failure.
func (c *Client) FetchTopology() (*Topology, error) {
	var (
		exchanges []Exchange
		queues    []Queue
		bindings  []Binding
		consumers []Consumer
	)

	// Order is important, as exchanges/queues may be referenced by bindings/consumers.
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
