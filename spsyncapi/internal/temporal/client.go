package temporal

import (
	"fmt"

	"spsyncapi/internal/config"

	"go.temporal.io/sdk/client"
)

// NewClient dials the Temporal cluster from application config.
func NewClient(cfg config.TemporalConfig) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("temporal: dial: %w", err)
	}
	return c, nil
}
