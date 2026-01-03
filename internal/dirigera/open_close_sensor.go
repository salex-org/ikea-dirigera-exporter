package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type openCloseSensorMetric struct {
	openCloseMetric *prometheus.GaugeVec
}

func newOpenCloseSensorMetric() dirigeraMetric {
	metric := &openCloseSensorMetric{
		openCloseMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "open_close_sensor",
			Name:      "current_state",
			Help:      "Current status of an open-close sensor (0 = closed, 1 = open)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.openCloseMetric)

	return metric
}

func (m *openCloseSensorMetric) update(device client.Device) {
	isOpen, hasIsOpen := device.Attributes["isOpen"].(bool)
	var value float64 = 0
	if hasIsOpen {
		if isOpen {
			value = 1
		}
	}
	m.openCloseMetric.With(createLabels(device)).Set(value)
}
