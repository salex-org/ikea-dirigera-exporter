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
	GetHubName() string
}

type dirigeraClient struct {
	hub               client.Client
	labelManager      *labelManager
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

	// Reading hubName
	hubStatus, err := newClient.hub.GetHubStatus()
	if err != nil {
		return nil, fmt.Errorf("error getting hub status: %w", err)
	}
	hubName, hasHubName := hubStatus.Attributes["customName"].(string)
	if !hasHubName {
		return nil, fmt.Errorf("error reading hub name: no custom name defined")
	}

	// Register metrics
	newClient.baseMetrics = newBaseDeviceMetric()
	newClient.additionalMetrics = make(map[string]map[string]dirigeraMetric)
	newClient.additionalMetrics["sensor"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["sensor"]["openCloseSensor"] = newOpenCloseSensorMetric()
	newClient.additionalMetrics["sensor"]["environmentSensor"] = newEnvironmentSensorMetric()
	newClient.additionalMetrics["outlet"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["outlet"]["outlet"] = newOutletMetric()
	newClient.additionalMetrics["controller"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["controller"]["lightController"] = newLightControllerMetric()
	newClient.additionalMetrics["light"] = make(map[string]dirigeraMetric)
	newClient.additionalMetrics["light"]["light"] = newLightMetric()

	// Register event handler
	newClient.hub.RegisterEventHandler(newClient.updateMetricFromEvent, "deviceStateChanged")

	// Load initial data
	devices, err := newClient.hub.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("error loading devices: %w", err)
	}
	newClient.labelManager = createLabelManager(hubName, devices)
	for _, device := range devices {
		newClient.updateMetric(*device, nil)
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

func (d *dirigeraClient) GetHubName() string {
	return d.labelManager.hubName
}

func (d *dirigeraClient) updateMetric(device client.Device, event *client.Event) {
	if device.Type == "gateway" {
		return // skipping gateway itself
	}

	if detailTypes, typeFound := d.additionalMetrics[device.Type]; typeFound {
		if metric, metricFound := detailTypes[device.DetailedType]; metricFound {
			labels, err := d.labelManager.createLabels(device)
			if err != nil {
				fmt.Printf("Warning: Could not create labels - skipping metric update: %v\n", err)
				return
			}
			d.baseMetrics.update(device, labels)
			metric.update(device, labels)
			return
		}
	}
	fmt.Printf("Warning: No metric registered for %s:%s\n", device.Type, device.DetailedType)
	if event != nil {
		fmt.Printf("Received event %v\n", event)
	}
}

func (d *dirigeraClient) updateMetricFromEvent(event client.Event) {
	d.updateMetric(event.Device, &event)
}

type dirigeraMetric interface {
	update(device client.Device, labels prometheus.Labels)
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

var metricLabelNames = []string{"hub", "device", "room", "type", "device_type"}

type deviceLabels struct {
	deviceName string
	roomName   string
}

type labelManager struct {
	hubName      string
	deviceLabels map[string]deviceLabels
}

func createLabelManager(hubName string, devices []*client.Device) *labelManager {
	newLabelManager := &labelManager{
		hubName:      hubName,
		deviceLabels: make(map[string]deviceLabels),
	}

	for _, device := range devices {
		if device.Type == "gateway" {
			continue // skipping gateway itself
		}
		deviceID, updateCache := normalizeDeviceID(device.ID)
		if updateCache {
			newDeviceName, hasDeviceName := device.Attributes["customName"]
			if !hasDeviceName {
				fmt.Printf("Warning: device %s has no name\n", device.ID)
				continue
			}
			if device.Room.Name == "" {
				fmt.Printf("Warning: device %s has no room name\n", device.ID)
			}
			newLabelManager.deviceLabels[deviceID] = deviceLabels{
				deviceName: newDeviceName.(string),
				roomName:   device.Room.Name,
			}
		}
	}

	return newLabelManager
}

func (lm *labelManager) createLabels(device client.Device) (prometheus.Labels, error) {
	deviceID, updateCache := normalizeDeviceID(device.ID)
	cachedLabels, hasCachedLabels := lm.deviceLabels[deviceID]

	if !hasCachedLabels {
		return nil, fmt.Errorf("no labels found in cache for device %s", device.ID)
	}

	if updateCache {
		updated := false
		if newDeviceName, hasDeviceName := device.Attributes["customName"]; hasDeviceName {
			cachedLabels.deviceName = newDeviceName.(string)
			updated = true
		}
		if device.Room.Name != "" && device.Room.Name != cachedLabels.roomName {
			cachedLabels.roomName = device.Room.Name
			updated = true
		}
		if updated {
			lm.deviceLabels[deviceID] = cachedLabels
		}
	}

	return prometheus.Labels{
		"hub":         lm.hubName,
		"device":      cachedLabels.deviceName,
		"room":        cachedLabels.roomName,
		"type":        device.Type,
		"device_type": device.DetailedType,
	}, nil
}

func normalizeDeviceID(deviceID string) (string, bool) {
	idx := strings.LastIndex(deviceID, "_")
	if idx == -1 {
		return deviceID, true
	}

	base := deviceID[:idx]
	suffix := deviceID[idx+1:]

	if suffix == "1" {
		return base, true
	}

	return base, false
}
