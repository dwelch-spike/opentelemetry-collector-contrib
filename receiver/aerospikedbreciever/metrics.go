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
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

type metricType int

const (
	client_read_success metricType = iota
)

var metricTypes = map[metricType]string{
	client_read_success: "client_read_success",
}

type metricProcessor interface {
	request() (map[string]string, error)
	process(metrics map[string]string) (map[string]string, error)
	record()
}

type metricGroup struct {
	client      *client
	metrics     map[metricType]string
	metricNames []string
	delimeter   string
}

type namespaceMetrics metricGroup

func (m namespaceMetrics) request() (map[string]string, error) {
	metrics, err := m.client.requestMetricsInfo(m.metricNames...)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (a *aerospikedbScraper) recordMetrics(now pcommon.Timestamp, metrics map[string]string) {
	fmt.Printf("%+v", metrics)
	a.mb.RecordClientReadSuccessDataPoint(now, metrics[metricTypes[client_read_success]])
}

func processMetricsMap(metrics map[string]string) {
	for k, v := range metrics {
		metrics[k] = sanitizeUTF8(v)
		parseStats(v)
	}

}
