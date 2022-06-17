package aerospikedbreciever // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikedbreciever"

import (
	"fmt"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v5"
	"go.opentelemetry.io/collector/config/confignet"
	"go.uber.org/zap"
)

type clientConfig struct {
	Username string            `mapstructure:username`
	Password string            `mapstructure:password`
	Host     confignet.NetAddr `mapstructure:"hosts"` // TODO does this need to be a NetAddr? could it just be a string?
}

type client struct {
	cfg        *clientConfig
	logger     *zap.Logger
	connection *aero.Client
}

// TODO should the client use a single connection or a go client?
// do we want to contact one node only, or monitro the whole cluster?
func newClient(cfg *clientConfig, logger *zap.Logger) (*client, error) {
	var res *client

	clientPolicy := aero.NewClientPolicy()
	clientPolicy.User = cfg.Username
	clientPolicy.Password = cfg.Password

	hostIPAndPort := strings.SplitN(cfg.Host.Endpoint, ":", 2)
	hostIP := hostIPAndPort[0]
	hostPort, err := strconv.Atoi(hostIPAndPort[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert host port to int: %w", err)
	}

	conn, err := aero.NewClientWithPolicy(clientPolicy, hostIP, hostPort)
	if err != nil {
		return nil, fmt.Errorf("new client with policy failed with: %w", err)
	}

	res = &client{
		logger:     logger,
		cfg:        cfg,
		connection: conn,
	}

	return res, nil
}
