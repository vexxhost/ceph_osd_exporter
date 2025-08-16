// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/vexxhost/ceph_osd_exporter/internal/ceph"
)

type FragmentationCollector struct {
	logger *slog.Logger

	rating *prometheus.Desc
}

func NewFragmentationCollector(logger *slog.Logger) prometheus.Collector {
	return &FragmentationCollector{
		logger: logger,

		rating: prometheus.NewDesc(
			prometheus.BuildFQName("ceph_osd", "fragmentation", "rating"),
			"Fragmentation rating of the OSD",
			[]string{"osd"},
			nil,
		),
	}
}

func (c *FragmentationCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.rating
}

func (c *FragmentationCollector) Collect(ch chan<- prometheus.Metric) {
	filesystem := afero.NewOsFs()

	sockets, err := ceph.GetAllAdminSockets(filesystem)
	if err != nil {
		c.logger.Error("failed to get admin sockets", "err", err)
		return
	}

	var wg sync.WaitGroup
	for _, socket := range sockets {
		wg.Go(func() {
			response, err := socket.SendCommand(ceph.AdminSocketCommand{
				Prefix: "bluestore allocator score block",
			})
			if err != nil {
				c.logger.Error("failed to get osd fragmentation status", "osd", socket.Osd(), "err", err)
				return
			}

			rating, ok := response["fragmentation_rating"].(float64)
			if !ok {
				c.logger.Error("failed to parse fragmentation rating", "osd", socket.Osd(), "response", response)
				return
			}

			ch <- prometheus.MustNewConstMetric(c.rating, prometheus.GaugeValue, rating, socket.Osd())
		})
	}

	wg.Wait()
}
