package importing

import "atrakta/internal/util"

type CapabilityKind string

const (
	KindSkill           CapabilityKind = "skill"
	KindRecipeCandidate CapabilityKind = "recipe_candidate"
	KindReferenceMemory CapabilityKind = "reference_memory"
	KindGateway         CapabilityKind = "gateway"
	KindAPI             CapabilityKind = "api"
	KindUnsupported     CapabilityKind = "unsupported"
)

const (
	AnalysisPending  = "pending"
	AnalysisAnalyzed = "analyzed"

	ConversionNone             = "none"
	ConversionReviewPending    = "review_pending"
	ConversionReviewedApproved = "reviewed_approved"
	ConversionReviewedRejected = "reviewed_rejected"

	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

type LoadedFile struct {
	RelPath     string `json:"rel_path"`
	AbsPath     string `json:"-"`
	ContentHash string `json:"content_hash"`
	ByteSize    int64  `json:"byte_size"`
	Binary      bool   `json:"binary"`
	Executable  bool   `json:"executable"`
	SecretLike  bool   `json:"secret_like"`
	Content     string `json:"-"`
}

type LoadResult struct {
	SourceType    string       `json:"source_type"`
	SourcePath    string       `json:"source_path"`
	ImportBatchID string       `json:"import_batch_id"`
	Files         []LoadedFile `json:"files"`
}

type CapabilityProvenance struct {
	RootPath    string `json:"root_path"`
	SourcePath  string `json:"source_path"`
	ContentHash string `json:"content_hash"`
	ImportedAt  string `json:"imported_at"`
}

type CapabilityAnalysis struct {
	FilesystemAccess bool   `json:"filesystem_access"`
	NetworkAccess    bool   `json:"network_access"`
	SecretsAccess    bool   `json:"secrets_access"`
	Bounded          bool   `json:"bounded"`
	Summary          string `json:"summary"`
	Risk             string `json:"risk"`
}

type RecipeCandidate struct {
	TimeoutSec             int      `json:"timeout_sec"`
	MaxSteps               int      `json:"max_steps"`
	Allowlist              []string `json:"allowlist"`
	ApprovalRequired       bool     `json:"approval_required"`
	DeterministicInputNote string   `json:"deterministic_input_note"`
	InputContractRef       string   `json:"input_contract_ref,omitempty"`
}

type CapabilityEntry struct {
	ID                   string               `json:"id"`
	Kind                 CapabilityKind       `json:"kind"`
	Path                 string               `json:"path"`
	SourceType           string               `json:"source_type,omitempty"`
	SourcePath           string               `json:"source_path,omitempty"`
	ImportBatchID        string               `json:"import_batch_id,omitempty"`
	AnalysisStatus       string               `json:"analysis_status,omitempty"`
	QuarantineReason     string               `json:"quarantine_reason,omitempty"`
	ConversionStatus     string               `json:"conversion_status,omitempty"`
	DefaultMemorySurface string               `json:"default_memory_surface,omitempty"`
	CurrentMemorySurface string               `json:"current_memory_surface,omitempty"`
	ReviewStatus         string               `json:"review_status,omitempty"`
	Executable           bool                 `json:"executable,omitempty"`
	Denied               bool                 `json:"denied,omitempty"`
	DenyReason           string               `json:"deny_reason,omitempty"`
	ContentHash          string               `json:"content_hash,omitempty"`
	Provenance           CapabilityProvenance `json:"provenance"`
	Analysis             *CapabilityAnalysis  `json:"analysis,omitempty"`
	Recipe               *RecipeCandidate     `json:"recipe_candidate,omitempty"`
	UpdatedAt            string               `json:"updated_at,omitempty"`
}

type CapabilityRegistry struct {
	V       int               `json:"v"`
	Entries []CapabilityEntry `json:"entries"`
}

type ImportReport struct {
	V                    int      `json:"v"`
	ImportBatchID        string   `json:"import_batch_id"`
	SourceType           string   `json:"source_type"`
	SourcePath           string   `json:"source_path"`
	ImportedAt           string   `json:"imported_at"`
	ImportedCapabilities []string `json:"imported_capabilities"`
	DeniedCapabilities   []string `json:"denied_capabilities"`
	QuarantinedCount     int      `json:"quarantined_count"`
	PendingConversions   int      `json:"pending_conversions"`
	PendingMemoryReviews int      `json:"pending_memory_reviews"`
}

type MemoryReviewResult struct {
	Promoted bool   `json:"promoted"`
	Reason   string `json:"reason"`
}

type CatalogItem struct {
	CapabilityID string `json:"capability_id"`
	Kind         string `json:"kind"`
	SourcePath   string `json:"source_path"`
	ReviewStatus string `json:"review_status"`
	Attribution  string `json:"attribution"`
}

type Catalog struct {
	Items []CatalogItem `json:"items"`
}

type Decision struct {
	Promote bool   `json:"promote"`
	Reason  string `json:"reason"`
}

func normalizePath(p string) string {
	return util.NormalizeRelPath(p)
}
