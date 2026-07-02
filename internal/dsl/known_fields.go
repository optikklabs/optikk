package dsl

// KnownField describes a scalar field accepted by the DSL parser.
// This is a Go port of web/src/features/explorer/search/knownFields.ts.
type KnownField struct {
	Key         string
	Label       string
	FieldType   string // "string", "number", "bool"
	Description string
}

// FindKnownField returns the field with the given key, or nil.
func FindKnownField(key string, fields []KnownField) *KnownField {
	for i := range fields {
		if fields[i].Key == key {
			return &fields[i]
		}
	}
	return nil
}

// TraceKnownFields mirrors TRACE_KNOWN_FIELDS from the web UI.
var TraceKnownFields = []KnownField{
	{Key: "service", Label: "Service", FieldType: "string", Description: "OTel service.name resource attribute"},
	{Key: "operation", Label: "Operation", FieldType: "string", Description: "Span operation name"},
	{Key: "span_kind", Label: "Span kind", FieldType: "string", Description: "SERVER / CLIENT / PRODUCER / CONSUMER / INTERNAL"},
	{Key: "http_method", Label: "HTTP method", FieldType: "string", Description: "GET / POST / …"},
	{Key: "http_status", Label: "HTTP status", FieldType: "number", Description: "Response status code (200, 404, 500…)"},
	{Key: "status", Label: "Status", FieldType: "string", Description: "OK / ERROR / UNSET"},
	{Key: "environment", Label: "Environment", FieldType: "string", Description: "deployment.environment resource attribute"},
	{Key: "peer_service", Label: "Peer service", FieldType: "string", Description: "Downstream service called by this span"},
	{Key: "trace_id", Label: "Trace ID", FieldType: "string", Description: "Hex-encoded trace identifier"},
	{Key: "duration_ms", Label: "Duration (ms)", FieldType: "number", Description: "Span duration in milliseconds"},
	{Key: "has_error", Label: "Has error", FieldType: "bool", Description: "Whether the span recorded an error"},
}

// LogKnownFields mirrors LOG_KNOWN_FIELDS from the web UI.
var LogKnownFields = []KnownField{
	{Key: "service_name", Label: "Service", FieldType: "string", Description: "OTel service.name resource attribute"},
	{Key: "severity_text", Label: "Severity", FieldType: "string", Description: "TRACE / DEBUG / INFO / WARN / ERROR / FATAL"},
	{Key: "body", Label: "Message body", FieldType: "string", Description: "Free-text search across log message body"},
	{Key: "trace_id", Label: "Trace ID", FieldType: "string", Description: "Hex-encoded trace identifier"},
	{Key: "span_id", Label: "Span ID", FieldType: "string", Description: "Hex-encoded span identifier"},
	{Key: "host", Label: "Host", FieldType: "string", Description: "Hostname emitting the log"},
	{Key: "pod", Label: "Pod", FieldType: "string", Description: "Kubernetes pod name"},
	{Key: "container", Label: "Container", FieldType: "string", Description: "Kubernetes container name"},
	{Key: "environment", Label: "Environment", FieldType: "string", Description: "deployment.environment resource attribute"},
}

// AIKnownFields mirrors AI_KNOWN_FIELDS from the web UI.
var AIKnownFields = []KnownField{
	{Key: "provider", Label: "Provider", FieldType: "string", Description: "GenAI provider name"},
	{Key: "model", Label: "Model", FieldType: "string", Description: "Request or response model"},
	{Key: "operation", Label: "Operation", FieldType: "string", Description: "GenAI operation name"},
	{Key: "spanType", Label: "Span type", FieldType: "string", Description: "MLflow-style AI span type"},
	{Key: "service", Label: "Service", FieldType: "string", Description: "OTel service.name resource attribute"},
	{Key: "environment", Label: "Environment", FieldType: "string", Description: "deployment.environment resource attribute"},
	{Key: "promptName", Label: "Prompt", FieldType: "string", Description: "Prompt tracking name"},
	{Key: "promptVersion", Label: "Prompt version", FieldType: "string", Description: "Prompt tracking version"},
	{Key: "agentName", Label: "Agent", FieldType: "string", Description: "Agent or workflow name"},
	{Key: "toolName", Label: "Tool", FieldType: "string", Description: "Tool call name"},
	{Key: "dataSource", Label: "Data source", FieldType: "string", Description: "Retrieval store or data source"},
}
