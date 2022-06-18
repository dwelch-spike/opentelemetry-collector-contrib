// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aerospikedbreciever // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikedbreciever"

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v5"
	"go.opentelemetry.io/collector/config/confignet"
	"go.uber.org/zap"
)

type clientConfig struct {
	Username string            `mapstructure:"username"`
	Password string            `mapstructure:"password"`
	Host     confignet.NetAddr `mapstructure:"hosts"` // TODO does this need to be a NetAddr? could it just be a string?
}

type client struct {
	cfg        *clientConfig
	logger     *zap.Logger // TODO is logger needed?
	connection *aero.Connection
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

	aeroHost := aero.NewHost(hostIP, hostPort)
	conn, err := aero.NewConnection(clientPolicy, aeroHost)
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

func (c client) requestMetricsInfo(mType ...string) (map[string]string, error) {
	var err error
	var metrics map[string]string

	if c.connection == nil {
		return metrics, errors.New("client connection is nil")
	}

	metrics, err = c.connection.RequestInfo(mType...)
	if err != nil {
		return metrics, err
	}

	return metrics, nil
}
