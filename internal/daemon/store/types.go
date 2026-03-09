package store

import "encoding/json"

const (
	RouteTypeWebhook = "webhook"
	RouteTypeCron    = "cron"

	HealthOK       = "ok"
	HealthWarning  = "warning"
	HealthDegraded = "degraded"
	HealthFail     = "fail"

	RunQueued    = "queued"
	RunRunning   = "running"
	RunSucceeded = "succeeded"
	RunFailed    = "failed"
	RunRejected  = "rejected"
	RunDeduped   = "deduped"
	RunSkipped   = "skipped"

	AuthNone   = "none"
	AuthBearer = "bearer"
	AuthHMAC   = "hmac"

	OverlapSingleFlight = "single_flight"
	OverlapQueueOne     = "queue_one"
)

type Config struct {
	AdminSocketPath         string `json:"admin_socket_path"`
	WebhookHTTPAddr         string `json:"webhook_http_addr"`
	MaxBodyBytes            int64  `json:"max_body_bytes"`
	GlobalQueueCapacity     int    `json:"global_queue_capacity"`
	PerWorkspaceConcurrency int    `json:"per_workspace_concurrency"`
	DedupTTLSeconds         int    `json:"dedup_ttl_seconds"`
	SchedulerTickSeconds    int    `json:"scheduler_tick_seconds"`
	AllowInsecureAuthNone   bool   `json:"allow_insecure_auth_none"`
	ShutdownTimeoutSeconds  int    `json:"shutdown_timeout_seconds"`
}

type State struct {
	StartedAt           string   `json:"startedAt"`
	LastSchedulerTickAt string   `json:"lastSchedulerTickAt,omitempty"`
	Status              string   `json:"status"`
	DegradedSubsystems  []string `json:"degradedSubsystems"`
}

type WorkspaceRecord struct {
	ID            string `json:"id"`
	Path          string `json:"path"`
	DisplayName   string `json:"displayName"`
	Enabled       bool   `json:"enabled"`
	Health        string `json:"health"`
	HealthMessage string `json:"healthMessage,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	LastScannedAt string `json:"lastScannedAt,omitempty"`
}

type RouteRecord struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	WorkspaceID      string `json:"workspaceId"`
	EpicSlug         string `json:"epicSlug"`
	Provider         string `json:"provider,omitempty"`
	EndpointKey      string `json:"endpointKey,omitempty"`
	JobName          string `json:"jobName,omitempty"`
	CronExpr         string `json:"cronExpr,omitempty"`
	PreferredAdapter string `json:"preferredAdapter,omitempty"`
	PinnedAdapter    string `json:"pinnedAdapter,omitempty"`
	Enabled          bool   `json:"enabled"`
	OverlapPolicy    string `json:"overlapPolicy,omitempty"`
	AuthMode         string `json:"authMode"`
	BearerSecretRef  string `json:"bearerSecretRef,omitempty"`
	HMACSecretRef    string `json:"hmacSecretRef,omitempty"`
	HMACHeader       string `json:"hmacHeader,omitempty"`
	SelectedAdapter  string `json:"selectedAdapter,omitempty"`
	LastDeliveryAt   string `json:"lastDeliveryAt,omitempty"`
	LastSuccessAt    string `json:"lastSuccessAt,omitempty"`
	LastErrorAt      string `json:"lastErrorAt,omitempty"`
	LastErrorMessage string `json:"lastErrorMessage,omitempty"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type RunRecord struct {
	ID            string `json:"id"`
	RouteID       string `json:"routeId"`
	WorkspaceID   string `json:"workspaceId"`
	EpicSlug      string `json:"epicSlug"`
	TriggerType   string `json:"triggerType"`
	DedupKey      string `json:"dedupKey,omitempty"`
	Adapter       string `json:"adapter,omitempty"`
	ExecutorID    string `json:"executorId,omitempty"`
	Outcome       string `json:"outcome"`
	FailureReason string `json:"failureReason,omitempty"`
	EnqueuedAt    string `json:"enqueuedAt"`
	StartedAt     string `json:"startedAt,omitempty"`
	FinishedAt    string `json:"finishedAt,omitempty"`
	HTTPStatus    int    `json:"httpStatus,omitempty"`
	OutputPath    string `json:"outputPath,omitempty"`
}

type Event struct {
	RouteID     string            `json:"routeId"`
	TriggerType string            `json:"triggerType"`
	Provider    string            `json:"provider,omitempty"`
	ExternalID  string            `json:"externalId,omitempty"`
	DedupKey    string            `json:"dedupKey,omitempty"`
	OccurredAt  string            `json:"occurredAt"`
	Payload     json.RawMessage   `json:"payload"`
	Headers     map[string]string `json:"headers,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
