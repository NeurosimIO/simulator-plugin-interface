// Package client provides utilities for plugin registration and communication
// with the NeuroSim simulation API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/neurosimio/simulator-plugin-interface/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DefaultAPIHost        = "localhost"
	DefaultAPIPort        = "8080"
	DefaultTimeout        = 30 * time.Second
	RegisterPath          = "/api/v1/plugins/register"
	UnregisterPathPattern = "/api/v1/plugins/%s"
	HealthPath            = "/health"
)

// RegistrationConfig contains configuration for plugin registration
type RegistrationConfig struct {
	APIHost     string        `json:"apiHost"`
	APIPort     string        `json:"apiPort"`
	Timeout     time.Duration `json:"timeout"`
	PluginID    string        `json:"pluginId"`
	Manifest    *pb.PluginManifest `json:"manifest"`
	GRPCAddress string        `json:"grpcAddress"`
}

// RegistrationClient handles plugin registration with the simulation API
type RegistrationClient struct {
	config     *RegistrationConfig
	httpClient *http.Client
	baseURL    string
}

// RegistrationRequest wraps the manifest for API registration
type RegistrationRequest struct {
	Manifest *pb.PluginManifest `json:"manifest"`
}

// RegistrationResponse represents the response from plugin registration
type RegistrationResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PluginID  string `json:"pluginId"`
	Timestamp string `json:"timestamp"`
}

// NewRegistrationClient creates a new plugin registration client
func NewRegistrationClient(config *RegistrationConfig) *RegistrationClient {
	if config.APIHost == "" {
		config.APIHost = DefaultAPIHost
	}
	if config.APIPort == "" {
		config.APIPort = DefaultAPIPort
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	client := &RegistrationClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL: fmt.Sprintf("http://%s:%s", config.APIHost, config.APIPort),
	}

	return client
}

// RegisterPlugin registers the plugin with the simulation API
func (c *RegistrationClient) RegisterPlugin() error {
	if c.config.Manifest == nil {
		return fmt.Errorf("plugin manifest is required for registration")
	}

	// Update manifest with actual gRPC endpoint
	if c.config.GRPCAddress != "" {
		c.config.Manifest.GrpcEndpoint = c.config.GRPCAddress
	}

	request := &RegistrationRequest{
		Manifest: c.config.Manifest,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	registerURL := c.baseURL + RegisterPath
	resp, err := c.httpClient.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	var regResp RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	if !regResp.Success {
		return fmt.Errorf("registration failed: %s", regResp.Message)
	}

	c.config.PluginID = regResp.PluginID
	return nil
}

// UnregisterPlugin removes the plugin from the simulation API
func (c *RegistrationClient) UnregisterPlugin() error {
	if c.config.PluginID == "" {
		return fmt.Errorf("plugin ID is required for unregistration")
	}

	unregisterURL := c.baseURL + fmt.Sprintf(UnregisterPathPattern, c.config.PluginID)
	req, err := http.NewRequest("DELETE", unregisterURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create unregister request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send unregister request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unregistration failed with status: %d", resp.StatusCode)
	}

	return nil
}

// HealthCheck performs a health check against the simulation API
func (c *RegistrationClient) HealthCheck() error {
	healthURL := c.baseURL + HealthPath
	resp, err := c.httpClient.Get(healthURL)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// RegisterWithRetries attempts to register the plugin with exponential backoff
func (c *RegistrationClient) RegisterWithRetries(maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// First check if API is healthy
		if err := c.HealthCheck(); err != nil {
			lastErr = fmt.Errorf("API health check failed: %w", err)
		} else {
			// API is healthy, try to register
			if err := c.RegisterPlugin(); err != nil {
				lastErr = fmt.Errorf("registration failed: %w", err)
			} else {
				return nil // Success
			}
		}

		// Wait before retrying (exponential backoff)
		if attempt < maxRetries {
			delay := time.Duration(attempt) * baseDelay
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all registration attempts failed, last error: %w", lastErr)
}

// PluginServiceClient creates a gRPC client for connecting to other plugins
type PluginServiceClient struct {
	conn   *grpc.ClientConn
	client pb.PluginServiceClient
}

// NewPluginServiceClient creates a new gRPC client for plugin communication
func NewPluginServiceClient(endpoint string) (*PluginServiceClient, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to plugin at %s: %w", endpoint, err)
	}

	return &PluginServiceClient{
		conn:   conn,
		client: pb.NewPluginServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *PluginServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// HealthCheck performs a health check on the connected plugin
func (c *PluginServiceClient) HealthCheck(ctx context.Context, service string) (*pb.HealthCheckResponse, error) {
	req := &pb.HealthCheckRequest{
		Service: service,
	}
	return c.client.HealthCheck(ctx, req)
}

// GetManifest retrieves the plugin's manifest
func (c *PluginServiceClient) GetManifest(ctx context.Context) (*pb.PluginManifest, error) {
	req := &pb.GetManifestRequest{}
	resp, err := c.client.GetManifest(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Manifest, nil
}

// ProcessMessage sends a message to a component plugin for processing
func (c *PluginServiceClient) ProcessMessage(ctx context.Context, instanceID string, message *pb.SimMessage) (*pb.ProcessMessageResponse, error) {
	req := &pb.ProcessMessageRequest{
		InstanceId: instanceID,
		Message:    message,
	}
	return c.client.ProcessMessage(ctx, req)
}

// EncodeMessage encodes a message using a message ICD plugin
func (c *PluginServiceClient) EncodeMessage(ctx context.Context, messageType, format string, payload map[string]interface{}) (*pb.EncodeMessageResponse, error) {
	// Convert payload to protobuf Struct
	// Note: This is a simplified implementation - you may want to use structpb.NewStruct
	req := &pb.EncodeMessageRequest{
		MessageType: messageType,
		Format:      format,
		// Payload: payloadStruct, // TODO: Convert map to structpb.Struct
	}
	return c.client.EncodeMessage(ctx, req)
}

// DecodeMessage decodes a message using a message ICD plugin
func (c *PluginServiceClient) DecodeMessage(ctx context.Context, messageType, contentType string, encodedPayload []byte) (*pb.DecodeMessageResponse, error) {
	req := &pb.DecodeMessageRequest{
		MessageType:    messageType,
		ContentType:    contentType,
		EncodedPayload: encodedPayload,
	}
	return c.client.DecodeMessage(ctx, req)
}