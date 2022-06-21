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
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikedbreciever/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type metricName = string

type metricFuncMap map[metricName]interface{}

type metricProcessor interface {
	request() (map[string]string, error)
	// process(metrics map[string]string) (map[string]string, error)
	record(now pcommon.Timestamp, metrics map[string]string) error
}

type metricGroup struct {
	client          *client
	metricFunctions metricFuncMap
	metricNames     []metricName
	delimeter       string
}

// TODO devise a way to gather namespace names, then create namespace queries from them
// will prbably need a namespace label

func (m metricGroup) request() (map[string]string, error) {
	queries := []string{
		"namespace/test",
	}

	metrics, err := m.client.requestMetricsInfo(queries...)
	if err != nil {
		return nil, err
	}

	m.client.logger.Sugar().Infof("metricGroup.request got: %+v", metrics)

	return metrics, nil
}

// func (m metricGroup) process(metrics map[string]string) (map[string]string, error) {
// 	for _, v := range metrics {
// 		v := sanitizeUTF8(v)
// 		tmp := parseStats(v, m.delimeter)
// 	}

// }

type namespaceMetrics struct {
	metricGroup
}

func newNamespaceMetrics(m *metadata.MetricsBuilder, c *client) *namespaceMetrics {
	metricFunctions := metricFuncMap{
		"client_read_success":  m.RecordClientReadSuccessDataPoint,
		"client_write_success": m.RecordClientWriteSuccessDataPoint,
	}

	metricNames := make([]metricName, len(metricFunctions))
	// NOTE: this means the order of metric names will be random each time
	i := 0
	for name := range metricFunctions {
		metricNames[i] = name
		i++
	}

	return &namespaceMetrics{
		metricGroup{
			client:          c,
			metricFunctions: metricFunctions,
			metricNames:     metricNames,
			delimeter:       ";",
		},
	}
}

func (m namespaceMetrics) record(now pcommon.Timestamp, metrics map[string]string) error {
	m.client.logger.Sugar().Infof("record called with metrics: %+v", metrics)

	for _, stats := range metrics {
		statsMap := parseStats(stats, m.delimeter)

		for name, recorder := range m.metricFunctions {
			switch record := recorder.(type) {
			case func(pcommon.Timestamp, int64):
				mString := statsMap[name]

				statsMap[name] = sanitizeUTF8(mString)

				val, err := tryConvert(mString)
				if err != nil {
					return err
				}

				record(now, int64(val))
			default:
				panic("unkown recorder type, shouldn't be here")
			}
		}
	}

	return nil
}
