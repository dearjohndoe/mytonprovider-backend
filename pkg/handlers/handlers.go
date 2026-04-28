// Package handlers — обработчики результатов агентских циклов из result-стримов.
// Каждый хендлер вызывается уже внутри активной транзакции и должен только
// применить writes; commit/rollback делает redisstream.Consumer.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"

	"mytonprovider-backend/pkg/jobs"
	"mytonprovider-backend/pkg/models/db"
	"mytonprovider-backend/pkg/redisstream"
	providersRepo "mytonprovider-backend/pkg/repositories/providers"
	systemRepo "mytonprovider-backend/pkg/repositories/system"
)

const masterWalletLastLTKey = "masterWalletLastLT"

type Set struct {
	logger     *slog.Logger
	providers  providersRepo.Repository
	system     systemRepo.Repository
}

func NewSet(
	logger *slog.Logger,
	providers providersRepo.Repository,
	system systemRepo.Repository,
) *Set {
	return &Set{
		logger:    logger,
		providers: providers,
		system:    system,
	}
}

// Handler возвращает redisstream.ResultHandler для указанного типа цикла.
// nil — если тип неизвестен.
func (s *Set) Handler(cycleType string) redisstream.ResultHandler {
	switch cycleType {
	case jobs.CycleScanMaster:
		return s.handleScanMaster
	case jobs.CycleScanWallets:
		return s.handleScanWallets
	case jobs.CycleResolveEndpoints:
		return s.handleResolveEndpoints
	case jobs.CycleProbeRates:
		return s.handleProbeRates
	case jobs.CycleInspectContracts:
		return s.handleInspectContracts
	case jobs.CycleCheckProofs:
		return s.handleCheckProofs
	case jobs.CycleLookupIPInfo:
		return s.handleLookupIPInfo
	default:
		return nil
	}
}

// ----- scan_master: новые провайдеры + masterWalletLastLT -----

func (s *Set) handleScanMaster(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.scan_master"

	var result jobs.ScanMasterResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	if len(result.NewProviders) > 0 {
		dtos := make([]db.ProviderCreate, 0, len(result.NewProviders))
		for _, p := range result.NewProviders {
			dtos = append(dtos, p.ToDB())
		}
		if err := s.providers.WithTx(tx).AddProviders(ctx, dtos); err != nil {
			return fmt.Errorf("%s: add providers: %w", op, err)
		}
	}

	if result.LastLT > 0 {
		if err := s.system.WithTx(tx).SetParam(
			ctx, masterWalletLastLTKey, strconv.FormatUint(result.LastLT, 10),
		); err != nil {
			return fmt.Errorf("%s: set last lt: %w", op, err)
		}
	}

	s.logger.Info(
		"scan_master applied",
		slog.Int("new_providers", len(result.NewProviders)),
		slog.Uint64("last_lt", result.LastLT),
	)
	return nil
}

// ----- scan_wallets: контракты + relations + per-wallet LT -----

func (s *Set) handleScanWallets(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.scan_wallets"

	var result jobs.ScanWalletsResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	repo := s.providers.WithTx(tx)

	if len(result.Contracts) > 0 {
		dtos := make([]db.StorageContract, 0, len(result.Contracts))
		for _, c := range result.Contracts {
			dtos = append(dtos, c.ToDB())
		}
		if err := repo.AddStorageContracts(ctx, dtos); err != nil {
			return fmt.Errorf("%s: add contracts: %w", op, err)
		}
	}

	if len(result.UpdatedWallets) > 0 {
		dtos := make([]db.ProviderWalletLT, 0, len(result.UpdatedWallets))
		for _, w := range result.UpdatedWallets {
			dtos = append(dtos, db.ProviderWalletLT{
				PubKey: w.PublicKey,
				LT:     w.LT,
			})
		}
		if err := repo.UpdateProvidersLT(ctx, dtos); err != nil {
			return fmt.Errorf("%s: update lt: %w", op, err)
		}
	}

	s.logger.Info(
		"scan_wallets applied",
		slog.Int("contracts", len(result.Contracts)),
		slog.Int("updated_wallets", len(result.UpdatedWallets)),
	)
	return nil
}

// ----- resolve_endpoints: upsert IP/port в providers -----

func (s *Set) handleResolveEndpoints(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.resolve_endpoints"

	var result jobs.ResolveEndpointsResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	if len(result.Endpoints) > 0 {
		dtos := make([]db.ProviderIP, 0, len(result.Endpoints))
		for _, ep := range result.Endpoints {
			dtos = append(dtos, ep.ToDB())
		}
		if err := s.providers.WithTx(tx).UpdateProvidersIPs(ctx, dtos); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	s.logger.Info(
		"resolve_endpoints applied",
		slog.Int("endpoints", len(result.Endpoints)),
		slog.Int("skipped", result.Skipped),
		slog.Int("failed", result.Failed),
	)
	return nil
}

// ----- probe_rates: rates + statuses -----

func (s *Set) handleProbeRates(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.probe_rates"

	var result jobs.ProbeRatesResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	repo := s.providers.WithTx(tx)

	if len(result.Statuses) > 0 {
		dtos := make([]db.ProviderStatusUpdate, 0, len(result.Statuses))
		for _, st := range result.Statuses {
			dtos = append(dtos, st.ToDB())
		}
		if err := repo.AddStatuses(ctx, dtos); err != nil {
			return fmt.Errorf("%s: add statuses: %w", op, err)
		}
	}

	if len(result.Rates) > 0 {
		dtos := make([]db.ProviderUpdate, 0, len(result.Rates))
		for _, r := range result.Rates {
			dtos = append(dtos, r.ToDB())
		}
		if err := repo.UpdateProviders(ctx, dtos); err != nil {
			return fmt.Errorf("%s: update providers: %w", op, err)
		}
	}

	s.logger.Info(
		"probe_rates applied",
		slog.Int("statuses", len(result.Statuses)),
		slog.Int("rates", len(result.Rates)),
	)
	return nil
}

// ----- inspect_contracts: rejected -----

func (s *Set) handleInspectContracts(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.inspect_contracts"

	var result jobs.InspectContractsResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	if len(result.Rejected) > 0 {
		dtos := make([]db.ContractToProviderRelation, 0, len(result.Rejected))
		for _, rel := range result.Rejected {
			dtos = append(dtos, rel.ToDB())
		}
		if err := s.providers.WithTx(tx).UpdateRejectedStorageContracts(ctx, dtos); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	s.logger.Info(
		"inspect_contracts applied",
		slog.Int("rejected", len(result.Rejected)),
		slog.Int("skipped", len(result.Skipped)),
	)
	return nil
}

// ----- check_proofs: proof_checks -----

func (s *Set) handleCheckProofs(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.check_proofs"

	var result jobs.CheckProofsResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	if len(result.Results) > 0 {
		dtos := make([]db.ContractProofsCheck, 0, len(result.Results))
		for _, p := range result.Results {
			dtos = append(dtos, p.ToDB())
		}
		if err := s.providers.WithTx(tx).UpdateContractProofsChecks(ctx, dtos); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	s.logger.Info(
		"check_proofs applied",
		slog.Int("results", len(result.Results)),
	)
	return nil
}

// ----- lookup_ipinfo: ip_info JSONB -----

func (s *Set) handleLookupIPInfo(ctx context.Context, tx pgx.Tx, env jobs.ResultEnvelope) error {
	const op = "handlers.lookup_ipinfo"

	var result jobs.LookupIPInfoResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		return fmt.Errorf("%s: decode: %w", op, err)
	}

	if len(result.Items) > 0 {
		dtos := make([]db.ProviderIPInfo, 0, len(result.Items))
		for _, it := range result.Items {
			info := it.Info
			info.IP = it.IP
			data, err := json.Marshal(info)
			if err != nil {
				return fmt.Errorf("%s: marshal info: %w", op, err)
			}
			dtos = append(dtos, db.ProviderIPInfo{
				PublicKey: it.PublicKey,
				IPInfo:    string(data),
			})
		}
		if err := s.providers.WithTx(tx).UpdateProvidersIPInfo(ctx, dtos); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	s.logger.Info(
		"lookup_ipinfo applied",
		slog.Int("items", len(result.Items)),
	)
	return nil
}
