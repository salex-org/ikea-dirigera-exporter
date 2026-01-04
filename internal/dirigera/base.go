package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type baseDeviceMetric struct {
	reachableMetric    *prometheus.GaugeVec
	lastSeenMetric     *prometheus.GaugeVec
	batteryLevelMetric *prometheus.GaugeVec
}

func newBaseDeviceMetric() dirigeraMetric {
	metric := &baseDeviceMetric{
		reachableMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "device",
			Name:      "reachable",
			Help:      "Reachability of a device (0 = unreachable, 1 = reachable)",
		}, metricLabelNames),
		lastSeenMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "device",
			Name:      "last_seen_timestamp",
			Help:      "Last time the device was seen (Unix timestamp in seconds)",
		}, metricLabelNames),
		batteryLevelMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "device",
			Name:      "current_battery_level",
			Help:      "Current battery level of a device (percent)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.reachableMetric)
	prometheus.MustRegister(metric.lastSeenMetric)
	prometheus.MustRegister(metric.batteryLevelMetric)

	return metric
}

func (m *baseDeviceMetric) update(device client.Device, labels prometheus.Labels) {
	var value float64 = 0
	if device.IsReachable {
		value = 1
	}
	m.reachableMetric.With(labels).Set(value)
	m.lastSeenMetric.With(labels).Set(float64(device.LastSeen.Unix()))
	if batteryLevel, hasBatteryLevel := device.Attributes["batteryPercentage"].(float64); hasBatteryLevel {
		m.batteryLevelMetric.With(labels).Set(batteryLevel)
	}
}
