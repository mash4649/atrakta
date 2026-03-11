package model

type DetectReason string

const (
	ReasonExplicit      DetectReason = "explicit"
	ReasonTriggered     DetectReason = "triggered"
	ReasonAutoLast      DetectReason = "auto_last"
	ReasonObservedExact DetectReason = "observed_exact"
	ReasonUnknown       DetectReason = "unknown"
	ReasonConflict      DetectReason = "conflict"
	ReasonMixed         DetectReason = "mixed"
)

type DetectResult struct {
	Signals      map[string]any `json:"signals"`
	TargetSet    []string       `json:"target_set"`
	PruneAllowed bool           `json:"prune_allowed"`
	Reason       DetectReason   `json:"reason"`
}

type Operation struct {
	Op               string   `json:"op"`
	Path             string   `json:"path"`
	TaskID           string   `json:"task_id,omitempty"`
	TaskBlockedBy    []string `json:"task_blocked_by,omitempty"`
	RequiresApproval bool     `json:"requires_approval"`
	Target           string   `json:"target,omitempty"`
	Source           string   `json:"source,omitempty"`
	Fingerprint      string   `json:"fingerprint,omitempty"`
	Interface        string   `json:"interface,omitempty"`
	TemplateID       string   `json:"template_id,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

type Permission string

const (
	PermissionReadOnly       Permission = "read_only"
	PermissionWorkspaceWrite Permission = "workspace_write"
	PermissionFull           Permission = "full"
)

type PlanResult struct {
	ID                 string      `json:"id"`
	TaskGraphID        string      `json:"task_graph_id,omitempty"`
	TaskCount          int         `json:"task_count,omitempty"`
	TaskEdgeCount      int         `json:"task_edge_count,omitempty"`
	FeatureID          string      `json:"feature_id,omitempty"`
	Ops                []Operation `json:"ops"`
	RequiredPermission Permission  `json:"required_permission,omitempty"`
	Summary            string      `json:"summary"`
	Details            string      `json:"details"`
	RequiresApproval   bool        `json:"requires_approval"`
	ApprovalContext    any         `json:"approval_context,omitempty"`
}

type OpResult struct {
	TaskID      string `json:"task_id,omitempty"`
	Path        string `json:"path"`
	Op          string `json:"op"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	Interface   string `json:"interface,omitempty"`
	TemplateID  string `json:"template_id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Target      string `json:"target,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type ApplyResult struct {
	PlanID    string     `json:"plan_id"`
	FeatureID string     `json:"feature_id,omitempty"`
	Result    string     `json:"result"`
	Ops       []OpResult `json:"ops"`
}

type GateState string

const (
	GatePass GateState = "pass"
	GateFail GateState = "fail"
	GateSkip GateState = "skip"
)

type GateResult struct {
	Safety GateState `json:"safety"`
	Quick  GateState `json:"quick"`
	Reason string    `json:"reason"`
}

type NextAction struct {
	Kind     string `json:"kind"`
	Hint     string `json:"hint"`
	Command  string `json:"command,omitempty"`
	UIAction string `json:"ui_action,omitempty"`
}

type StepEvent struct {
	ActorRole  string     `json:"actor_role"`
	TaskID     string     `json:"task_id"`
	Outcome    string     `json:"outcome"`
	Gate       GateResult `json:"gate"`
	NextAction NextAction `json:"next_action"`
}

type SyncProposal struct {
	Needed           bool            `json:"needed"`
	Prefer           []string        `json:"prefer,omitempty"`
	Disable          []string        `json:"disable,omitempty"`
	Allowed          []SyncFieldDiff `json:"allowed,omitempty"`
	Denied           []SyncFieldDiff `json:"denied,omitempty"`
	RequiresApproval bool            `json:"requires_approval"`
	Summary          string          `json:"summary"`
}

type SyncFieldDiff struct {
	Field  string `json:"field"`
	Status string `json:"status"`
	From   any    `json:"from,omitempty"`
	To     any    `json:"to,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ProjectionManifest struct {
	V       int                       `json:"v"`
	Entries []ProjectionManifestEntry `json:"entries"`
}

type ProjectionManifestEntry struct {
	Interface  string   `json:"interface"`
	Kind       string   `json:"kind"`
	Files      []string `json:"files"`
	SourceHash string   `json:"source_hash"`
	RenderHash string   `json:"render_hash"`
	Status     string   `json:"status"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

type ExtensionManifest struct {
	V       int                      `json:"v"`
	Entries []ExtensionManifestEntry `json:"entries"`
}

type ExtensionManifestEntry struct {
	Kind       string   `json:"kind"`
	ID         string   `json:"id"`
	Files      []string `json:"files"`
	SourceHash string   `json:"source_hash"`
	RenderHash string   `json:"render_hash"`
	Status     string   `json:"status"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}
