package dirigera

import (
	"fmt"
	"ikea-dirigera-exporter/internal/util"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type Client interface {
	Start() error
	Shutdown() error
	Health() error
}

type dirigeraClient struct {
	hub               client.Client
	additionalMetrics map[string]map[string]dirigeraMetric
	baseMetrics       dirigeraMetric
}

func NewClient() (Client, error) {
	// Create client
	dirigeraAddress := util.ReadEnvVar("IKEA_ADDRESS")
	dirigeraPort, err := strconv.Atoi(util.ReadEnvVarWithDefault("IKEA_PORT", "8443"))
	if err != nil {
		return nil, fmt.Errorf("error parsing IKEA_PORT value: %w", err)
	}
	newClient := &dirigeraClient{
		hub: client.Connect(dirigeraAddress, dirigeraPort, &client.Authorization{
			AccessToken:    util.ReadEnvVar("IKEA_TOKEN"),
			TLSFingerprint: util.ReadEnvVar("IKEA_TLS_FINGERPRINT"),
		}),
	}

	// Register metrics
	newClient.baseMetrics = newBaseDeviceMetric()
	newClient.additionalMetrics = make(map[string]map[string]dirigeraMetric)
	newClient.additionalMetrics["sensor"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["sensor"]["openCloseSensor"] = newOpenCloseSensorMetric()
	newClient.additionalMetrics["sensor"]["environmentSensor"] = newEnvironmentSensorMetric()
	newClient.additionalMetrics["outlet"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["outlet"]["outlet"] = newOutletMetric()
	newClient.additionalMetrics["gateway"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["gateway"]["gateway"] = newGatewayMetric()

	// Register event handler
	newClient.hub.RegisterEventHandler(newClient.updateMetricFromEvent, "deviceStateChanged")

	// Load initial data
	devices, err := newClient.hub.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("error loading devices: %w", err)
	}
	for _, device := range devices {
		newClient.updateMetric(*device)
	}

	return newClient, nil
}

func (d *dirigeraClient) Start() error {
	return d.hub.ListenForEvents()
}

func (d *dirigeraClient) Shutdown() error {
	return d.hub.StopEventListening()
}

func (d *dirigeraClient) Health() error {
	return d.hub.GetEventLoopState()
}

func (d *dirigeraClient) updateMetric(device client.Device) {
	if detailTypes, typeFound := d.additionalMetrics[device.Type]; typeFound {
		if metric, metricFound := detailTypes[device.DetailedType]; metricFound {
			d.baseMetrics.update(device)
			metric.update(device)
			return
		}
	}
	fmt.Printf("Warning: No metric registered for %s:%s\n", device.Type, device.DetailedType)
}

func (d *dirigeraClient) updateMetricFromEvent(event client.Event) {
	d.updateMetric(event.Device)
}

type dirigeraMetric interface {
	update(device client.Device)
}

var metricLabelNames = []string{"device", "type", "device_type"}

func createLabels(device client.Device) prometheus.Labels {
	return prometheus.Labels{
		"device":      normalizeDeviceID(device.ID),
		"type":        device.Type,
		"device_type": device.DetailedType,
	}
}

func normalizeDeviceID(deviceID string) string {
	if idx := strings.LastIndex(deviceID, "_"); idx != -1 {
		return deviceID[:idx]
	}

	return deviceID
}

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

func (m *baseDeviceMetric) update(device client.Device) {
	var value float64 = 0
	if device.IsReachable {
		value = 1
	}
	m.reachableMetric.With(createLabels(device)).Set(value)
	m.lastSeenMetric.With(createLabels(device)).Set(float64(device.LastSeen.Unix()))
	if batteryLevel, hasBatteryLevel := device.Attributes["batteryPercentage"].(float64); hasBatteryLevel {
		m.batteryLevelMetric.With(createLabels(device)).Set(batteryLevel)
	}
}
