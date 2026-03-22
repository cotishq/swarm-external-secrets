package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/go-plugins-helpers/secrets"
	log "github.com/sirupsen/logrus"
)

// DopplerProvider implements the SecretsProvider interface for Doppler
type DopplerProvider struct {
	client *http.Client
	config *DopplerConfig
}

// DopplerConfig holds configuration for Doppler
type DopplerConfig struct {
	Token   string
	Project string
	Config  string
}

// Initialize sets up the Doppler provider
func (d *DopplerProvider) Initialize(config map[string]string) error {
	d.config = &DopplerConfig{
		Token:   getConfigOrDefault(config, "DOPPLER_TOKEN", ""),
		Project: getConfigOrDefault(config, "DOPPLER_PROJECT", ""),
		Config:  getConfigOrDefault(config, "DOPPLER_CONFIG", "prd"),
	}

	if d.config.Token == "" {
		return fmt.Errorf("DOPPLER_TOKEN is required")
	}

	d.client = &http.Client{}
	log.Printf("Successfully initialized Doppler provider")
	return nil
}

// GetSecret retrieves a secret from Doppler
func (d *DopplerProvider) GetSecret(ctx context.Context, req secrets.Request) ([]byte, error) {
	secretName := req.SecretLabels["doppler_name"]
	if secretName == "" {
		secretName = req.SecretName
	}

	project := req.SecretLabels["doppler_project"]
	if project == "" {
		project = d.config.Project
	}

	cfg := req.SecretLabels["doppler_config"]
	if cfg == "" {
		cfg = d.config.Config
	}

	url := fmt.Sprintf(
		"https://api.doppler.com/v3/configs/config/secret?project=%s&config=%s&name=%s",
		project, cfg, secretName,
	)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+d.config.Token)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Doppler API: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("failed to close doppler response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doppler API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Value struct {
			Computed string `json:"computed"`
		} `json:"value"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return []byte(result.Value.Computed), nil
}

// CheckSecretChanged checks if secret has changed using hash comparison
func (d *DopplerProvider) CheckSecretChanged(ctx context.Context, secretInfo *SecretInfo) (bool, error) {
	req := secrets.Request{
		SecretName:   secretInfo.DockerSecretName,
		SecretLabels: map[string]string{"doppler_name": secretInfo.SecretField},
	}

	value, err := d.GetSecret(ctx, req)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(value)
	currentHash := hex.EncodeToString(hash[:])
	return currentHash != secretInfo.LastHash, nil
}

// SupportsRotation returns true
func (d *DopplerProvider) SupportsRotation() bool {
	return true
}

// GetProviderName returns provider name
func (d *DopplerProvider) GetProviderName() string {
	return "doppler"
}

// Close cleans up resources
func (d *DopplerProvider) Close() error {
	return nil
}
