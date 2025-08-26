package reporter

import (
	"os"

	"github.com/solarwinds/apm-go/internal/config"
	"github.com/solarwinds/apm-go/internal/log"
)

func CreateGrpcConnection() (*grpcConnection, error) {
	// collector address override
	addr := config.GetCollector()

	var opts []GrpcConnOpt
	// certificate override
	if certPath := config.GetTrustedPath(); certPath != "" {
		var err error
		cert, err := os.ReadFile(certPath)
		if err != nil {
			log.Errorf("Error reading cert file %s: %v", certPath, err)
			return nil, err
		}
		opts = append(opts, WithCert(string(cert)))
	}

	opts = append(opts, WithMaxReqBytes(config.ReporterOpts().GetMaxReqBytes()))

	if proxy := getProxy(); proxy != "" {
		opts = append(opts, WithProxy(proxy))
		opts = append(opts, WithProxyCertPath(getProxyCertPath()))
	}

	// create connection object for events client and metrics client
	grpcConn, err1 := newGrpcConnection("SolarWinds Observability gRPC channel", addr, opts...)
	if err1 != nil {
		log.Errorf("Failed to initialize gRPC reporter %v: %v", addr, err1)
		return nil, err1
	}
	return grpcConn, nil
}
