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
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikedbreciever/internal/metadata"
)

type aerospikedbScraper struct {
	cfg            *Config
	client         *client
	mb             *metadata.MetricsBuilder
	createSettings *component.ReceiverCreateSettings
}

func newAerospikedbScraper(cfg *Config, createSettings *component.ReceiverCreateSettings) *aerospikedbScraper {
	return &aerospikedbScraper{
		cfg:            cfg,
		mb:             metadata.NewMetricsBuilder(cfg.MetricsSettings, createSettings.BuildInfo),
		createSettings: createSettings,
	}
}

func (a *aerospikedbScraper) start(_ context.Context, _ component.Host) error {
	var err error
	a.client, err = newClient(&a.cfg.clientConfig, a.createSettings.Logger)
	return err
}

func (a *aerospikedbScraper) scrape(_ context.Context) (pmetric.Metrics, error) {
	var metrics map[string]string

	metricNames := make([]string, len(metrics))

	i := 0
	for _, name := range metrics {
		metricNames[i] = name
		i++
	}

	metrics, err := a.client.requestMetricsInfo(metricNames...)
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	now := pcommon.NewTimestampFromTime(time.Now())
	a.recordMetrics(now, metrics)

	return a.mb.Emit(), nil
}
