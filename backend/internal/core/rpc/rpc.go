package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/icholy/digest"
)

type Client struct {
	Endpoint string
	Username string
	Password string
	client   *http.Client
	// monerod and wallet-rpc track digest nonce counts per connection, so calls
	// go one at a time over a single connection
	slot chan struct{}
}

// NewClient creates a JSON-RPC client for monerod or monero-wallet-rpc.
// Both use HTTP digest auth when started with --rpc-login.
func NewClient(endpoint, username, password string) *Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if username != "" || password != "" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxConnsPerHost = 1
		httpClient.Transport = &digest.Transport{Username: username, Password: password, Transport: transport}
	}

	return &Client{
		Endpoint: endpoint,
		Username: username,
		Password: password,
		client:   httpClient,
		slot:     make(chan struct{}, 1),
	}
}

// Error is a JSON-RPC error returned by monerod or monero-wallet-rpc.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

type rpcRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	Jsonrpc string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Call sends a JSON-RPC request and unmarshals the result into the provided result pointer.
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	reqBody, err := json.Marshal(rpcRequest{
		Jsonrpc: "2.0",
		ID:      "0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create RPC request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	select {
	case c.slot <- struct{}{}:
		defer func() { <-c.slot }()
	case <-ctx.Done():
		return fmt.Errorf("RPC %s request failed: %w", method, ctx.Err())
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("RPC %s request failed: %w", method, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	return decodeResponse(resp, result)
}

func decodeResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read RPC response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RPC HTTP %d: %s", resp.StatusCode, body)
	}
	return parseRPCBody(body, result)
}

func parseRPCBody(body []byte, result interface{}) error {
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("failed to decode RPC response: %w (body: %s)", err, body)
	}
	if rpcResp.Error != nil {
		return &Error{Code: rpcResp.Error.Code, Message: rpcResp.Error.Message}
	}
	if result != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(*rpcResp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal RPC result: %w", err)
		}
	}
	return nil
}
