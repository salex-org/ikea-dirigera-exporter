package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	powerUsage := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ikea_outlet_power_watts",
		Help: "Leistung die durch eine Steckdose im IKEA Smart-Home-System geht.",
	})

	// 2. Metrik registrieren
	prometheus.MustRegister(powerUsage)

	// 3. Wert aktualisieren (Beispielhaft)
	powerUsage.Set(42)

	// 4. HTTP-Endpunkt bereitstellen
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)
}
