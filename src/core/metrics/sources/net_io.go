/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/shirou/gopsutil/v3/net"
)

const NETWORK_INTERFACE = "network_interface"

type NetIO struct {
	*namedMetric
	// This is for keeping the previous net io stats.  Need to report the delta.
	// The first level key is the network interface name, and the inside map is the net
	// io stats for that particular network interface.
	netIOStats   map[string]map[string]float64
	netOverflows float64
	init         sync.Once
	env          core.Environment
	logger       *MetricSourceLogger
	// Needed for unit tests
	netIOInterfacesFunc func(ctx context.Context) (net.InterfaceStatList, error)
	netIOCountersFunc   func(ctx context.Context, pernic bool) ([]net.IOCountersStat, error)
}

func NewNetIOSource(namespace string, env core.Environment) *NetIO {
	return &NetIO{
		namedMetric:         &namedMetric{namespace, "net"},
		env:                 env,
		logger:              NewMetricSourceLogger(),
		netIOInterfacesFunc: net.InterfacesWithContext,
		netIOCountersFunc:   net.IOCountersWithContext,
	}
}

func (nio *NetIO) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	nio.init.Do(func() {
		ifs, err := nio.newNetInterfaces(ctx)
		if err != nil || ifs == nil {
			nio.logger.Log("Cannot initialize network interfaces")
			ifs = make(map[string]map[string]float64)
		}
		nio.netIOStats = ifs
		nio.netOverflows = -1
	})

	// retrieve the current net IO stats
	currentNetIOStats, err := nio.newNetInterfaces(ctx)
	if err != nil || currentNetIOStats == nil {
		nio.logger.Log("Cannot get new network interface statistics")
		return
	}

	// calculate the delta between current and previous Net IO stats
	diffNetIOStats := Delta(currentNetIOStats, nio.netIOStats)

	// to keep the Net IO stats total over all interfaces
	totalStats := make(map[string]float64)

	// Net IO stats for each interface
	for k, v := range diffNetIOStats {

		for kk, vv := range v {
			if savedV, ok := totalStats[kk]; ok {
				totalStats[kk] = savedV + vv
			} else {
				totalStats[kk] = vv
			}
		}

		simpleMetrics := nio.convertSamplesToSimpleMetrics(v)
		select {
		case <-ctx.Done():
			return
		// network_interface is not a common dim
		case m <- metrics.NewStatsEntity([]*proto.Dimension{{Name: NETWORK_INTERFACE, Value: k}}, simpleMetrics):
		}
	}

	// collect net overflow. This is not easily obtained by gopsutil, so we exec netstat to get these values
	overflows, err := nio.env.GetNetOverflow()
	if err != nil {
		nio.logger.Log(fmt.Sprintf("Error occurred getting network overflow metrics, %v", err))
	}

	if nio.netOverflows < 0 {
		nio.netOverflows = overflows
	}
	// add listen_overflow to our total stats over all interfaces
	totalStats["listen_overflows"] = float64(overflows - nio.netOverflows)

	nio.netOverflows = overflows

	simpleMetrics := nio.convertSamplesToSimpleMetrics(totalStats)
	m <- metrics.NewStatsEntity([]*proto.Dimension{}, simpleMetrics)

	nio.netIOStats = currentNetIOStats
}

func (nio *NetIO) newNetInterfaces(ctx context.Context) (map[string]map[string]float64, error) {
	upInterfaces := make(map[string]map[string]float64)
	interfaces, err := nio.netIOInterfacesFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not initialise NetIO Interfaces: %v", err)
	}
	counters, err := nio.netIOCountersFunc(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("could not collect NetIO Counters: %v", err)
	}

	for _, interf := range interfaces {
		if isUp(interf.Flags) && !strings.HasPrefix(interf.Name, "lo") {
			for _, netio := range counters {
				if netio.Name == interf.Name {
					vvs := map[string]float64{
						"packets_out.count": float64(netio.PacketsSent),
						"packets_in.count":  float64(netio.PacketsRecv),
						"bytes_sent":        float64(netio.BytesSent),
						"bytes_rcvd":        float64(netio.BytesRecv),
						"packets_in.error":  float64(netio.Errin),
						"packets_out.error": float64(netio.Errout),
						"drops_in.count":    float64(netio.Dropin),
						"drops_out.count":   float64(netio.Dropout),
					}
					upInterfaces[interf.Name] = vvs
					break
				}
			}
		}
	}

	return upInterfaces, nil
}

func isUp(flags []string) bool {
	for _, f := range flags {
		if f == "up" {
			return true
		}
	}
	return false
}
