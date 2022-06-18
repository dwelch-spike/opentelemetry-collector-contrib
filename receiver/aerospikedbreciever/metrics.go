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

// type metricType int

// const (
// 	client_read_success metricType = iota
// )

// var metricTypes = map[metricType]string{
// 	client_read_success: "client_read_success",
// }

func (a *aerospikedbScraper) recordMetrics(now pcommon.Timestamp, metrics map[string]string) {
	fmt.Printf("%+v", metrics)
	// a.mb.RecordClientReadSuccessDataPoint(now, metrics[metrics[client_read_success]])
}

// func processMetricsMap(metrics map[string]string) {
// 	for k, v := range metrics {
// 		metrics[k] = sanitizeUTF8(v)
// 	}
// }

// func sanitizeUTF8(lv string) string {
// 	if utf8.ValidString(lv) {
// 		return lv
// 	}
// 	fixUtf := func(r rune) rune {
// 		if r == utf8.RuneError {
// 			return 65533
// 		}
// 		return r
// 	}

// 	return strings.Map(fixUtf, lv)
// }
