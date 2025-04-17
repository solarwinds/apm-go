package exporter

import (
	"context"
	"fmt"
)

type bearerTokenAuthCred struct {
	token string
}

func (cred *bearerTokenAuthCred) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", cred.token),
	}, nil
}

func (cred *bearerTokenAuthCred) RequireTransportSecurity() bool {
	return true
}
