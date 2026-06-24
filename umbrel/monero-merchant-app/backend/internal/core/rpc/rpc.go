package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	Endpoint string
	Username string
	Password string
	client   *http.Client
	// If true, use curl --digest for this endpoint (handles monerod digest auth correctly)
	UseCurlDigest bool
}

func NewClient(endpoint, username, password string) *Client {
	return &Client{
		Endpoint:      endpoint,
		Username:        username,
		Password:        password,
		client:          &http.Client{Timeout: 30 * time.Second},
		UseCurlDigest:   false,
	}
}

// NewDaemonClient creates a client for monerod (uses curl --digest for digest auth)
func NewDaemonClient(endpoint, username, password string) *Client {
	c := NewClient(endpoint, username, password)
	c.UseCurlDigest = true
	return c
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

// Call sends a JSON-RPC request. For daemon endpoints with digest auth,
// shells out to curl --digest (handles connection reuse correctly).
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	reqBody, err := json.Marshal(rpcRequest{
		Jsonrpc: "2.0",
		ID:      "0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// For daemon RPC with digest auth, use curl --digest (handles keep-alive correctly)
	if c.UseCurlDigest && c.Username != "" {
		return c.callWithCurlDigest(ctx, reqBody, result)
	}

	// Standard HTTP client for wallet-rpc (basic auth or no auth)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Username != "" || c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[RPC] HTTP request error: %v", err)
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	return c.decodeResponse(resp, result)
}

func (c *Client) callWithCurlDigest(ctx context.Context, reqBody []byte, result interface{}) error {
	// Build curl command: curl -s --digest -u user:pass -X POST url -d body -H "Content-Type: application/json"
	// Escape password for shell safety
	escapedPass := strings.ReplaceAll(c.Password, "'", "'\"'\"'")
	cmd := exec.CommandContext(ctx, "curl", "-s", "--digest",
		"-u", c.Username+":"+escapedPass,
		"-X", "POST", c.Endpoint,
		"-d", string(reqBody),
		"-H", "Content-Type: application/json",
		"--max-time", "10",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// curl returns exit code 22 for HTTP >= 400
		log.Printf("[RPC] curl --digest error: %v, output: %s", err, string(out))
		return fmt.Errorf("curl digest request failed: %w (output: %s)", err, string(out))
	}

	return parseRPCBody(out, result)
}

func (c *Client) decodeResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return parseRPCBody(body, result)
}

func parseRPCBody(body []byte, result interface{}) error {
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("failed to decode RPC response: %w (body: %s)", err, string(body))
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if result != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(*rpcResp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal RPC result: %w", err)
		}
	}
	return nil
}
