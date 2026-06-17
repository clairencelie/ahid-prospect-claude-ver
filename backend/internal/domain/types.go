// Package domain holds the shared data types used across store, matching,
// core, enrichment and http packages, so none of them need to import each
// other just to pass records around.
package domain

import (
	"encoding/json"
	"time"
)

type Role string

const (
	RoleMarketing   Role = "marketing"
	RoleBranchCo    Role = "branch_co"
	RoleUnderwriter Role = "underwriter"
	RoleCompliance  Role = "compliance"
	RoleAudit       Role = "audit"
	RoleAdmin       Role = "admin"
)

type Band string

const (
	BandPass   Band = "PASS"
	BandReview Band = "REVIEW"
	BandBlock  Band = "BLOCK"
)

type VerificationStatus string

const (
	Verified   VerificationStatus = "verified"
	Unverified VerificationStatus = "unverified"
)

type Branch struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
}

type User struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Role     Role    `json:"role"`
	BranchID *string `json:"branch_id,omitempty"`
}

type Company struct {
	ID                 string             `json:"id"`
	NPWP               *string            `json:"npwp,omitempty"`
	NIB                *string            `json:"nib,omitempty"`
	LegalName          string             `json:"legal_name"`
	NormalizedName     string             `json:"normalized_name"`
	BrandNames         []string           `json:"brand_names"`
	Domain             *string            `json:"domain,omitempty"`
	Phone              *string            `json:"phone,omitempty"`
	Address            *string            `json:"address,omitempty"`
	OccupationLOB      *string            `json:"occupation_lob,omitempty"`
	GroupID            *string            `json:"group_id,omitempty"`
	Source             string             `json:"source"`
	Confidence         float64            `json:"confidence"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type CompanyRelationship struct {
	ID                 string             `json:"id"`
	ParentCompanyID    string             `json:"parent_company_id"`
	ChildCompanyID     string             `json:"child_company_id"`
	RelationType       string             `json:"relation_type"`
	Confidence         float64            `json:"confidence"`
	Source             string             `json:"source"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	CreatedAt          time.Time          `json:"created_at"`
}

type Prospect struct {
	ID              string     `json:"id"`
	CompanyID       *string    `json:"company_id,omitempty"`
	RawCompanyName  string     `json:"raw_company_name"`
	NPWP            *string    `json:"npwp,omitempty"`
	NIB             *string    `json:"nib,omitempty"`
	OccupationLOB   *string    `json:"occupation_lob,omitempty"`
	Address         *string    `json:"address,omitempty"`
	Domain          *string    `json:"domain,omitempty"`
	Phone           *string    `json:"phone,omitempty"`
	Headcount       *int       `json:"headcount,omitempty"`
	SourceBusiness  *string    `json:"source_business,omitempty"`
	BranchID        *string    `json:"branch_id,omitempty"`
	MarketingID     *string    `json:"marketing_id,omitempty"`
	Stage           string     `json:"stage"`
	Status          string     `json:"status"`
	TanggalMasukUW  *time.Time `json:"tanggal_masuk_uw,omitempty"`
	TanggalKeluarUW *time.Time `json:"tanggal_keluar_uw,omitempty"`
	UWRationale     *string    `json:"uw_rationale,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ProspectLock struct {
	ID            string    `json:"id"`
	ProspectID    string    `json:"prospect_id"`
	CompanyID     *string   `json:"company_id,omitempty"`
	NPWP          string    `json:"npwp"`
	BranchID      *string   `json:"branch_id,omitempty"`
	MarketingID   *string   `json:"marketing_id,omitempty"`
	LockedAt      time.Time `json:"locked_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"`
	ReleaseReason *string   `json:"release_reason,omitempty"`
}

type MatchedSignal struct {
	Signal       string  `json:"signal"`
	Similarity   float64 `json:"similarity"`
	Contribution float64 `json:"contribution"`
}

type Candidate struct {
	CompanyID      string          `json:"company_id"`
	LegalName      string          `json:"legal_name"`
	Score          float64         `json:"score"`
	MatchedSignals []MatchedSignal `json:"matched_signals"`
}

type MatchResult struct {
	ID                 string          `json:"id"`
	ProspectID         string          `json:"prospect_id"`
	CandidateCompanyID *string         `json:"candidate_company_id,omitempty"`
	CompositeScore     float64         `json:"composite_score"`
	Band               Band            `json:"band"`
	Reason             string          `json:"reason"`
	MatchedSignals     []MatchedSignal `json:"matched_signals"`
	DecidedBy          string          `json:"decided_by"`
	MakerID            *string         `json:"maker_id,omitempty"`
	CheckerID          *string         `json:"checker_id,omitempty"`
	Decision           *string         `json:"decision,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type EnrichmentJob struct {
	ID             string          `json:"id"`
	CompanyName    string          `json:"company_name"`
	NormalizedName string          `json:"normalized_name"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	RequestedAt    time.Time       `json:"requested_at"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

type Client struct {
	ID             string  `json:"id"`
	NPWP           *string `json:"npwp,omitempty"`
	ClientName     string  `json:"client_name"`
	OwningBranchID *string `json:"owning_branch_id,omitempty"`
}

type Policy struct {
	ID             string  `json:"id"`
	ClientID       *string `json:"client_id,omitempty"`
	NPWP           *string `json:"npwp,omitempty"`
	PolicyNo       string  `json:"policy_no"`
	Status         string  `json:"status"`
	PeriodStart    *string `json:"period_start,omitempty"`
	PeriodEnd      *string `json:"period_end,omitempty"`
	OwningBranchID *string `json:"owning_branch_id,omitempty"`
}

type AuditLog struct {
	ID         string          `json:"id"`
	ActorID    *string         `json:"actor_id,omitempty"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   *string         `json:"entity_id,omitempty"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}
