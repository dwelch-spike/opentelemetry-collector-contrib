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

package nsxtreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nsxtreceiver"

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/scrapertest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/scrapertest/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nsxtreceiver/internal/metadata"
	dm "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nsxtreceiver/internal/model"
)

func TestScrape(t *testing.T) {
	mockClient := NewMockClient(t)

	mockClient.On("ClusterNodes", mock.Anything).Return(loadTestClusterNodes())
	mockClient.On("TransportNodes", mock.Anything).Return(loadTestTransportNodes())

	mockClient.On("NodeStatus", mock.Anything, transportNode1, transportClass).Return(loadTestNodeStatus(t, transportNode1, transportClass))
	mockClient.On("NodeStatus", mock.Anything, transportNode2, transportClass).Return(loadTestNodeStatus(t, transportNode2, transportClass))
	mockClient.On("NodeStatus", mock.Anything, transportNode2, transportClass).Return(loadTestNodeStatus(t, transportNode2, transportClass))
	mockClient.On("NodeStatus", mock.Anything, managerNode1, managerClass).Return(loadTestNodeStatus(t, managerNode1, managerClass))

	mockClient.On("Interfaces", mock.Anything, managerNode1, managerClass).Return(loadTestNodeInterfaces(t, managerNode1, managerClass))
	mockClient.On("Interfaces", mock.Anything, transportNode1, transportClass).Return(loadTestNodeInterfaces(t, transportNode1, transportClass))
	mockClient.On("Interfaces", mock.Anything, transportNode2, transportClass).Return(loadTestNodeInterfaces(t, transportNode2, transportClass))

	mockClient.On("InterfaceStatus", mock.Anything, transportNode1, transportNodeNic1, transportClass).Return(loadInterfaceStats(t, transportNode1, transportNodeNic1, transportClass))
	mockClient.On("InterfaceStatus", mock.Anything, transportNode1, transportNodeNic2, transportClass).Return(loadInterfaceStats(t, transportNode1, transportNodeNic2, transportClass))
	mockClient.On("InterfaceStatus", mock.Anything, transportNode2, transportNodeNic1, transportClass).Return(loadInterfaceStats(t, transportNode2, transportNodeNic1, transportClass))
	mockClient.On("InterfaceStatus", mock.Anything, transportNode2, transportNodeNic2, transportClass).Return(loadInterfaceStats(t, transportNode2, transportNodeNic2, transportClass))
	mockClient.On("InterfaceStatus", mock.Anything, managerNode1, managerNodeNic1, managerClass).Return(loadInterfaceStats(t, managerNode1, managerNodeNic1, managerClass))
	mockClient.On("InterfaceStatus", mock.Anything, managerNode1, managerNodeNic2, managerClass).Return(loadInterfaceStats(t, managerNode1, managerNodeNic2, managerClass))

	scraper := newScraper(
		&Config{
			Metrics: metadata.DefaultMetricsSettings(),
		},
		componenttest.NewNopReceiverCreateSettings(),
	)
	scraper.client = mockClient

	metrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)

	expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "expected_metrics.json"))
	require.NoError(t, err)

	err = scrapertest.CompareMetrics(metrics, expectedMetrics)
	require.NoError(t, err)
}

func TestScrapeTransportNodeErrors(t *testing.T) {
	mockClient := NewMockClient(t)
	mockClient.On("TransportNodes", mock.Anything).Return(nil, errUnauthorized)
	scraper := newScraper(
		&Config{
			Metrics: metadata.DefaultMetricsSettings(),
		},
		componenttest.NewNopReceiverCreateSettings(),
	)
	scraper.client = mockClient

	_, err := scraper.scrape(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, errUnauthorized.Error())
}

func TestScrapeClusterNodeErrors(t *testing.T) {
	mockClient := NewMockClient(t)

	mockClient.On("ClusterNodes", mock.Anything).Return(nil, errUnauthorized)
	mockClient.On("TransportNodes", mock.Anything).Return(loadTestTransportNodes())
	scraper := newScraper(
		&Config{
			Metrics: metadata.DefaultMetricsSettings(),
		},
		componenttest.NewNopReceiverCreateSettings(),
	)
	scraper.client = mockClient

	_, err := scraper.scrape(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, errUnauthorized.Error())
}

func TestStartClientAlreadySet(t *testing.T) {
	mockClient := mockServer(t)
	scraper := newScraper(
		&Config{
			Metrics: metadata.DefaultMetricsSettings(),
			HTTPClientSettings: confighttp.HTTPClientSettings{
				Endpoint: mockClient.URL,
			},
		},
		componenttest.NewNopReceiverCreateSettings(),
	)
	_ = scraper.start(context.Background(), componenttest.NewNopHost())
	require.NotNil(t, scraper.client)
}

func TestStartBadUrl(t *testing.T) {
	scraper := newScraper(
		&Config{
			Metrics: metadata.DefaultMetricsSettings(),
			HTTPClientSettings: confighttp.HTTPClientSettings{
				Endpoint: "\x00",
			},
		},
		componenttest.NewNopReceiverCreateSettings(),
	)

	_ = scraper.start(context.Background(), componenttest.NewNopHost())
	require.Nil(t, scraper.client)
}

func TestScraperRecordNoStat(t *testing.T) {
	scraper := newScraper(
		&Config{
			HTTPClientSettings: confighttp.HTTPClientSettings{
				Endpoint: "http://localhost",
			},
			Metrics: metadata.DefaultMetricsSettings(),
		},
		componenttest.NewNopReceiverCreateSettings(),
	)
	scraper.host = componenttest.NewNopHost()
	scraper.recordNode(pcommon.NewTimestampFromTime(time.Now()), &nodeInfo{stats: nil})
}

func loadTestNodeStatus(t *testing.T, nodeID string, class nodeClass) (*dm.NodeStatus, error) {
	var classType string
	switch class {
	case transportClass:
		classType = "transport"
	default:
		classType = "cluster"
	}
	testFile, err := os.ReadFile(filepath.Join("testdata", "metrics", "nodes", classType, nodeID, "status.json"))
	require.NoError(t, err)
	switch class {
	case transportClass:
		var stats dm.TransportNodeStatus
		err = json.Unmarshal(testFile, &stats)
		require.NoError(t, err)
		return &stats.NodeStatus, err
	default:
		var stats dm.NodeStatus
		err = json.Unmarshal(testFile, &stats)
		require.NoError(t, err)
		return &stats, err
	}
}

func loadTestNodeInterfaces(t *testing.T, nodeID string, class nodeClass) ([]dm.NetworkInterface, error) {
	var classType string
	switch class {
	case transportClass:
		classType = "transport"
	default:
		classType = "cluster"
	}
	testFile, err := os.ReadFile(filepath.Join("testdata", "metrics", "nodes", classType, nodeID, "interfaces", "index.json"))
	require.NoError(t, err)
	var interfaces dm.NodeNetworkInterfacePropertiesListResult
	err = json.Unmarshal(testFile, &interfaces)
	require.NoError(t, err)
	return interfaces.Results, err
}

func loadInterfaceStats(t *testing.T, nodeID, interfaceID string, class nodeClass) (*dm.NetworkInterfaceStats, error) {
	var classType string
	switch class {
	case transportClass:
		classType = "transport"
	default:
		classType = "cluster"
	}
	testFile, err := os.ReadFile(filepath.Join("testdata", "metrics", "nodes", classType, nodeID, "interfaces", interfaceID, "stats.json"))
	require.NoError(t, err)
	var stats dm.NetworkInterfaceStats
	err = json.Unmarshal(testFile, &stats)
	require.NoError(t, err)
	return &stats, err
}

func loadTestClusterNodes() ([]dm.ClusterNode, error) {
	testFile, err := os.ReadFile(filepath.Join("testdata", "metrics", "cluster_nodes.json"))
	if err != nil {
		return nil, err
	}
	var nodes dm.ClusterNodeList
	err = json.Unmarshal(testFile, &nodes)
	return nodes.Results, err
}

func loadTestTransportNodes() ([]dm.TransportNode, error) {
	testFile, err := os.ReadFile(filepath.Join("testdata", "metrics", "transport_nodes.json"))
	if err != nil {
		return nil, err
	}
	var nodes dm.TransportNodeList
	err = json.Unmarshal(testFile, &nodes)
	return nodes.Results, err
}
