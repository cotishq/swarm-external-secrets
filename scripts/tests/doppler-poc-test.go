package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/docker/go-plugins-helpers/secrets"

	"github.com/sugar-org/vault-swarm-plugin/providers"
)

func main() {
	token := os.Getenv("DOPPLER_TOKEN")
	project := os.Getenv("DOPPLER_PROJECT")
	config := os.Getenv("DOPPLER_CONFIG")

	if token == "" || project == "" || config == "" {
		log.Fatalf("set DOPPLER_TOKEN, DOPPLER_PROJECT, and DOPPLER_CONFIG before running")
	}

	provider := &providers.DopplerProvider{}
	if err := provider.Initialize(map[string]string{
		"DOPPLER_TOKEN":   token,
		"DOPPLER_PROJECT": project,
		"DOPPLER_CONFIG":  config,
	}); err != nil {
		log.Fatalf("initialize failed: %v", err)
	}
	defer func() {
		if err := provider.Close(); err != nil {
			log.Printf("close failed: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	secret, err := provider.GetSecret(ctx, secrets.Request{
		SecretName: "DB_PASSWORD",
		SecretLabels: map[string]string{
			"doppler_name": "DB_PASSWORD",
		},
	})
	if err != nil {
		log.Fatalf("get secret failed: %v", err)
	}

	fmt.Printf("DB_PASSWORD=%s\n", string(secret))
}
