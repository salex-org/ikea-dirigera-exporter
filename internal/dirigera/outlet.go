package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type outletMetric struct {
	isOnMetric               *prometheus.GaugeVec
	currentVoltageMetric     *prometheus.GaugeVec
	currentAmpsMetric        *prometheus.GaugeVec
	currentActivePowerMetric *prometheus.GaugeVec
}

func newOutletMetric() dirigeraMetric {
	metric := &outletMetric{
		isOnMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "outlet",
			Name:      "current_state",
			Help:      "Current switch state of an outlet (0 = off, 1 = on)",
		}, metricLabelNames),
		currentVoltageMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "outlet",
			Name:      "current_voltage",
			Help:      "Voltage currently applied to an oputlet (volts)",
		}, metricLabelNames),
		currentAmpsMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "outlet",
			Name:      "current_amps",
			Help:      "Amps currently consumed by an outlet - consumers and outlet itself (amps)",
		}, metricLabelNames),
		currentActivePowerMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "outlet",
			Name:      "current_active_power",
			Help:      "Power currently consumed at an outlet - consumers only (watts)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.isOnMetric)
	prometheus.MustRegister(metric.currentVoltageMetric)
	prometheus.MustRegister(metric.currentAmpsMetric)
	prometheus.MustRegister(metric.currentActivePowerMetric)

	return metric
}

func (m *outletMetric) update(device client.Device) {
	if isOn, hasIsOn := device.Attributes["isOn"].(bool); hasIsOn {
		var value float64 = 0
		if isOn {
			value = 1
		}
		m.isOnMetric.With(createLabels(device)).Set(value)
	}
	if voltage, hasVoltage := device.Attributes["currentVoltage"].(float64); hasVoltage {
		m.currentVoltageMetric.With(createLabels(device)).Set(voltage)
	}
	if amps, hasAmps := device.Attributes["currentAmps"].(float64); hasAmps {
		m.currentAmpsMetric.With(createLabels(device)).Set(amps)
	}
	if power, hasPower := device.Attributes["currentActivePower"].(float64); hasPower {
		m.currentActivePowerMetric.With(createLabels(device)).Set(power)
	}
}
