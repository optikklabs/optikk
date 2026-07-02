package dsl

import (
	"strconv"
	"strings"
)

// TraceFilters mirrors query/internal/modules/traces/filter.Filters for use
// by the CLI. We duplicate the struct to avoid importing the query module
// (which pulls in ClickHouse).
type TraceFilters struct {
	Services      []string     `json:"services,omitempty"`
	Operations    []string     `json:"operations,omitempty"`
	SpanKinds     []string     `json:"spanKinds,omitempty"`
	HTTPMethods   []string     `json:"httpMethods,omitempty"`
	HTTPStatuses  []string     `json:"httpStatuses,omitempty"`
	Statuses      []string     `json:"statuses,omitempty"`
	Environments  []string     `json:"environments,omitempty"`
	PeerServices  []string     `json:"peerServices,omitempty"`
	TraceID       string       `json:"traceId,omitempty"`
	MinDurationNs int64        `json:"minDurationNs,omitempty"`
	MaxDurationNs int64        `json:"maxDurationNs,omitempty"`
	HasError      *bool        `json:"hasError,omitempty"`
	Search        string       `json:"search,omitempty"`
	Attributes    []AttrFilter `json:"attributes,omitempty"`

	ExcludeServices []string `json:"excludeServices,omitempty"`
	ExcludeStatuses []string `json:"excludeStatuses,omitempty"`
}

// LogFilters mirrors query/internal/modules/logs/filter.Filters.
type LogFilters struct {
	Services     []string     `json:"services,omitempty"`
	Hosts        []string     `json:"hosts,omitempty"`
	Pods         []string     `json:"pods,omitempty"`
	Containers   []string     `json:"containers,omitempty"`
	Environments []string     `json:"environments,omitempty"`
	Severities   []string     `json:"severities,omitempty"`
	TraceID      string       `json:"traceId,omitempty"`
	SpanID       string       `json:"spanId,omitempty"`
	Search       string       `json:"search,omitempty"`
	Attributes   []AttrFilter `json:"attributes,omitempty"`

	ExcludeServices   []string `json:"excludeServices,omitempty"`
	ExcludeHosts      []string `json:"excludeHosts,omitempty"`
	ExcludeSeverities []string `json:"excludeSeverities,omitempty"`
}

// AttrFilter is a custom attribute filter (@ prefixed fields).
type AttrFilter struct {
	Key   string `json:"key"`
	Op    string `json:"op,omitempty"`
	Value string `json:"value"`
}

// MapToTraceFilters converts parsed DSL filters to the traces API request shape.
func MapToTraceFilters(filters []Filter) TraceFilters {
	var tf TraceFilters
	for _, f := range filters {
		// Custom attribute filter.
		if strings.HasPrefix(f.Field, "@") {
			tf.Attributes = append(tf.Attributes, AttrFilter{
				Key:   f.Field[1:],
				Op:    string(f.Op),
				Value: f.Value,
			})
			continue
		}

		switch f.Field {
		case "search":
			tf.Search = appendSearch(tf.Search, f.Value)
		case "service":
			tf = mapStringFilter(tf, f, func(t *TraceFilters, vals []string) { t.Services = append(t.Services, vals...) },
				func(t *TraceFilters, vals []string) { t.ExcludeServices = append(t.ExcludeServices, vals...) })
		case "operation":
			addStrings(&tf.Operations, f)
		case "span_kind":
			addStrings(&tf.SpanKinds, f)
		case "http_method":
			addStrings(&tf.HTTPMethods, f)
		case "http_status":
			addStrings(&tf.HTTPStatuses, f)
		case "status":
			tf = mapStringFilter(tf, f, func(t *TraceFilters, vals []string) { t.Statuses = append(t.Statuses, vals...) },
				func(t *TraceFilters, vals []string) { t.ExcludeStatuses = append(t.ExcludeStatuses, vals...) })
		case "environment":
			addStrings(&tf.Environments, f)
		case "peer_service":
			addStrings(&tf.PeerServices, f)
		case "trace_id":
			tf.TraceID = f.Value
		case "duration_ms":
			ms, _ := strconv.ParseFloat(f.Value, 64)
			ns := int64(ms * 1e6)
			switch f.Op {
			case OpGte, OpGt:
				tf.MinDurationNs = ns
			case OpLte, OpLt:
				tf.MaxDurationNs = ns
			case OpEq:
				tf.MinDurationNs = ns
				tf.MaxDurationNs = ns
			}
		case "has_error":
			b := strings.EqualFold(f.Value, "true") || f.Value == "1"
			if f.Op == OpNeq {
				b = !b
			}
			tf.HasError = &b
		}
	}
	return tf
}

// MapToLogFilters converts parsed DSL filters to the logs API request shape.
func MapToLogFilters(filters []Filter) LogFilters {
	var lf LogFilters
	for _, f := range filters {
		if strings.HasPrefix(f.Field, "@") {
			lf.Attributes = append(lf.Attributes, AttrFilter{
				Key:   f.Field[1:],
				Op:    string(f.Op),
				Value: f.Value,
			})
			continue
		}

		switch f.Field {
		case "search":
			lf.Search = appendSearch(lf.Search, f.Value)
		case "service_name":
			lf = mapStringFilterLogs(lf, f,
				func(t *LogFilters, vals []string) { t.Services = append(t.Services, vals...) },
				func(t *LogFilters, vals []string) { t.ExcludeServices = append(t.ExcludeServices, vals...) })
		case "severity_text":
			lf = mapStringFilterLogs(lf, f,
				func(t *LogFilters, vals []string) { t.Severities = append(t.Severities, vals...) },
				func(t *LogFilters, vals []string) { t.ExcludeSeverities = append(t.ExcludeSeverities, vals...) })
		case "host":
			lf = mapStringFilterLogs(lf, f,
				func(t *LogFilters, vals []string) { t.Hosts = append(t.Hosts, vals...) },
				func(t *LogFilters, vals []string) { t.ExcludeHosts = append(t.ExcludeHosts, vals...) })
		case "pod":
			addStringsTo(&lf.Pods, f)
		case "container":
			addStringsTo(&lf.Containers, f)
		case "environment":
			addStringsTo(&lf.Environments, f)
		case "trace_id":
			lf.TraceID = f.Value
		case "span_id":
			lf.SpanID = f.Value
		case "body":
			lf.Search = appendSearch(lf.Search, f.Value)
		}
	}
	return lf
}

// --- helpers ---

func appendSearch(existing, val string) string {
	if existing == "" {
		return val
	}
	return existing + " " + val
}

func addStrings(target *[]string, f Filter) {
	switch f.Op {
	case OpIn, OpNotIn:
		*target = append(*target, strings.Split(f.Value, ",")...)
	default:
		*target = append(*target, f.Value)
	}
}

func addStringsTo(target *[]string, f Filter) {
	switch f.Op {
	case OpIn, OpNotIn:
		*target = append(*target, strings.Split(f.Value, ",")...)
	default:
		*target = append(*target, f.Value)
	}
}

func mapStringFilter(tf TraceFilters, f Filter,
	include func(*TraceFilters, []string),
	exclude func(*TraceFilters, []string),
) TraceFilters {
	vals := splitValues(f)
	switch f.Op {
	case OpNeq, OpNotIn, OpNotContains:
		exclude(&tf, vals)
	default:
		include(&tf, vals)
	}
	return tf
}

func mapStringFilterLogs(lf LogFilters, f Filter,
	include func(*LogFilters, []string),
	exclude func(*LogFilters, []string),
) LogFilters {
	vals := splitValues(f)
	switch f.Op {
	case OpNeq, OpNotIn, OpNotContains:
		exclude(&lf, vals)
	default:
		include(&lf, vals)
	}
	return lf
}

func splitValues(f Filter) []string {
	switch f.Op {
	case OpIn, OpNotIn:
		return strings.Split(f.Value, ",")
	default:
		return []string{f.Value}
	}
}
