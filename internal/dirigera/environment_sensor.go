package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type environmentSensorMetric struct {
	temperatureMetric *prometheus.GaugeVec
	humidityMetric    *prometheus.GaugeVec
}

func newEnvironmentSensorMetric() dirigeraMetric {
	metric := &environmentSensorMetric{
		temperatureMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "environment_sensor",
			Name:      "current_temperature",
			Help:      "Current temperature measured by an environment sensor (degree celsius)",
		}, metricLabelNames),
		humidityMetric: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ikea",
			Subsystem: "environment_sensor",
			Name:      "current_humidity",
			Help:      "Current relative humidity measured by an environment sensor (percent)",
		}, metricLabelNames),
	}
	prometheus.MustRegister(metric.temperatureMetric)
	prometheus.MustRegister(metric.humidityMetric)

	return metric
}

func (m *environmentSensorMetric) update(device client.Device, labels prometheus.Labels) {
	if temperature, hasTemperature := device.Attributes["currentTemperature"].(float64); hasTemperature {
		m.temperatureMetric.With(labels).Set(temperature)
	}
	if humidity, hasHumidity := device.Attributes["currentRH"].(float64); hasHumidity {
		m.humidityMetric.With(labels).Set(humidity)
	}
}
