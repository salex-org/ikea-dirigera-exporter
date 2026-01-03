package dirigera

import (
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

// gatewayMetric contains specific metrics for the DIRIGERA Hub itself
// Attention: Currently there are no specific metrics for the hub, just the basicDeviceMetric
type gatewayMetric struct{}

func newGatewayMetric() dirigeraMetric {
	return &gatewayMetric{}
}

func (m *gatewayMetric) update(device client.Device) {}
