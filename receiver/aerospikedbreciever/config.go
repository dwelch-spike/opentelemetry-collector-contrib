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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikedbreciever/internal/metadata"
)

const (
	typeStr config.Type = "aerospike"
)

type Config struct {
	// TODO configtls.TLSClientSetting
	clientConfig `mapstructure:",squash"`
	metadata.MetricsSettings
	scraperhelper.ScraperControllerSettings
	id config.Type
}

func (c *Config) Validate() error {
	// TODO fill this out
	return nil
}

func createDefaultConfig() config.Receiver {
	return &Config{
		clientConfig: clientConfig{
			Host: confignet.NetAddr{},
		},
		MetricsSettings:           metadata.DefaultMetricsSettings(),
		ScraperControllerSettings: scraperhelper.NewDefaultScraperControllerSettings(typeStr),
		id:                        typeStr,
	}
}
