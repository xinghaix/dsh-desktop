package main

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

const hostRecoveryRpcPrefix = "DSH_HOST_RECOVERY_RPC "
const recoveryURLSentinel = "recovery://rpc"

// RecoveryRpcClient calls the Node Host loopback Recovery RPC.
type RecoveryRpcClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type recoveryHealth struct {
	OK            bool   `json:"ok"`
	Detail        string `json:"detail"`
	HasController bool   `json:"hasController"`
}

type RecoverySnapshot struct {
	ProfileName string               `json:"profileName"`
	Bundles     []RecoveryBundle     `json:"bundles"`
	Checkpoints []RecoveryCheckpoint `json:"checkpoints"`
	Controller  bool                 `json:"controller"`
}

type RecoveryBundle struct {
	BundleID    string  `json:"bundleId"`
	PackageName string  `json:"packageName"`
	Status      string  `json:"status"`
	Owner       string  `json:"owner"`
	Action      *string `json:"action"`
}

type RecoveryCheckpoint struct {
	SlotID      string  `json:"slotId"`
	Status      string  `json:"status"`
	CapturedAt  *string `json:"capturedAt,omitempty"`
	AppVersion  *string `json:"appVersion,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	FileCount   *int    `json:"fileCount,omitempty"`
	PluginCount *int    `json:"pluginCount,omitempty"`
	TotalBytes  *int64  `json:"totalBytes,omitempty"`
}

type checkpointPreview struct {
	PreviewID  string `json:"previewId"`
	SlotID     string `json:"slotId"`
	CapturedAt string `json:"capturedAt"`
	ExpiresAt  string `json:"expiresAt"`
}

type uninstallPreview struct {
	PreviewID   string `json:"previewId"`
	PackageName string `json:"packageName"`
	ExpiresAt   string `json:"expiresAt"`
}

type recoveryRPCError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseRecoveryRpcAnnounceLine extracts base URL + bearer token from Host stdout.
func ParseRecoveryRpcAnnounceLine(line string) (baseURL, token string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, hostRecoveryRpcPrefix) {
		return "", "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, hostRecoveryRpcPrefix))
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return "", "", false
	}
	u := parts[0]
	tok := ""
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "token=") {
			tok = strings.TrimPrefix(p, "token=")
		}
	}
	if u == "" || tok == "" {
		return "", "", false
	}
	return u, tok, true
}

func (c *RecoveryRpcClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *RecoveryRpcClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("recovery rpc client is not configured")
	}
	base := strings.TrimRight(c.BaseURL, "/") + "/"
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+strings.TrimLeft(path, "/"), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var rpcErr recoveryRPCError
		if json.Unmarshal(payload, &rpcErr) == nil && rpcErr.Error.Message != "" {
			return fmt.Errorf("%s: %s", rpcErr.Error.Code, rpcErr.Error.Message)
		}
		return fmt.Errorf("recovery rpc HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(payload)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(payload, out)
}

func (c *RecoveryRpcClient) Health(ctx context.Context) (*recoveryHealth, error) {
	var out recoveryHealth
	if err := c.doJSON(ctx, http.MethodGet, "v1/health", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *RecoveryRpcClient) Snapshot(ctx context.Context) (*RecoverySnapshot, error) {
	var out RecoverySnapshot
	if err := c.doJSON(ctx, http.MethodGet, "v1/snapshot", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *RecoveryRpcClient) PreviewCheckpointRestore(ctx context.Context, slotID string) (*checkpointPreview, error) {
	var out checkpointPreview
	if err := c.doJSON(ctx, http.MethodPost, "v1/checkpoint/preview", map[string]string{"slotId": slotID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *RecoveryRpcClient) ExecuteCheckpointRestore(ctx context.Context, previewID string) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "v1/checkpoint/execute", map[string]string{"previewId": previewID}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *RecoveryRpcClient) OpenCheckpoint(ctx context.Context, slotID string) error {
	return c.doJSON(ctx, http.MethodPost, "v1/checkpoint/open", map[string]string{"slotId": slotID}, &map[string]any{})
}

func (c *RecoveryRpcClient) PreviewUninstall(ctx context.Context, bundleID string) (*uninstallPreview, error) {
	var out uninstallPreview
	if err := c.doJSON(ctx, http.MethodPost, "v1/uninstall/preview", map[string]string{"bundleId": bundleID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *RecoveryRpcClient) ExecuteUninstall(ctx context.Context, previewID string) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "v1/uninstall/execute", map[string]string{"previewId": previewID}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *RecoveryRpcClient) Complete(ctx context.Context, action string) error {
	return c.doJSON(ctx, http.MethodPost, "v1/complete", map[string]string{"action": action}, &map[string]any{})
}
