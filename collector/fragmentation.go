// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
	"github.com/vexxhost/ceph_osd_exporter/internal/ceph"
)

type FragmentationCollector struct {
	logger log.Logger

	rating *prometheus.Desc
}

func NewFragmentationCollector(logger log.Logger) prometheus.Collector {
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
		level.Error(c.logger).Log("msg", "failed to get admin sockets", "err", err)
		return
	}

	for _, socket := range sockets {
		response, err := socket.SendCommand(ceph.AdminSocketCommand{
			Prefix: "bluestore allocator score block",
		})
		if err != nil {
			level.Error(c.logger).Log("msg", "failed to get osd fragmentation status", "err", err)
			continue
		}

		rating, ok := response["fragmentation_rating"].(float64)
		if !ok {
			level.Error(c.logger).Log("msg", "failed to parse fragmentation rating", "response", response)
			continue
		}

		ch <- prometheus.MustNewConstMetric(c.rating, prometheus.GaugeValue, rating, socket.Osd())
	}
}
