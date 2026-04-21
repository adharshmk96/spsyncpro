package temporalclient

import (
	"fmt"

	"go.temporal.io/sdk/client"
)

func New(hostPort, namespace string) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to temporal: %w", err)
	}

	return c, nil
}
