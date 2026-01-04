package dirigera

import (
	"fmt"
	"ikea-dirigera-exporter/internal/util"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

type DirigeraClient interface {
	Start() error
	Shutdown() error
	Health() error
	GetHubName() string
}

type dirigeraClient struct {
	hub               client.Client
	hubName           string
	hubID             string
	baseMetrics       dirigeraMetric
	additionalMetrics map[string]dirigeraMetric  // key: device type
	cache             map[string]*dirigeraDevice // key: normalized ID
}

type dirigeraDevice struct {
	deviceName string
	deviceType string
	roomName   string
	roomID     string
}

type dirigeraMetric interface {
	update(device client.Device, labels prometheus.Labels)
}

func NewDirigeraClient() (DirigeraClient, error) {
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
		cache:       make(map[string]*dirigeraDevice),
		baseMetrics: newBaseDeviceMetric(),
		additionalMetrics: map[string]dirigeraMetric{
			"openCloseSensor":   newOpenCloseSensorMetric(),
			"environmentSensor": newEnvironmentSensorMetric(),
			"outlet":            newOutletMetric(),
			"lightController":   newLightControllerMetric(),
			"light":             newLightMetric(),
		},
	}

	// Load hub information
	hubStatus, err := newClient.hub.GetHubStatus()
	if err != nil {
		return nil, fmt.Errorf("error loading hub status: %w", err)
	}
	hubName, hasHubName := hubStatus.Attributes["customName"]
	if !hasHubName {
		return nil, fmt.Errorf("hub %s has no customName", hubStatus.ID)
	}
	newClient.hubName = hubName.(string)
	newClient.hubID, _ = normalizeID(hubStatus.ID)

	// Register event handler
	newClient.hub.RegisterEventHandler(newClient.updateMetricFromEvent, "deviceStateChanged")

	// Load initial data
	devices, err := newClient.hub.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("error loading devices: %w", err)
	}
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
	return d.hubName
}

func (d *dirigeraClient) updateMetric(device client.Device, event *client.Event) {
	if device.DetailedType == "gateway" {
		return // skipping gateway itself
	}
	deviceID, _ := normalizeID(device.ID)

	cachedDevice, err := d.readFromCache(device, deviceID)
	if err != nil {
		fmt.Printf("Warning: Could not read from cache: %v\n", err)
		return
	}

	if metric, metricFound := d.additionalMetrics[cachedDevice.deviceType]; metricFound {
		labels := d.createLabels(cachedDevice, deviceID)
		d.baseMetrics.update(device, labels)
		metric.update(device, labels)
		return
	}
	fmt.Printf("Warning: No metric registered for %s:%s\n", device.Type, device.DetailedType)
	if event != nil {
		fmt.Printf("Received event %v\n", event)
	}
}

func (d *dirigeraClient) updateMetricFromEvent(event client.Event) {
	d.updateMetric(event.Device, &event)
}

func (d *dirigeraClient) readFromCache(device client.Device, deviceID string) (*dirigeraDevice, error) {
	cachedDevice, isCached := d.cache[deviceID]

	if !isCached {
		return d.addToCache(device)
	}

	return cachedDevice, nil
}

func (d *dirigeraClient) addToCache(device client.Device) (*dirigeraDevice, error) {
	rootDeviceID := device.ID
	deviceID, isRoot := normalizeID(device.ID)
	if !isRoot { // read deviceDetails from attached major device to ensure correct names and rooms
		rootDeviceID = fmt.Sprintf("%s_1", deviceID)
	}
	rootDevice, err := d.hub.GetDevice(rootDeviceID) // read deviceDetails from hub to ensure completeness
	if err != nil {
		return nil, fmt.Errorf("error getting device details for device %s: %w", rootDeviceID, err)
	}
	deviceName, hasDeviceName := rootDevice.Attributes["customName"].(string)
	if !hasDeviceName {
		return nil, fmt.Errorf("device %s has no customName", rootDeviceID)
	}
	if device.Room.Name == "" {
		return nil, fmt.Errorf("device %s has no room name", rootDeviceID)
	}
	cachedDevice := &dirigeraDevice{
		deviceName: deviceName,
		deviceType: rootDevice.DetailedType,
		roomName:   rootDevice.Room.Name,
		roomID:     rootDevice.Room.ID,
	}
	d.cache[deviceID] = cachedDevice

	return cachedDevice, nil
}

var metricLabelNames = []string{"hub_id", "hub_name", "room_id", "room_name", "device_id", "device_name", "device_type"}

func (d *dirigeraClient) createLabels(device *dirigeraDevice, deviceID string) prometheus.Labels {
	return prometheus.Labels{
		"hub_id":      d.hubID,
		"hub_name":    d.hubName,
		"room_id":     device.roomID,
		"room_name":   device.roomName,
		"device_id":   deviceID,
		"device_name": device.deviceName,
		"device_type": device.deviceType,
	}
}

// normalizeID cuts of the suffix with '_' at the end of the id if present.
// Returns the id without suffix.
// Also returns a flag indicating if it is the first/only id:
// * suffix '_1' or  no suffix: `true`
// * suffix other than '_1': 'false'
func normalizeID(id string) (string, bool) {
	idx := strings.LastIndex(id, "_")
	if idx == -1 {
		return id, true
	}

	base := id[:idx]
	suffix := id[idx+1:]

	if suffix == "1" {
		return base, true
	}

	return base, false
}
