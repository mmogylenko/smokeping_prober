// Copyright 2018 Ben Kochie <superq@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/sparrc/go-ping"
	"strconv"
	"strings"
	"time"
)

const (
	namespace = "smokeping"
)

var (
	labelNames = []string{"ip", "host"}

	summary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "smokeping_response_latency_summary",
		Help:       "Summary for ping response latencies",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	},
		labelNames,
	)
	packetsTx = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "smokeping_packets_sent",
		Help: "counter of all packets being sent out",
	},
		labelNames,
	)
	packetsRx = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "smokeping_packets_received",
		Help: "counter of all responses received (ignoring dups)",
	},
		labelNames,
	)
	histo *prometheus.HistogramVec
)

func init() {
	prometheus.MustRegister(summary, packetsTx, packetsRx)
}

func newHisto(buckets string) {
	bucketstrings := strings.Split(buckets, ",")
	bucketlist := make([]float64, len(bucketstrings))
	for i := range bucketstrings {
		value, err := strconv.ParseFloat(bucketstrings[i], 64)
		if err != nil {
			log.Fatalf("invalid float in bucket: %s", err)
		}
		bucketlist[i] = value
	}

	histo = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "response_duration_seconds",
			Help:      "A histogram of latencies for ping responses.",
			Buckets:   bucketlist,
		},
		labelNames,
	)
	prometheus.MustRegister(histo)
}
func (pe *pingEntry) Run() {
	pe.pinger.Run()
}

func (pe *pingEntry) OnRecv(pkt *ping.Packet) {
	summary.WithLabelValues(pkt.IPAddr.String(), pkt.Addr).Observe(pkt.Rtt.Seconds())
	histo.WithLabelValues(pkt.IPAddr.String(), pkt.Addr).Observe(pkt.Rtt.Seconds())
	log.Debugf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
		pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
}
func (pe *pingEntry) OnFinish(stats *ping.Statistics) {
	log.Debugf("\n--- %s ping statistics ---\n", stats.Addr)
	log.Debugf("%d packets transmitted, %d packets received, %v%% packet loss\n",
		stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
	log.Debugf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
}

// ocassionally reset the counters, because otherwise, if we lost a packet we will _never_ reach 100% packet received status
func (pe *pingEntry) ResetIfDue() {
	if time.Since(pe.lastReset) > (time.Duration(5) * time.Minute) {
		pe.pinger.PacketsSent = 0
		pe.pinger.PacketsRecv = 0
		pe.lastReset = time.Now()
	}
}
