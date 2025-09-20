package rabbitmq_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Patrick-Ivann/AIM-Q/internal/rabbitmq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock http.Client ---

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	resp, _ := args.Get(0).(*http.Response)
	return resp, args.Error(1)
}

// helper: create http.Response from status, body string
func httpResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		uri     string
		wantErr bool
	}{
		{"http://user:pass@localhost:15672", false},
		{"'failure://[::1", true},
		{"http://localhost:15672", false},
	}

	for _, tc := range tests {
		c, err := rabbitmq.NewClient(tc.uri, &MockHTTPClient{})
		if tc.wantErr {
			assert.Error(t, err)
			assert.Nil(t, c)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, c)
		}
	}
}

func TestClient_get_HTTPError(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	mockClient := client.Http.(*MockHTTPClient)
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("fail network"))

	var result interface{}
	err := client.Get("exchanges", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestClient_get_NonOKStatus(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	responseBody := `{"error":"not found"}`
	mockClient := client.Http.(*MockHTTPClient)
	mockClient.On("Do", mock.Anything).
		Return(httpResponse(404, responseBody), nil)

	var result interface{}
	err := client.Get("queues", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP 404")
}

func TestClient_get_JSONDecodeError(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	mockClient := client.Http.(*MockHTTPClient)
	mockClient.On("Do", mock.Anything).
		Return(httpResponse(200, "invalid-json"), nil)

	var result map[string]interface{}
	err := client.Get("bindings", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestClient_get_Success(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	mockRespBody := `[{"name":"ex1","type":"direct"}]`
	mockClient := client.Http.(*MockHTTPClient)
	mockClient.On("Do", mock.Anything).
		Return(httpResponse(200, mockRespBody), nil)

	var result []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	err := client.Get("exchanges", &result)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "ex1", result[0].Name)
	assert.Equal(t, "direct", result[0].Type)
}

func TestClient_FetchTopology_AllSuccess(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	exchangesJSON := `[{"name":"ex1","type":"direct","vhost":"/"}]`
	queuesJSON := `[{"name":"q1","vhost":"/"}]`
	bindingsJSON := `[{"source":"ex1","destination":"q1","destination_type":"queue","vhost":"/","routing_key":""}]`
	consumersJSON := `[{"queue":"q1","consumer_tag":"ctag","vhost":"/"}]`

	mockClient := client.Http.(*MockHTTPClient)
	callOrder := []string{}

	// Setup ordered mock responses for each get call
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		callOrder = append(callOrder, "exchanges")
		return true
	})).Return(httpResponse(200, exchangesJSON), nil).Once()

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		callOrder = append(callOrder, "queues")
		return true
	})).Return(httpResponse(200, queuesJSON), nil).Once()

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		callOrder = append(callOrder, "bindings")
		return true
	})).Return(httpResponse(200, bindingsJSON), nil).Once()

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		callOrder = append(callOrder, "consumers")
		return true
	})).Return(httpResponse(200, consumersJSON), nil).Once()

	topo, err := client.FetchTopology()
	assert.NoError(t, err)
	assert.Len(t, topo.Exchanges, 1)
	assert.Len(t, topo.Queues, 1)
	assert.Len(t, topo.Bindings, 1)
	assert.Len(t, topo.Consumers, 1)
}

func TestClient_FetchTopology_FailMiddle(t *testing.T) {
	client := &rabbitmq.Client{
		Http: &MockHTTPClient{},
	}

	// exchanges succeed but queues fail
	exchangesJSON := `[{"name":"ex1","type":"direct","vhost":"/"}]`
	mockClient := client.Http.(*MockHTTPClient)
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.Path, "exchanges")
	})).Return(httpResponse(200, exchangesJSON), nil).Once()

	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.Path, "queues")
	})).Return(nil, errors.New("queue fetch failure")).Once()

	// No further calls expected
	mockClient.On("Do", mock.Anything).Return(nil, errors.New("unexpected call")).Maybe()

	_, err := client.FetchTopology()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue fetch failure")
}
