// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aerospikereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikereceiver"

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/scrapererror"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikereceiver/internal/metadata"
)

// aerospikeReceiver is a metrics receiver using the Aerospike interface to collect
type aerospikeReceiver struct {
	config        *Config
	consumer      consumer.Metrics
	host          string // host/IP of configured Aerospike node
	port          int    // port of configured Aerospike node
	clientFactory clientFactoryFunc
	client        Aerospike
	mb            *metadata.MetricsBuilder
	logger        *zap.Logger
}

// clientFactoryFunc creates an Aerospike connection to the given host and port
type clientFactoryFunc func(host string, port int) (Aerospike, error)

// newAerospikeReceiver creates a new aerospikeReceiver connected to the endpoint provided in cfg
//
// If the host or port can't be parsed from endpoint, an error is returned.
func newAerospikeReceiver(params component.ReceiverCreateSettings, cfg *Config, consumer consumer.Metrics) (*aerospikeReceiver, error) {
	host, portStr, err := net.SplitHostPort(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errBadEndpoint, err)
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errBadPort, err)
	}

	var tlsCfg *tls.Config
	if cfg.TLS != nil {
		tlsCfg, err = cfg.TLS.LoadTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errFailedTLSLoad, err)
		}
	}

	return &aerospikeReceiver{
		logger:   params.Logger,
		config:   cfg,
		consumer: consumer,
		clientFactory: func(host string, port int) (Aerospike, error) {
			conf := &clientConfig{
				host:                  host,
				port:                  port,
				username:              cfg.Username,
				password:              cfg.Password,
				timeout:               cfg.Timeout,
				logger:                params.Logger,
				collectClusterMetrics: cfg.CollectClusterMetrics,
				tls:                   tlsCfg,
			}
			return newASClient(
				conf,
				nodeGetterFactory,
			)
		},
		host: host,
		port: int(port),
		mb:   metadata.NewMetricsBuilder(cfg.Metrics, params.BuildInfo),
	}, nil
}

func (r *aerospikeReceiver) start(_ context.Context, _ component.Host) error {
	r.logger.Debug("executing start")

	client, err := r.clientFactory(r.host, r.port)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	r.client = client
	return nil
}

func (r *aerospikeReceiver) shutdown(_ context.Context) error {
	r.logger.Debug("executing close")
	r.client.Close()
	r.logger.Debug("returning close")
	return nil
}

// scrape scrapes both Node and Namespace metrics from the provided Aerospike node.
// If CollectClusterMetrics is true, it then scrapes every discovered node
func (r *aerospikeReceiver) scrape(ctx context.Context) (pmetric.Metrics, error) {
	r.logger.Sugar().Debug("begining scrape")
	errs := &scrapererror.ScrapeErrors{}
	now := pcommon.NewTimestampFromTime(time.Now().UTC())
	client := r.client

	info := client.Info()
	for _, nodeInfo := range info {
		r.emitNode(nodeInfo, now, errs)
	}
	r.scrapeNamespaces(client, now, errs)

	return r.mb.Emit(), errs.Combine()
}

// emitNode records node metrics and emits the resource. If statistics are missing in INFO, nothing is recorded
func (r *aerospikeReceiver) emitNode(info map[string]string, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	r.logger.Sugar().Debugf("emitNode len(info): %v", len(info))
	for k, v := range info {
		switch k {
		case "client_connections":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionOpenDataPoint(now, v, metadata.AttributeConnectionTypeClient))
		case "fabric_connections":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionOpenDataPoint(now, v, metadata.AttributeConnectionTypeFabric))
		case "heartbeat_connections":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionOpenDataPoint(now, v, metadata.AttributeConnectionTypeHeartbeat))
		case "client_connections_closed":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeClient, metadata.AttributeConnectionOpClose))
		case "client_connections_opened":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeClient, metadata.AttributeConnectionOpOpen))
		case "fabric_connections_closed":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeFabric, metadata.AttributeConnectionOpClose))
		case "fabric_connections_opened":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeFabric, metadata.AttributeConnectionOpOpen))
		case "heartbeat_connections_closed":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeHeartbeat, metadata.AttributeConnectionOpClose))
		case "heartbeat_connections_opened":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeConnectionCountDataPoint(now, v, metadata.AttributeConnectionTypeHeartbeat, metadata.AttributeConnectionOpOpen))
		case "system_free_mem_pct":
			addPartialIfError(errs, r.mb.RecordAerospikeNodeMemoryFreeDataPoint(now, v))
		}
	}

	r.mb.EmitForResource(metadata.WithAerospikeNodeName(info["node"]))
	r.logger.Sugar().Debug("finished emitNode")
}

// scrapeNamespaces records metrics for all namespaces on a node
// The given client is used to collect namespace metrics, which is connected to a single node
func (r *aerospikeReceiver) scrapeNamespaces(client Aerospike, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	r.logger.Sugar().Debug("scraping namespaces")
	nInfo := client.NamespaceInfo()
	r.logger.Sugar().Debugf("scrapeNamespaces len(nInfo): %v", len(nInfo))
	for node, nsMap := range nInfo {
		for nsName, nsStats := range nsMap {
			nsStats["node"] = node
			nsStats["name"] = nsName
			r.emitNamespace(nsStats, now, errs)
		}
	}
}

// emitNamespace emits a namespace resource with its name as resource attribute
func (r *aerospikeReceiver) emitNamespace(info map[string]string, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	r.logger.Sugar().Debugf("emitNamespace len(info): %v", len(info))
	for k, v := range info {
		switch k {
		// Capacity
		case "device_available_pct":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceDiskAvailableDataPoint(now, v))
		case "memory_free_pct":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceMemoryFreeDataPoint(now, v))

		// Memory usage
		case "memory_used_data_bytes":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceMemoryUsageDataPoint(now, v, metadata.AttributeNamespaceComponentData))
		case "memory_used_index_bytes":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceMemoryUsageDataPoint(now, v, metadata.AttributeNamespaceComponentIndex))
		case "memory_used_sindex_bytes":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceMemoryUsageDataPoint(now, v, metadata.AttributeNamespaceComponentSecondaryIndex))
		case "memory_used_set_index_bytes":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceMemoryUsageDataPoint(now, v, metadata.AttributeNamespaceComponentSetIndex))

		// Scans
		case "scan_aggr_abort":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeAggregation, metadata.AttributeScanResultAbort))
		case "scan_aggr_complete":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeAggregation, metadata.AttributeScanResultComplete))
		case "scan_aggr_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeAggregation, metadata.AttributeScanResultError))
		case "scan_basic_abort":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeBasic, metadata.AttributeScanResultAbort))
		case "scan_basic_complete":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeBasic, metadata.AttributeScanResultComplete))
		case "scan_basic_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeBasic, metadata.AttributeScanResultError))
		case "scan_ops_bg_abort":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeOpsBackground, metadata.AttributeScanResultAbort))
		case "scan_ops_bg_complete":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeOpsBackground, metadata.AttributeScanResultComplete))
		case "scan_ops_bg_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeOpsBackground, metadata.AttributeScanResultError))
		case "scan_udf_bg_abort":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeUdfBackground, metadata.AttributeScanResultAbort))
		case "scan_udf_bg_complete":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeUdfBackground, metadata.AttributeScanResultComplete))
		case "scan_udf_bg_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceScanCountDataPoint(now, v, metadata.AttributeScanTypeUdfBackground, metadata.AttributeScanResultError))

		// 'Delete' transactions
		case "client_delete_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeDelete, metadata.AttributeTransactionResultError))
		case "client_delete_filtered_out":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeDelete, metadata.AttributeTransactionResultFilteredOut))
		case "client_delete_not_found":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeDelete, metadata.AttributeTransactionResultNotFound))
		case "client_delete_success":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeDelete, metadata.AttributeTransactionResultSuccess))
		case "client_delete_timeout":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeDelete, metadata.AttributeTransactionResultTimeout))

		// 'Read' transactions
		case "client_read_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeRead, metadata.AttributeTransactionResultError))
		case "client_read_filtered_out":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeRead, metadata.AttributeTransactionResultFilteredOut))
		case "client_read_not_found":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeRead, metadata.AttributeTransactionResultNotFound))
		case "client_read_success":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeRead, metadata.AttributeTransactionResultSuccess))
		case "client_read_timeout":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeRead, metadata.AttributeTransactionResultTimeout))

		// UDF transactions
		case "client_udf_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeUdf, metadata.AttributeTransactionResultError))
		case "client_udf_filtered_out":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeUdf, metadata.AttributeTransactionResultFilteredOut))
		case "client_udf_not_found":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeUdf, metadata.AttributeTransactionResultNotFound))
		case "client_udf_success":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeUdf, metadata.AttributeTransactionResultSuccess))
		case "client_udf_timeout":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeUdf, metadata.AttributeTransactionResultTimeout))

		// 'Write' transactions
		case "client_write_error":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeWrite, metadata.AttributeTransactionResultError))
		case "client_write_filtered_out":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeWrite, metadata.AttributeTransactionResultFilteredOut))
		case "client_write_not_found":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeWrite, metadata.AttributeTransactionResultNotFound))
		case "client_write_success":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeWrite, metadata.AttributeTransactionResultSuccess))
		case "client_write_timeout":
			addPartialIfError(errs, r.mb.RecordAerospikeNamespaceTransactionCountDataPoint(now, v, metadata.AttributeTransactionTypeWrite, metadata.AttributeTransactionResultTimeout))

		}
	}

	r.mb.EmitForResource(metadata.WithAerospikeNamespace(info["name"]), metadata.WithAerospikeNodeName(info["node"]))
	r.logger.Sugar().Debug("finished emitNamespace")
}

// addPartialIfError adds a partial error if the given error isn't nil
func addPartialIfError(errs *scrapererror.ScrapeErrors, err error) {
	if err != nil {
		errs.AddPartial(1, err)
	}
}
