package aerospikedbreciever

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type aerospikedbScraper struct {
	cfg    *Config
	client *client
}

func NewAerospikedbScraper(cfg *Config) *aerospikedbScraper {
	return &aerospikedbScraper{
		cfg: cfg,
	}
}

func (a *aerospikedbScraper) Scrape(_ context.Context) (pmetric.Metrics, error) {

}
