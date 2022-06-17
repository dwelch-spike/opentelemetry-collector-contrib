package aerospikedbreciever

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	typeStr config.Type = "aerospike"
)

func NewFactory() component.ReceiverFactory {
	return component.NewReceiverFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsReceiver(createMetricsReciever),
	)
}

func createDefaultConfig() config.Receiver {
	return &Config{
		scraperhelper.NewDefaultScraperControllerSettings(typeStr),
		clientConfig{
			Host: confignet.NetAddr{},
		},
	}
}

// type CreateMetricsReceiverFunc func(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Metrics) (MetricsReceiver, error)

func createMetricsReciever(
	_ context.Context,
	creatorConf component.ReceiverCreateSettings,
	recieverConf config.Receiver,
	metricsConsumer consumer.Metrics,
) (component.MetricsReceiver, error) {
	cfg := recieverConf.(*Config)
	as := NewAerospikedbScraper(cfg)
	scraper, err := scraperhelper.NewScraper(
		string(typeStr),
		as.scrape,
		scraperhelper.WithStart(as.start),
	)

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ScraperControllerSettings,
		creatorConf,
		metricsConsumer,
		scraperhelper.AddScraper(as),
	)
}
