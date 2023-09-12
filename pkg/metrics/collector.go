package metrics

import (
	"sync"

	"k8s.io/component-base/metrics"
)

type Collector struct {
	metrics.BaseStableCollector
	lock sync.RWMutex

	storedMetrics    []metrics.Metric
	staleMetrics     []metrics.Metric
	markStaleMetrics bool
}

var _ metrics.StableCollector = &Collector{}

const (
	VersionLabel    = "version"
	BuildLabel      = "build"
	ApiVersionLabel = "api_version"
	vCenterUUID     = "uuid"

	HwVersionLabel   = "hw_version"
	cbtMismatchLabel = "cbt"
)

// Metrics in this package can be used when we expect metrics with certain label to stop
// being reported when cluster configuration changes.
var (
	EsxiVersionMetric = metrics.NewDesc(
		"vsphere_esxi_version_total",
		"Number of ESXi hosts with given version.",
		[]string{VersionLabel, ApiVersionLabel}, nil,
		metrics.ALPHA, "",
	)
	HwVersionMetric = metrics.NewDesc(
		"vsphere_node_hw_version_total",
		"Number of vSphere nodes with given HW version.",
		[]string{HwVersionLabel}, nil,
		metrics.ALPHA, "",
	)

	CbtMismatchMetric = metrics.NewDesc(
		"vsphere_vm_cbt_checks",
		"Boolean metric based on whether ctkEnabled is consistent or not across all nodes in the cluster.",
		[]string{cbtMismatchLabel}, nil,
		metrics.ALPHA, "",
	)
)

func NewMetricsCollector() *Collector {
	return &Collector{
		storedMetrics:    []metrics.Metric{},
		staleMetrics:     []metrics.Metric{},
		markStaleMetrics: false,
	}
}

func (c *Collector) DescribeWithStability(ch chan<- *metrics.Desc) {
	ch <- EsxiVersionMetric
	ch <- HwVersionMetric
	ch <- CbtMismatchMetric
}

func (c *Collector) CollectWithStability(ch chan<- metrics.Metric) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.markStaleMetrics {
		for _, m := range c.staleMetrics {
			ch <- m
		}
	} else {
		for _, m := range c.storedMetrics {
			ch <- m
		}
	}
}

func (c *Collector) AddMetric(m metrics.Metric) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.storedMetrics = append(c.storedMetrics, m)
}

func (c *Collector) ClearStoredMetric() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// if last check did not finish, we should keep reporting stale metrics rather than risk
	// clearing them out
	if !c.markStaleMetrics {
		c.markStaleMetrics = true
		c.staleMetrics = c.storedMetrics
	}
	c.storedMetrics = []metrics.Metric{}
}

// FinishedAllChecks updates staleMetrics with storedMetrics so as
// both slices point to same values
func (c *Collector) FinishedAllChecks() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.staleMetrics = c.storedMetrics
	c.markStaleMetrics = false
}
