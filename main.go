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
	"flag"
	"fmt"
	//	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sparrc/go-ping"
	"net/http"
	"time"
)

var (
	debug         = flag.Bool("debug", false, "debug mode")
	listenAddress = flag.String("web.listen-address", ":9374", "Address on which to expose metrics and web interface.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")

	buckets    = flag.String("buckets", defaultBuckets, "A comma delimited list of buckets to use")
	privileged = flag.Bool("privileged", false, "Run in privileged ICMP mode")

	// Generated with: prometheus.ExponentialBuckets(0.00005, 2, 20)
	defaultBuckets = "5e-05,0.0001,0.0002,0.0004,0.0008,0.0016,0.0032,0.0064,0.0128,0.0256,0.0512,0.1024,0.2048,0.4096,0.8192,1.6384,3.2768,6.5536,13.1072,26.2144"
	interval       = flag.Int("ping.interval", 1, "Ping interval seconds")
	timeout        = flag.Int("ping.timeout", 1, "ping timeout")
)

type pingEntry struct {
	received  bool
	hostname  string
	lastReset time.Time
	pinger    *ping.Pinger
}

func main() {
	flag.Parse()
	newHisto(*buckets)
	hosts := flag.Args()
	for _, h := range hosts {
		go pingThread(h)
	}

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Smokeping Exporter</title></head>
			<body>
			<h1>Smokeping Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	fmt.Printf("Listening on %s\n", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		fmt.Printf("Failed to start listener on %s: %s\n", *listenAddress, err)
	}
}
func pingThread(host string) {
	for {
		if *debug {
			fmt.Printf("Starting pinger for %s...\n", host)
		}
		pinger, err := ping.NewPinger(host)
		if err != nil {
			fmt.Printf("failed to create pinger: %s\n", err.Error())
			return
		}
		pe := &pingEntry{pinger: pinger, hostname: host}
		//		pinger.Interval = *interval
		pinger.Count = 1
		pinger.Timeout = time.Duration(*timeout) * time.Second
		pinger.Interval = time.Duration(*interval) * time.Second
		pinger.SetPrivileged(*privileged)
		pinger.OnRecv = pe.OnRecv
		pinger.OnFinish = pe.OnFinish
		pe.Ping()
		// technically we sleep for "interval+ping_duration", but that's ok. either it's a few ms or timeout
		// so this is seems about right
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func pingerThread() {
	/*
		for {
			for _, p := range pingers {
				fmt.Printf("Host: %s, IP=%s, sent=%d, received: %d\n", p.hostname, p.pinger.IPAddr(), p.pinger.PacketsSent, p.pinger.PacketsRecv)
				packetsTx.WithLabelValues(p.pinger.IPAddr().String(), p.hostname).Set(float64(p.pinger.PacketsSent))
				packetsRx.WithLabelValues(p.pinger.IPAddr().String(), p.hostname).Set(float64(p.pinger.PacketsRecv))
				p.ResetIfDue()
			}
			time.Sleep(*interval)
		}
	*/
}
