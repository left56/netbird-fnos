package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, args...).Output()
}

type Client struct {
	runner  Runner
	binary  string
	timeout time.Duration
}
type Status struct {
	State     string `json:"state"`
	Connected bool   `json:"connected"`
	Detail    string `json:"detail,omitempty"`
}

func NewClient(runner Runner, binary string, timeout time.Duration) Client {
	return Client{runner: runner, binary: binary, timeout: timeout}
}

func (c Client) Status(ctx context.Context) Status {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	output, err := c.runner.Run(ctx, c.binary, "status", "--json")
	if err != nil {
		return unavailable(err)
	}
	var response struct {
		Status    string `json:"status"`
		Connected bool   `json:"connected"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return Status{State: "unavailable", Detail: "official NetBird CLI returned an unsupported response"}
	}
	if response.Status == "" {
		response.Status = "unknown"
	}
	return Status{State: response.Status, Connected: response.Connected}
}

func unavailable(err error) Status {
	if errors.Is(err, exec.ErrNotFound) {
		return Status{State: "unavailable", Detail: "official NetBird CLI is not installed"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Status{State: "unavailable", Detail: "official NetBird CLI timed out"}
	}
	return Status{State: "unavailable", Detail: "official NetBird CLI is unavailable"}
}

func (c Client) String() string { return fmt.Sprintf("netbird client (%s)", c.binary) }
