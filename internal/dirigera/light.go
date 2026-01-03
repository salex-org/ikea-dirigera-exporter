package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type lightMetric struct {
	isOnMetric  *prometheus.GaugeVec
	levelMetric *prometheus.GaugeVec
}

func newLightMetric() dirigeraMetric {
	metric := &lightMetric{
		isOnMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "light",
			Name:      "current_state",
			Help:      "Current switch state of a light (0 = off, 1 = on)",
		}, metricLabelNames),
		levelMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "light",
			Name:      "current_level",
			Help:      "Current brightness of a light (percent)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.isOnMetric)
	prometheus.MustRegister(metric.levelMetric)

	return metric
}

func (m *lightMetric) update(device client.Device, labels prometheus.Labels) {
	if isOn, hasIsOn := device.Attributes["isOn"].(bool); hasIsOn {
		var value float64 = 0
		if isOn {
			value = 1
		}
		m.isOnMetric.With(labels).Set(value)
	}
	if level, hasLevel := device.Attributes["lightLevel"].(float64); hasLevel {
		m.levelMetric.With(labels).Set(level)
	}
}
