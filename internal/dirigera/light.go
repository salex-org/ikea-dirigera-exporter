package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type lightMetric struct {
	isOnMetric             *prometheus.GaugeVec
	levelMetric            *prometheus.GaugeVec
	colorHueMetric         *prometheus.GaugeVec
	colorSaturationMetric  *prometheus.GaugeVec
	colorTemperatureMetric *prometheus.GaugeVec
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
		colorHueMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "light",
			Name:      "current_color_hue",
			Help:      "Current color hue of a light (degrees 0 - 360)",
		}, metricLabelNames),
		colorSaturationMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "light",
			Name:      "current_color_saturation",
			Help:      "Current color saturation of a light (0 - 1, 0 = white mode, >0 = color mode)",
		}, metricLabelNames),
		colorTemperatureMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "light",
			Name:      "current_color_temperature_kelvin",
			Help:      "Current color temperature of a light in kelvin (used only when in white mode)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.isOnMetric)
	prometheus.MustRegister(metric.levelMetric)
	prometheus.MustRegister(metric.colorHueMetric)
	prometheus.MustRegister(metric.colorSaturationMetric)
	prometheus.MustRegister(metric.colorTemperatureMetric)

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
	if colorHue, hasColorHue := device.Attributes["colorHue"].(float64); hasColorHue {
		m.colorHueMetric.With(labels).Set(colorHue)
	}
	if colorSaturation, hasColorSaturation := device.Attributes["colorSaturation"].(float64); hasColorSaturation {
		m.colorSaturationMetric.With(labels).Set(colorSaturation)
	}
	if colorTemperature, hasColorTemperature := device.Attributes["colorTemperature"].(float64); hasColorTemperature {
		m.colorTemperatureMetric.With(labels).Set(colorTemperature)
	}
}
