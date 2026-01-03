package dirigera

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/salex-org/ikea-dirigera-client/pkg/client"
)

// lightControllerMetric contains specific metrics for light controllers
// Attention: Currently there are no specific metrics for light controllers, just the baseDeviceMetric including
// the battery level
type lightControllerMetric struct{}

func newLightControllerMetric() dirigeraMetric {
	return &lightControllerMetric{}
}

func (m *lightControllerMetric) update(device client.Device, labels prometheus.Labels) {}
