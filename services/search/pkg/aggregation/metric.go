package aggregation

import (
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
)

func add(into, from *searchsvc.Metric) {
	if from.GetCount() == 0 {
		return
	}
	if into.GetCount() == 0 {
		into.Min, into.Max = from.GetMin(), from.GetMax()
	} else {
		into.Min, into.Max = min(into.GetMin(), from.GetMin()), max(into.GetMax(), from.GetMax())
	}
	into.Sum += from.GetSum()
	into.Count += from.GetCount()
}

// MetricValue reduces the accumulators to the value of the metric's kind;
// there is none without a single observed value.
func MetricValue(m *searchsvc.Metric) (float64, bool) {
	if m.GetCount() == 0 {
		return 0, false
	}
	switch m.GetKind() {
	case searchsvc.MetricKind_METRIC_KIND_SUM:
		return m.GetSum(), true
	case searchsvc.MetricKind_METRIC_KIND_MIN:
		return m.GetMin(), true
	case searchsvc.MetricKind_METRIC_KIND_MAX:
		return m.GetMax(), true
	case searchsvc.MetricKind_METRIC_KIND_AVG:
		return m.GetSum() / float64(m.GetCount()), true
	}
	return 0, false
}
