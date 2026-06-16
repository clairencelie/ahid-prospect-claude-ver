// Package core defines the contract for looking up existing clients/policies
// in the real core insurance system (§9 of the PRD). The demo only ships a
// mock implementation backed by the seeded clients/policies tables; swapping
// to the real core system later only requires a new CoreClient implementation.
package core

import "context"

type Client struct {
	ID             string
	NPWP           string
	ClientName     string
	OwningBranchID *string
}

type Policy struct {
	ID             string
	PolicyNo       string
	Status         string
	PeriodStart    *string
	PeriodEnd      *string
	OwningBranchID *string
}

type CoreClient interface {
	LookupClientByNPWP(ctx context.Context, npwp string) (*Client, error)
	ActivePoliciesByNPWP(ctx context.Context, npwp string) ([]Policy, error)
}
