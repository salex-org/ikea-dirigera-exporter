package dirigera

import (
	"fmt"
	"ikea-dirigera-exporter/internal/util"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type DirigeraClient interface {
	Start() error
	Shutdown() error
	Health() error
}

func NewDirigeraClient() (DirigeraClient, error) {
	dirigeraAddress := util.ReadEnvVar("IKEA_ADDRESS")
	dirigeraPort, err := strconv.Atoi(util.ReadEnvVarWithDefault("IKEA_PORT", "8443"))
	if err != nil {
		return nil, fmt.Errorf("error parsing IKEA_PORT value: %w", err)
	}

	client := &dirigeraClient{
		client: client.Connect(dirigeraAddress, dirigeraPort, &client.Authorization{
			AccessToken:    util.ReadEnvVar("IKEA_TOKEN"),
			TLSFingerprint: util.ReadEnvVar("IKEA_TLS_FINGERPRINT"),
		}),
		powerUsageMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ikea_outlet_power_watts",
			Help: "Leistung die durch eine Steckdose im IKEA Smart-Home-System geht.",
		}),
		openCloseMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ikea_open_close_state",
			Help: "Zustand eines Open-Close-Sensors (0 = geschlossen, 1 = offen)",
		}, []string{"sensor"}),
	}

	prometheus.MustRegister(client.powerUsageMetric)
	prometheus.MustRegister(client.openCloseMetric)

	client.client.RegisterEventHandler(client.updateMetric, "deviceStateChanged")

	return client, nil
}

type dirigeraClient struct {
	client           client.Client
	powerUsageMetric prometheus.Gauge
	openCloseMetric  *prometheus.GaugeVec
}

func (d *dirigeraClient) Start() error {
	return d.client.ListenForEvents()
}

func (d *dirigeraClient) Shutdown() error {
	return d.client.StopEventListening()
}

func (d *dirigeraClient) Health() error {
	return d.client.GetEventLoopState()
}

func (d *dirigeraClient) updateMetric(event client.Event) {
	if event.Device.Type == "sensor" {
		if event.Device.DetailedType == "openCloseSensor" {
			state, hasState := event.Device.Attributes["isOpen"].(bool)
			var value float64 = 0
			if hasState {
				if state {
					value = 1
				}
			}
			d.openCloseMetric.With(prometheus.Labels{
				"sensor": event.Device.ID,
			}).Set(value)
		}
	}
}
