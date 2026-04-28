package jobs

import (
	"encoding/json"
	"time"

	"mytonprovider-backend/pkg/constants"
	"mytonprovider-backend/pkg/models/db"
)

// Cycle types — должны совпадать с ключами стримов на стороне агента.
const (
	CycleScanMaster       = "scan_master"
	CycleScanWallets      = "scan_wallets"
	CycleResolveEndpoints = "resolve_endpoints"
	CycleProbeRates       = "probe_rates"
	CycleInspectContracts = "inspect_contracts"
	CycleCheckProofs      = "check_proofs"
	CycleLookupIPInfo     = "lookup_ipinfo"
)

// AllCycles — порядок имеет смысл только для bootstrap (EnsureGroup).
var AllCycles = []string{
	CycleScanMaster,
	CycleScanWallets,
	CycleResolveEndpoints,
	CycleProbeRates,
	CycleInspectContracts,
	CycleCheckProofs,
	CycleLookupIPInfo,
}

// TriggerEnvelope — сообщение в `mtpa:cycle:<type>`. Бэкенд кладёт, агент читает.
type TriggerEnvelope struct {
	JobID      string          `json:"job_id"`
	Type       string          `json:"type"`
	Hint       json.RawMessage `json:"hint,omitempty"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
}

const (
	StatusOK    = "ok"
	StatusError = "error"
)

// ResultEnvelope — сообщение в `mtpa:result:<type>`. Агент кладёт, бэкенд читает.
type ResultEnvelope struct {
	JobID       string          `json:"job_id"`
	Type        string          `json:"type"`
	Status      string          `json:"status"`
	Error       string          `json:"error,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	ProcessedAt time.Time       `json:"processed_at"`
	AgentID     string          `json:"agent_id"`
}

// ----- per-cycle result payloads (зеркало агентских типов) -----

type ProviderInfo struct {
	PublicKey    string    `json:"public_key"`
	Address      string    `json:"address"`
	LT           uint64    `json:"lt"`
	RegisteredAt time.Time `json:"registered_at"`
}

type Endpoint struct {
	PublicKey []byte `json:"public_key"`
	IP        string `json:"ip"`
	Port      int32  `json:"port"`
}

type ProviderEndpoint struct {
	PublicKey string    `json:"public_key"`
	Provider  Endpoint  `json:"provider"`
	Storage   Endpoint  `json:"storage"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProviderStatus struct {
	PublicKey string    `json:"public_key"`
	IsOnline  bool      `json:"is_online"`
	CheckedAt time.Time `json:"checked_at"`
}

type Rates struct {
	RatePerMBDay int64  `json:"rate_per_mb_day"`
	MinBounty    int64  `json:"min_bounty"`
	MinSpan      uint32 `json:"min_span"`
	MaxSpan      uint32 `json:"max_span"`
}

type IpInfo struct {
	Country    string `json:"country"`
	CountryISO string `json:"country_iso"`
	City       string `json:"city"`
	TimeZone   string `json:"timezone"`
	IP         string `json:"ip"`
}

type StorageContract struct {
	Address   string   `json:"address"`
	BagID     string   `json:"bag_id"`
	OwnerAddr string   `json:"owner_address"`
	Size      uint64   `json:"size"`
	ChunkSize uint64   `json:"chunk_size"`
	LastLT    uint64   `json:"last_tx_lt"`
	Providers []string `json:"providers"`
}

type ContractProviderRelation struct {
	ContractAddr    string `json:"contract_address"`
	ProviderPubkey  string `json:"provider_public_key"`
	ProviderAddress string `json:"provider_address"`
	BagID           string `json:"bag_id"`
	Size            uint64 `json:"size"`
}

type ProofResult struct {
	ContractAddr string               `json:"contract_address"`
	ProviderAddr string               `json:"provider_address"`
	Reason       constants.ReasonCode `json:"reason"`
	CheckedAt    time.Time            `json:"checked_at"`
}

// ----- результаты циклов -----

type ScanMasterResult struct {
	NewProviders []ProviderInfo `json:"new_providers"`
	LastLT       uint64         `json:"last_lt"`
	ScannedCount int            `json:"scanned_count"`
}

type ScanWalletsResult struct {
	Contracts      []StorageContract          `json:"contracts"`
	Relations      []ContractProviderRelation `json:"relations"`
	UpdatedWallets []ProviderInfo             `json:"updated_wallets"`
}

type ResolveEndpointsResult struct {
	Endpoints []ProviderEndpoint `json:"endpoints"`
	Skipped   int                `json:"skipped"`
	Failed    int                `json:"failed"`
}

type ProviderRateUpdate struct {
	PublicKey string `json:"public_key"`
	Rates     Rates  `json:"rates"`
}

type ProbeRatesResult struct {
	Statuses []ProviderStatus     `json:"statuses"`
	Rates    []ProviderRateUpdate `json:"rates"`
}

type InspectContractsResult struct {
	Rejected []ContractProviderRelation `json:"rejected"`
	Skipped  []string                   `json:"skipped_addrs"`
}

type CheckProofsResult struct {
	Results []ProofResult `json:"results"`
}

type IPInfoUpdate struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
	Info      IpInfo `json:"info"`
}

type LookupIPInfoResult struct {
	Items []IPInfoUpdate `json:"items"`
}

// ----- helper'ы конвертации в db-структуры -----

func (p ProviderInfo) ToDB() db.ProviderCreate {
	return db.ProviderCreate{
		Pubkey:       p.PublicKey,
		Address:      p.Address,
		RegisteredAt: p.RegisteredAt,
	}
}

func (s StorageContract) ToDB() db.StorageContract {
	addrs := make(map[string]struct{}, len(s.Providers))
	for _, a := range s.Providers {
		addrs[a] = struct{}{}
	}
	return db.StorageContract{
		ProvidersAddresses: addrs,
		Address:            s.Address,
		BagID:              s.BagID,
		OwnerAddr:          s.OwnerAddr,
		Size:               s.Size,
		ChunkSize:          s.ChunkSize,
		LastLT:             s.LastLT,
	}
}

func (r ContractProviderRelation) ToDB() db.ContractToProviderRelation {
	return db.ContractToProviderRelation{
		ProviderPublicKey: r.ProviderPubkey,
		ProviderAddress:   r.ProviderAddress,
		Address:           r.ContractAddr,
		BagID:             r.BagID,
		Size:              r.Size,
	}
}

func (p ProofResult) ToDB() db.ContractProofsCheck {
	return db.ContractProofsCheck{
		ContractAddress: p.ContractAddr,
		ProviderAddress: p.ProviderAddr,
		Reason:          p.Reason,
	}
}

func (s ProviderStatus) ToDB() db.ProviderStatusUpdate {
	return db.ProviderStatusUpdate{
		Pubkey:   s.PublicKey,
		IsOnline: s.IsOnline,
	}
}

func (r ProviderRateUpdate) ToDB() db.ProviderUpdate {
	return db.ProviderUpdate{
		Pubkey:       r.PublicKey,
		RatePerMBDay: r.Rates.RatePerMBDay,
		MinBounty:    r.Rates.MinBounty,
		MinSpan:      r.Rates.MinSpan,
		MaxSpan:      r.Rates.MaxSpan,
	}
}

func (e ProviderEndpoint) ToDB() db.ProviderIP {
	return db.ProviderIP{
		PublicKey: e.PublicKey,
		Storage: db.IPInfo{
			PublicKey: e.Storage.PublicKey,
			IP:        e.Storage.IP,
			Port:      e.Storage.Port,
		},
		Provider: db.IPInfo{
			PublicKey: e.Provider.PublicKey,
			IP:        e.Provider.IP,
			Port:      e.Provider.Port,
		},
	}
}
