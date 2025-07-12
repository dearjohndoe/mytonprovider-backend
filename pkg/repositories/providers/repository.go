package providers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mytonprovider-backend/pkg/models/db"
)

type repository struct {
	db *pgxpool.Pool
}

type Repository interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) (providers []db.ProviderDB, err error)
	GetFilteredProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) (providers []db.ProviderDB, err error)
	GetFiltersRange(ctx context.Context) (filtersRange db.FiltersRange, err error)
	UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error)
	UpdateBenchmarks(ctx context.Context, benchmarks []db.BenchmarkUpdate) (err error)
	AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error)
	UpdateProvidersIPs(ctx context.Context, ips []db.ProviderIP) (err error)
	UpdateUptime(ctx context.Context) (err error)
	UpdateRating(ctx context.Context) (err error)
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	GetAllProvidersWallets(ctx context.Context) (wallets []db.ProviderWallet, err error)
	AddStorageContracts(ctx context.Context, contracts []db.StorageContract) (err error)
	UpdateStatuses(ctx context.Context) (err error)
	UpdateContractProofsChecks(ctx context.Context, contractsProofs []db.ContractProofsCheck) (err error)
	GetStorageContracts(ctx context.Context) (contracts []db.StorageContractShort, err error)
	UpdateProvidersLT(ctx context.Context, providers []db.ProviderWalletLT) (err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error)
	AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error)

	CleanOldProvidersHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldStatusesHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldBenchmarksHistory(ctx context.Context, days int) (removed int, err error)
	CleanOldTelemetryHistory(ctx context.Context, days int) (removed int, err error)
}

func (r *repository) GetProvidersByPubkeys(ctx context.Context, pubkeys []string) (resp []db.ProviderDB, err error) {
	query := `
		SELECT 
			p.public_key,
			p.address,
			p.status,
			COALESCE(p.uptime, 0) * 100 as uptime,
			COALESCE(p.rating, 0) as rating,
			p.max_span,
			p.rate_per_mb_per_day * 1024 * 200 * 30 as price,
			p.min_span,
			p.max_bag_size_bytes,
			p.registered_at,
			t.public_key is not null as is_send_telemetry,
			t.storage_git_hash,
			t.provider_git_hash,
			t.total_provider_space,
			t.used_provider_space,
			t.cpu_name,
			t.cpu_number,
			t.cpu_is_virtual,
			t.total_ram,
			t.usage_ram,
			t.ram_usage_percent,
			t.updated_at,
			b.qd64_disk_read_speed,
			b.qd64_disk_write_speed,
			b.speedtest_download,
			b.speedtest_upload,
			b.speedtest_ping,
			b.country,
			b.isp
		FROM providers.providers p
			LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
			LEFT JOIN providers.benchmarks b ON p.public_key = b.public_key
		WHERE p.public_key = ANY($1::text[])`

	rows, err := r.db.Query(ctx, query, pubkeys)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}
	defer rows.Close()

	resp, err = scanProviderDBRows(rows)
	if err != nil {
		return
	}

	return
}

func (r *repository) GetFilteredProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) (resp []db.ProviderDB, err error) {
	var filtersStr string
	args := []any{limit, offset}

	filtersStr, args = filtersToCondition(filters, args)
	sortingStr := sortToCondition(sort)
	query := fmt.Sprintf(providersQuerySelect, filtersStr, sortingStr)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}
	defer rows.Close()

	resp, err = scanProviderDBRows(rows)
	if err != nil {
		return
	}

	return
}

func (r *repository) GetFiltersRange(ctx context.Context) (filtersRange db.FiltersRange, err error) {
	query := `
		SELECT 
			-- Provider ranges
			COALESCE(MAX(p.rating), 5.0) + 0.1 as rating_max,
			MAX(EXTRACT(DAYS FROM (NOW() - p.registered_at)))::bigint as reg_time_days_max,
			COALESCE(MIN(p.min_span), 0)::bigint as min_span_min,
			COALESCE(MAX(p.min_span), 0)::bigint as min_span_max,
			COALESCE(MIN(p.max_span), 0)::bigint as max_span_min,
			COALESCE(MAX(p.max_span), 0)::bigint as max_span_max,
			MIN(p.max_bag_size_bytes / 1024 / 1024)::bigint as max_bag_size_mb_min,
			MAX(p.max_bag_size_bytes / 1024 / 1024)::bigint as max_bag_size_mb_max,
			COALESCE(MAX(p.rate_per_mb_per_day * 1024 * 200 * 30), 0.0) + 0.1 as price_max,
			
			-- Telemetry ranges
			COALESCE(MIN(t.total_provider_space), 0)::int as total_provider_space_min,
			(COALESCE(MAX(t.total_provider_space), 1) + 1)::int as total_provider_space_max,
			(COALESCE(MAX(t.used_provider_space), 1) + 1)::int as used_provider_space_max,
			COALESCE(MAX(t.cpu_number), 1)::int as cpu_number_max,
			(COALESCE(MIN(t.total_ram), 0.1) - 0.1)::real as total_ram_min,
			(COALESCE(MAX(t.total_ram), 1.0) + 0.1)::real as total_ram_max,
			
			-- Benchmark ranges (parsing string speeds to int for comparison)
			COALESCE(MIN(providers.parse_speed_to_int(b.qd64_disk_read_speed)), 0)::bigint as benchmark_disk_read_speed_min,
			COALESCE(MAX(providers.parse_speed_to_int(b.qd64_disk_read_speed)), 0)::bigint as benchmark_disk_read_speed_max,
			COALESCE(MIN(providers.parse_speed_to_int(b.qd64_disk_write_speed)), 0)::bigint as benchmark_disk_write_speed_min,
			COALESCE(MAX(providers.parse_speed_to_int(b.qd64_disk_write_speed)), 0)::bigint as benchmark_disk_write_speed_max,
			COALESCE(MIN(b.speedtest_download), 0)::bigint as speedtest_download_speed_min,
			COALESCE(MAX(b.speedtest_download), 0)::bigint as speedtest_download_speed_max,
			COALESCE(MIN(b.speedtest_upload), 0)::bigint as speedtest_upload_speed_min,
			COALESCE(MAX(b.speedtest_upload), 0)::bigint as speedtest_upload_speed_max,
			COALESCE(MIN(b.speedtest_ping), 0)::int as speedtest_ping_min,
			COALESCE(MAX(b.speedtest_ping), 0)::int as speedtest_ping_max
		FROM providers.providers p
			LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
			LEFT JOIN providers.benchmarks b ON p.public_key = b.public_key
		WHERE p.is_initialized AND p.rating IS NOT NULL AND p.uptime IS NOT NULL
	`

	row := r.db.QueryRow(ctx, query)

	err = row.Scan(
		&filtersRange.RatingMax,
		&filtersRange.RegTimeDaysMax,
		&filtersRange.MinSpanMin,
		&filtersRange.MinSpanMax,
		&filtersRange.MaxSpanMin,
		&filtersRange.MaxSpanMax,
		&filtersRange.MaxBagSizeMbMin,
		&filtersRange.MaxBagSizeMbMax,
		&filtersRange.PriceMax,

		&filtersRange.TotalProviderSpaceMin,
		&filtersRange.TotalProviderSpaceMax,
		&filtersRange.UsedProviderSpaceMax,
		&filtersRange.CPUNumberMax,
		&filtersRange.TotalRAMMin,
		&filtersRange.TotalRAMMax,

		&filtersRange.BenchmarkDiskReadSpeedMin,
		&filtersRange.BenchmarkDiskReadSpeedMax,
		&filtersRange.BenchmarkDiskWriteSpeedMin,
		&filtersRange.BenchmarkDiskWriteSpeedMax,
		&filtersRange.SpeedtestDownloadSpeedMin,
		&filtersRange.SpeedtestDownloadSpeedMax,
		&filtersRange.SpeedtestUploadSpeedMin,
		&filtersRange.SpeedtestUploadSpeedMax,
		&filtersRange.SpeedtestPingMin,
		&filtersRange.SpeedtestPingMax,
	)

	if err != nil {
		return
	}

	return
}

func (r *repository) UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error) {
	if len(telemetry) == 0 {
		return
	}

	query := `
		WITH upd_providers AS (
			UPDATE providers.providers p
			SET
				max_bag_size_bytes = t.max_bag_size_bytes
			FROM (
				SELECT 
					t->>'public_key' as public_key, 
					(t->>'max_bag_size_bytes')::bigint as max_bag_size_bytes
				FROM jsonb_array_elements($1::jsonb) t
			) as t
			WHERE p.public_key = t.public_key
		)
		INSERT INTO providers.telemetry (
			public_key,
			storage_git_hash,
			provider_git_hash,
			cpu_name,
			pings,
			cpu_product_name,
			uname_sysname,
			uname_release,
			uname_version,
			uname_machine,
			disk_name,
			cpu_load,
			total_space,
			used_space,
			free_space,
			used_provider_space,
			total_provider_space,
			total_swap,
			usage_swap,
			swap_usage_percent,
			usage_ram,
			total_ram,
			ram_usage_percent,
			cpu_number,
			cpu_is_virtual,
			x_real_ip
		)
		SELECT 
			lower(t->>'public_key'),
			t->>'storage_git_hash',
			t->>'provider_git_hash',
			t->>'cpu_name',
			t->>'pings',
			t->>'cpu_product_name',
			t->>'uname_sysname',
			t->>'uname_release',
			t->>'uname_version',
			t->>'uname_machine',
			t->>'disk_name',
			ARRAY(
				SELECT jsonb_array_elements_text(t->'cpu_load')::float8
			),
			(t->>'total_space')::double precision,
			(t->>'used_space')::double precision,
			(t->>'free_space')::double precision,
			(t->>'used_provider_space')::float8,
			(t->>'total_provider_space')::float8,
			(t->>'total_swap')::float4,
			(t->>'usage_swap')::float4,
			(t->>'swap_usage_percent')::float4,
			(t->>'usage_ram')::float4,
			(t->>'total_ram')::float4,
			(t->>'ram_usage_percent')::float4,
			(t->>'cpu_number')::int4,
			(t->>'cpu_is_virtual')::boolean,
			t->>'x_real_ip'
		FROM jsonb_array_elements($1::jsonb) t
		ON CONFLICT (public_key) DO UPDATE SET
			storage_git_hash = EXCLUDED.storage_git_hash,
			provider_git_hash = EXCLUDED.provider_git_hash,
			cpu_name = EXCLUDED.cpu_name,
			pings = EXCLUDED.pings,
			cpu_product_name = EXCLUDED.cpu_product_name,
			uname_sysname = EXCLUDED.uname_sysname,
			uname_release = EXCLUDED.uname_release,
			uname_version = EXCLUDED.uname_version,
			uname_machine = EXCLUDED.uname_machine,
			disk_name = EXCLUDED.disk_name,
			cpu_load = EXCLUDED.cpu_load,
			total_space = EXCLUDED.total_space,
			free_space = EXCLUDED.free_space,
			used_space = EXCLUDED.used_space,
			used_provider_space = EXCLUDED.used_provider_space,
			total_provider_space = EXCLUDED.total_provider_space,
			total_swap = EXCLUDED.total_swap,
			usage_swap = EXCLUDED.usage_swap,
			swap_usage_percent = EXCLUDED.swap_usage_percent,
			usage_ram = EXCLUDED.usage_ram,
			total_ram = EXCLUDED.total_ram,
			ram_usage_percent = EXCLUDED.ram_usage_percent,
			cpu_number = EXCLUDED.cpu_number,
			cpu_is_virtual = EXCLUDED.cpu_is_virtual,
			x_real_ip = EXCLUDED.x_real_ip,
			updated_at = now()
	`

	_, err = r.db.Exec(ctx, query, telemetry)

	return
}

func (r *repository) UpdateBenchmarks(ctx context.Context, benchmarks []db.BenchmarkUpdate) (err error) {
	if len(benchmarks) == 0 {
		return
	}

	query := `
		INSERT INTO providers.benchmarks (
			public_key,
			disk,
			network,
			qd64_disk_read_speed,
			qd64_disk_write_speed,
			benchmark_timestamp,
			speedtest_download,
			speedtest_upload,
			speedtest_ping,
			country,
			isp
		)
		SELECT
			lower(b->>'public_key'),
			(b->>'disk')::jsonb,
			(b->>'network')::jsonb,
			b->>'qd64_disk_read_speed',
			b->>'qd64_disk_write_speed',
			(b->>'benchmark_timestamp')::timestamptz,
			(b->>'speedtest_download')::double precision,
			(b->>'speedtest_upload')::double precision,
			(b->>'speedtest_ping')::float8,
			b->>'country',
			b->>'isp'
		FROM jsonb_array_elements($1::jsonb) AS b
		ON CONFLICT (public_key) DO UPDATE SET
			disk = EXCLUDED.disk,
			network = EXCLUDED.network,
			qd64_disk_read_speed = EXCLUDED.qd64_disk_read_speed,
			qd64_disk_write_speed = EXCLUDED.qd64_disk_write_speed,
			benchmark_timestamp = EXCLUDED.benchmark_timestamp,
			speedtest_download = EXCLUDED.speedtest_download,
			speedtest_upload = EXCLUDED.speedtest_upload,
			speedtest_ping = EXCLUDED.speedtest_ping,
			country = EXCLUDED.country,
			isp = EXCLUDED.isp
	`

	_, err = r.db.Exec(ctx, query, benchmarks)

	return
}

func (r *repository) AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error) {
	if len(providers) == 0 {
		return
	}

	query := `
		INSERT INTO providers.statuses (public_key, is_online, check_time)
		SELECT
			lower(p->>'public_key'),
			(p->>'is_online')::boolean,
			NOW()
		FROM jsonb_array_elements($1::jsonb) AS p
		ON CONFLICT (public_key) DO UPDATE SET
			is_online = EXCLUDED.is_online,
			check_time = NOW()
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) UpdateProvidersIPs(ctx context.Context, ips []db.ProviderIP) (err error) {
	if len(ips) == 0 {
		return
	}

	query := `
		UPDATE providers.providers p
		SET
			ip = COALESCE(p.ip, i.ip),
			port = i.port
		FROM (

			SELECT
				lower(i->>'public_key') AS public_key,
				i->>'ip' AS ip,
				(i->>'port')::int AS port
		FROM jsonb_array_elements($1::jsonb) AS i
		) AS i
		WHERE p.public_key = i.public_key
	`
	_, err = r.db.Exec(ctx, query, ips)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return fmt.Errorf("failed to update providers IPs: %w", err)
	}

	return nil
}

func (r *repository) UpdateUptime(ctx context.Context) (err error) {
	query := `
		WITH provider_uptime AS (
			SELECT
				public_key,
				count(*) AS total,
				count(*) filter (where is_online) AS online
			FROM providers.statuses_history
			GROUP BY public_key
		)
		UPDATE providers.providers p
		SET uptime = COALESCE((SELECT pu.online::float8 / pu.total), 0)
		FROM provider_uptime pu
		WHERE p.public_key = pu.public_key
	`

	_, err = r.db.Exec(ctx, query)

	return
}

func (r *repository) UpdateRating(ctx context.Context) (err error) {
	query := `
		WITH params AS (
			SELECT 
				p.public_key,
				p.registered_at,
				p.uptime,
				p.max_span,
				p.min_span,
				0 as max_bag_size_bytes, -- p.max_bag_size_bytes 
				p.rate_per_mb_per_day,
				t.total_provider_space,
				t.cpu_number,
				t.total_ram,
				b.qd64_disk_write_speed,
				b.qd64_disk_read_speed,
				b.speedtest_download,
				b.speedtest_upload,
				b.speedtest_ping
			FROM providers.providers p
				LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
				LEFT JOIN providers.benchmarks b ON p.public_key = b.public_key
			WHERE p.is_initialized
		)
		UPDATE providers.providers p
		SET rating = (
			(
				0.0001 * (EXTRACT(EPOCH FROM pr.registered_at) * COALESCE(pr.uptime, 0)) +
				0.00002 * (COALESCE(pr.max_span, 0) - COALESCE(pr.min_span, 0)) +
				0.00000000008 * COALESCE(pr.max_bag_size_bytes, 0) +
				0.000000004 * COALESCE(pr.total_provider_space, 0) +
				1.9 * LEAST(COALESCE(pr.cpu_number, 0), 128) +
				0.0000006 * COALESCE(pr.total_ram, 0) +
				0.00008 * COALESCE(providers.parse_speed_to_int(pr.qd64_disk_write_speed), 0) +
				0.00008 * COALESCE(providers.parse_speed_to_int(pr.qd64_disk_read_speed), 0) +
				0.00001 * COALESCE(pr.speedtest_download, 0) +
				0.00004 * COALESCE(pr.speedtest_upload, 0) +
				CASE WHEN COALESCE(pr.speedtest_ping, 0) > 0 THEN 400 / pr.speedtest_ping ELSE 1 END
			)
			/ GREATEST(LOG(COALESCE(NULLIF(pr.rate_per_mb_per_day / 100, 0), 1)), 1)
		) / 10000.0
		FROM params pr
		WHERE p.public_key = pr.public_key
    `
	_, err = r.db.Exec(ctx, query)

	return
}

func (r *repository) GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error) {
	query := `
		SELECT public_key
		FROM providers.providers`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}
	defer rows.Close()

	for rows.Next() {
		var pubkey string
		if rErr := rows.Scan(&pubkey); rErr != nil {
			err = rErr
			return
		}
		pubkeys = append(pubkeys, pubkey)
	}

	err = rows.Err()
	if err != nil {
		return
	}

	return
}

func (r *repository) GetAllProvidersWallets(ctx context.Context) (wallets []db.ProviderWallet, err error) {
	query := `
		SELECT p.public_key, p.address, p.last_tx_lt
		FROM providers.providers p
		`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}
	defer rows.Close()

	for rows.Next() {
		var pubkey string
		var address string
		var lt uint64
		if rErr := rows.Scan(&pubkey, &address, &lt); rErr != nil {
			err = rErr
			return
		}
		wallets = append(wallets, db.ProviderWallet{
			PubKey:  pubkey,
			Address: address,
			LT:      lt,
		})
	}

	err = rows.Err()

	return
}

func (r *repository) AddStorageContracts(ctx context.Context, contracts []db.StorageContract) (err error) {
	if len(contracts) == 0 {
		return
	}

	query := `
		WITH cte AS (
			SELECT 
				c->>'address' AS address,
        		ARRAY(SELECT jsonb_object_keys(c->'providers_addresses'))::text[] AS providers_addresses,
				c->>'bag_id' AS bag_id,
				c->>'owner_address' AS owner_address,
				(c->>'size')::bigint AS size,
				(c->>'chunk_size')::bigint AS chunk_size,
				(c->>'last_tx_lt')::bigint AS last_tx_lt
			FROM jsonb_array_elements($1::jsonb) AS c
		)
		INSERT INTO providers.storage_contracts (
			address,
			provider_address,
			bag_id,
			owner_address,
			size,
			chunk_size,
			last_tx_lt
		)
		SELECT
			address,
			unnest(providers_addresses),
			bag_id,
			owner_address,
			size,
			chunk_size,
			last_tx_lt
		FROM cte
		ON CONFLICT (address, provider_address) DO UPDATE SET
			last_tx_lt = EXCLUDED.last_tx_lt
	`

	_, err = r.db.Exec(ctx, query, contracts)

	return
}

func (r *repository) UpdateStatuses(ctx context.Context) (err error) {
	query := `
		UPDATE providers.providers p 
		SET status = selected_reasons.reason
		FROM (
			WITH collect_statuses AS (
				SELECT 
					p.address, 
					sc.reason, 
					count(*) as cnt,
					ROW_NUMBER() OVER (
						PARTITION BY p.address 
						ORDER BY count(*) DESC, 
								CASE WHEN sc.reason IS NULL THEN 1 ELSE 0 END ASC
					) as rn
				FROM providers.providers p
					LEFT JOIN providers.storage_contracts sc ON p.address = sc.provider_address
				-- get only the most recent reason for each address
				WHERE sc.reason IS NOT NULL AND sc.reason_timestamp > NOW() - INTERVAL '24 hours'
				GROUP BY p.address, sc.reason
			)
			SELECT address, reason
			FROM collect_statuses
			WHERE rn = 1
		) selected_reasons
		WHERE p.address = selected_reasons.address;
	`

	_, err = r.db.Exec(ctx, query)

	return
}

func (r *repository) UpdateContractProofsChecks(ctx context.Context, contractsProofs []db.ContractProofsCheck) (err error) {
	query := `
		WITH cte AS (
			SELECT
				c->>'address' AS address,
				c->>'provider_address' AS provider_address,
				(c->>'reason')::integer AS reason,
				(c->>'timestamp')::timestamptz AS timestamp
			FROM jsonb_array_elements($1::jsonb) AS c
		)
		UPDATE providers.storage_contracts sc
		SET
			reason = c.reason,
			reason_timestamp = c.timestamp
		FROM cte c
		WHERE sc.address = c.address AND c.provider_address = sc.provider_address
	`

	_, err = r.db.Exec(ctx, query, contractsProofs)

	return
}

func (r *repository) GetStorageContracts(ctx context.Context) (contracts []db.StorageContractShort, err error) {
	query := `
		SELECT 
			p.public_key as provider_public_key,
			sc.provider_address,
			sc.address,
			sc.size
		FROM providers.storage_contracts sc
			JOIN providers.providers p ON p.address = sc.provider_address
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}

	defer rows.Close()
	for rows.Next() {
		var contract db.StorageContractShort
		if rErr := rows.Scan(&contract.ProviderPublicKey, &contract.ProviderAddress, &contract.Address, &contract.Size); rErr != nil {
			err = rErr
			return
		}
		contracts = append(contracts, contract)
	}

	err = rows.Err()
	if err != nil {
		return
	}

	return
}

func (r *repository) UpdateProvidersLT(ctx context.Context, providers []db.ProviderWalletLT) (err error) {
	if len(providers) == 0 {
		return
	}

	query := `
		UPDATE providers.providers p
		SET
			last_tx_lt = c.last_tx_lt
		FROM (
			SELECT 
				lower(p->>'public_key') AS public_key,
				(p->>'last_tx_lt')::bigint AS last_tx_lt
			FROM jsonb_array_elements($1::jsonb) AS p
		) AS c
		WHERE p.public_key = c.public_key
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error) {
	if len(providers) == 0 {
		return
	}

	query := `
		UPDATE providers.providers
		SET
			rate_per_mb_per_day = p.rate_per_mb_per_day,
			min_bounty = p.min_bounty,
			min_span = p.min_span,
			max_span = p.max_span,
			is_initialized = true,
			updated_at = NOW()
		FROM (
			SELECT
				p->>'public_key' AS public_key,
				(p->>'rate_per_mb_per_day')::bigint AS rate_per_mb_per_day,
				(p->>'min_bounty')::bigint AS min_bounty,
				(p->>'min_span')::int AS min_span,
				(p->>'max_span')::int AS max_span
			FROM jsonb_array_elements($1::jsonb) AS p
		) AS p
		WHERE providers.providers.public_key = p.public_key
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error) {
	if len(providers) == 0 {
		return
	}

	query := `
		INSERT INTO providers.providers (public_key, address, registered_at, is_initialized)
		SELECT 
			lower(p->>'public_key'),
			p->>'address',
			(p->>'registered_at')::timestamptz,
			false
		FROM jsonb_array_elements($1::jsonb) AS p
		ON CONFLICT DO NOTHING
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) CleanOldProvidersHistory(ctx context.Context, days int) (removed int, err error) {
	query := `
		DELETE FROM providers.providers_history
		WHERE archived_at < NOW() - INTERVAL '1 day' * $1
	`
	resp, err := r.db.Exec(ctx, query, days)
	if err != nil {
		err = fmt.Errorf("failed to clean old providers history: %w", err)
		return
	}

	removed = int(resp.RowsAffected())

	return
}

func (r *repository) CleanOldStatusesHistory(ctx context.Context, days int) (removed int, err error) {
	query := `
		DELETE FROM providers.statuses_history
		WHERE check_time < NOW() - INTERVAL '1 day' * $1
	`
	resp, err := r.db.Exec(ctx, query, days)
	if err != nil {
		err = fmt.Errorf("failed to clean old statuses history: %w", err)
		return
	}

	removed = int(resp.RowsAffected())

	return
}

func (r *repository) CleanOldBenchmarksHistory(ctx context.Context, days int) (removed int, err error) {
	query := `
		DELETE FROM providers.benchmarks_history
		WHERE archived_at < NOW() - INTERVAL '1 day' * $1
	`
	resp, err := r.db.Exec(ctx, query, days)
	if err != nil {
		err = fmt.Errorf("failed to clean old benchmarks history: %w", err)
		return
	}

	removed = int(resp.RowsAffected())

	return
}

func (r *repository) CleanOldTelemetryHistory(ctx context.Context, days int) (removed int, err error) {
	query := `
		DELETE FROM providers.telemetry_history
		WHERE archived_at < NOW() - INTERVAL '1 day' * $1
	`
	resp, err := r.db.Exec(ctx, query, days)
	if err != nil {
		err = fmt.Errorf("failed to clean old telemetry history: %w", err)
		return
	}

	removed = int(resp.RowsAffected())

	return
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
	}
}
