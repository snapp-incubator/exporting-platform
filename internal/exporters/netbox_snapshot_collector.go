package exporters

import (
    "os"
    "strings"

dto "github.com/prometheus/client_model/go"
    "github.com/prometheus/client_golang/prometheus"
	    "github.com/prometheus/common/expfmt"

)

type NetboxSnapshotCollector struct {
    Path string
}

func NewNetBoxSnapshotCollector(path string) *NetboxSnapshotCollector {
    return &NetboxSnapshotCollector{Path: path}
}

func (c *NetboxSnapshotCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *NetboxSnapshotCollector) Collect(ch chan<- prometheus.Metric) {
    data, err := os.ReadFile(c.Path)
    if err != nil {
        c.collectFromMemory(ch)
        return
    }

    c.collectFromBytes(data, ch)
}

func (c *NetboxSnapshotCollector) collectFromMemory(ch chan<- prometheus.Metric) {
    lastSnapshotMu.RLock()
    data := append([]byte(nil), lastSnapshotMemory...)
    lastSnapshotMu.RUnlock()

    c.collectFromBytes(data, ch)
}

func (c *NetboxSnapshotCollector) collectFromBytes(data []byte, ch chan<- prometheus.Metric) {
    parser := expfmt.TextParser{}

    families, err := parser.TextToMetricFamilies(strings.NewReader(string(data)))
    if err != nil {
        return
    }

    for name, mf := range families {
        desc := prometheus.NewDesc(name, mf.GetHelp(), labelNames(mf), nil)

        for _, m := range mf.GetMetric() {
            value, labels := extractValueAndLabels(mf, m)

            metric, err := prometheus.NewConstMetric(desc, valueType(mf), value, labels...)
            if err == nil {
                ch <- metric
            }
        }
    }
}

func labelNames(mf *dto.MetricFamily) []string {
    if len(mf.Metric) == 0 {
        return nil
    }
    names := []string{}
    for _, lp := range mf.Metric[0].Label {
        names = append(names, lp.GetName())
    }
    return names
}

func extractValueAndLabels(mf *dto.MetricFamily, m *dto.Metric) (float64, []string) {
    labels := []string{}
    for _, lp := range m.Label {
        labels = append(labels, lp.GetValue())
    }

    switch mf.GetType() {
    case dto.MetricType_GAUGE:
        if m.Gauge != nil {
            return m.Gauge.GetValue(), labels
        }
    case dto.MetricType_COUNTER:
        if m.Counter != nil {
            return m.Counter.GetValue(), labels
        }
    }

    if m.Untyped != nil {
        return m.Untyped.GetValue(), labels
    }

    return 0, labels
}

func valueType(mf *dto.MetricFamily) prometheus.ValueType {
    switch mf.GetType() {
    case dto.MetricType_GAUGE:
        return prometheus.GaugeValue
    case dto.MetricType_COUNTER:
        return prometheus.CounterValue
    default:
        return prometheus.UntypedValue
    }
}
