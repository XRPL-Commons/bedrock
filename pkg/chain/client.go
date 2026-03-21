package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a Go-native XRPL JSON-RPC client
type Client struct {
	httpURL    string
	httpClient *http.Client
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient creates a new XRPL RPC client from a WebSocket or HTTP URL.
// An optional rpcURL can be provided to override the derived HTTP URL.
func NewClient(url string, rpcURL ...string) *Client {
	httpURL := toHTTPURL(url)
	if len(rpcURL) > 0 && rpcURL[0] != "" {
		httpURL = rpcURL[0]
	}
	return &Client{
		httpURL: httpURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Call sends a JSON-RPC request and returns the raw result
func (c *Client) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	req := RPCRequest{
		Method: method,
	}
	if len(params) > 0 {
		req.Params = params
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.httpURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Check for XRPL-style errors inside the result object
	var resultStatus struct {
		Status       string `json:"status"`
		Error        string `json:"error"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(rpcResp.Result, &resultStatus); err == nil {
		if resultStatus.Status == "error" {
			return nil, fmt.Errorf("RPC error: %s: %s", resultStatus.Error, resultStatus.ErrorMessage)
		}
	}

	return rpcResp.Result, nil
}

// CallTyped sends an RPC request and unmarshals the result into the target
func (c *Client) CallTyped(ctx context.Context, target interface{}, method string, params ...interface{}) error {
	raw, err := c.Call(ctx, method, params...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return nil
}

// toHTTPURL converts a WebSocket URL to an HTTP URL for JSON-RPC.
// This is a fallback conversion (ws:// -> http://, wss:// -> https://).
// For local development where ports differ, use NetworkConfig.GetRPCURL() or pass rpcURL to NewClient.
func toHTTPURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "ws://") {
		rawURL = strings.Replace(rawURL, "ws://", "http://", 1)
	} else if strings.HasPrefix(rawURL, "wss://") {
		rawURL = strings.Replace(rawURL, "wss://", "https://", 1)
	}
	return rawURL
}
