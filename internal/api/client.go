package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/it-objects/terra3-cli/internal/auth"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) addAuth(req *http.Request) error {
	token, err := auth.GetValidToken()
	if err != nil {
		return fmt.Errorf("authentication required: %w\nRun 'terra3 platform login' first", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

type Stage struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	ApplicationID string        `json:"applicationId"`
	EnvironmentID string        `json:"environmentId"`
	Status        string        `json:"status"`
	DbPartitions  []DbPartition `json:"dbPartitions"`
}

type DbPartition struct {
	Name         string `json:"name"`
	SchemaName   string `json:"schemaName"`
	Username     string `json:"username"`
	Status       string `json:"status"`
	DatabaseName string `json:"databaseName"`
}

type TunnelSession struct {
	SessionId         string `json:"sessionId"`
	StreamUrl         string `json:"streamUrl"`
	TokenValue        string `json:"tokenValue"`
	Region            string `json:"region"`
	LocalPort         int    `json:"localPort"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Engine            string `json:"engine"`
	Database          string `json:"database"`
	Username          string `json:"username"`
	BastionInstanceId string `json:"bastionInstanceId"`
}

func (c *Client) GetEnvironmentName(envID string) string {
	var result struct {
		Name string `json:"name"`
	}
	if err := c.get("/environments/"+envID, &result); err != nil {
		return envID
	}
	if result.Name == "" {
		return envID
	}
	return result.Name
}

func (c *Client) ListDeveloperStages() ([]Stage, error) {
	var result struct {
		Stages []Stage `json:"stages"`
	}
	if err := c.get("/developer/stages", &result); err != nil {
		return nil, err
	}
	return result.Stages, nil
}

func (c *Client) ListDbPartitions(stageID string) ([]DbPartition, error) {
	var result struct {
		DbPartitions []DbPartition `json:"dbPartitions"`
	}
	if err := c.get(fmt.Sprintf("/stages/%s/db-partitions", stageID), &result); err != nil {
		return nil, err
	}
	return result.DbPartitions, nil
}

func (c *Client) GetDbPartitionPassword(stageID, name string) (string, error) {
	var result struct {
		Password string `json:"password"`
	}
	if err := c.get(fmt.Sprintf("/stages/%s/db-partitions/%s/password", stageID, name), &result); err != nil {
		return "", err
	}
	return result.Password, nil
}

func (c *Client) StartDbTunnel(stageID, partitionName string, localPort int) (*TunnelSession, error) {
	var result TunnelSession
	if err := c.post(
		fmt.Sprintf("/stages/%s/db-partitions/%s/tunnel", stageID, partitionName),
		map[string]int{"localPort": localPort},
		&result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) EndDbTunnel(stageID, sessionID string) error {
	return c.delete(fmt.Sprintf("/stages/%s/db-tunnels/%s", stageID, sessionID), nil)
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	if err := c.addAuth(req); err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if err := c.addAuth(req); err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, result)
}

func (c *Client) delete(path string, result interface{}) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	if err := c.addAuth(req); err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeResponse(resp, result)
}

func (c *Client) decodeResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized — run 'terra3 platform login'")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("forbidden — insufficient permissions")
	}
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errBody) == nil && errBody.Error != "" {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody.Error)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, stringsTrim(string(body)))
	}

	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}

func stringsTrim(s string) string {
	if len(s) > 500 {
		return s[:500] + "…"
	}
	return s
}
