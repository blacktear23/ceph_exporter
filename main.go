package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const VERSION = "1.0.0"

func main() {
	var (
		listenAddr  string
		metricsPath string
		cephPath    string
	)

	flag.StringVar(&listenAddr, "web.listen-address", ":9128", "An address to listen for web interface and telemetry.")
	flag.StringVar(&metricsPath, "web.telemetry-path", "/metrics", "A path under which to expose metrics.")
	flag.StringVar(&cephPath, "ceph.bin", "/usr/bin/ceph", "ceph command path")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.Printf("Start pppd_exporter Prometheus Exporter Version=%v", VERSION)
	log.Printf("Listen Web server at %v", listenAddr)

	registry := prometheus.NewRegistry()
	collector := NewCEPHCollector(cephPath)
	registry.MustRegister(collector)
	http.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, `<!DOCTYPE html>
			<title>muss-tcpopt Exporter</title>
			<h1>muss-tcpopt Exporter</h1>
			<p><a href=%q>Metrics</a></p>`,
			metricsPath)
		if err != nil {
			log.Printf("Error while sending a response for '/' path: %v", err)
		}
	})
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
