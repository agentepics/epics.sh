package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/agentepics/epics.sh/internal/daemon/store"
)

type Request struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type Response struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *APIError       `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type Client struct {
	socketPath string
}

func NewClient(home string) (*Client, error) {
	cfg, err := store.Open(home).LoadConfig()
	if err != nil {
		return nil, err
	}
	return &Client{socketPath: cfg.AdminSocketPath}, nil
}

func NewDefaultClient() (*Client, error) {
	home, err := store.ResolveHome()
	if err != nil {
		return nil, err
	}
	return NewClient(home)
}

func (c *Client) Call(ctx context.Context, action string, payload any, out any) error {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	reqBody, err := json.Marshal(Request{
		Action:  action,
		Payload: rawPayload,
	})
	if err != nil {
		return err
	}

	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write(append(reqBody, '\n')); err != nil {
		return err
	}

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error != nil {
			return resp.Error
		}
		return errors.New("daemon returned an unknown error")
	}
	if out == nil || len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return fmt.Errorf("decode %s result: %w", action, err)
	}
	return nil
}
