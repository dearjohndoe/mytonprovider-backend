package providers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mytonprovider-backend/pkg/models/db"
)

type repository struct {
	db *pgxpool.Pool
}

type Repository interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) ([]db.ProviderDB, error)
	GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) ([]db.ProviderDB, error)
	UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error)
	AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error)
	UpdateUptime(ctx context.Context) (err error)
	UpdateRating(ctx context.Context) (err error)
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error)
	AddProviders(ctx context.Context, providers []db.ProviderCreate) (err error)
}

func (r *repository) GetProvidersByPubkeys(ctx context.Context, pubkeys []string) (resp []db.ProviderDB, err error) {
	query := `
		SELECT 
			p.public_key,
			p.uptime,
			p.rating,
			p.max_span,
			p.rate_per_mb_per_day,
			p.min_span,
			0,                  -- p.max_bag_size_bytes ???
			p.registered_at,
			coalesce(p.is_send_telemetry, false) as is_send_telemetry,
			t.storage_git_hash,
			t.provider_git_hash,
			t.total_provider_space,
			t.total_provider_space - t.used_provider_space as free_provider_space,
			t.cpu_name,
			t.cpu_number,
			t.cpu_is_virtual,
			t.total_ram,
			t.free_ram,
			t.benchmark_disk_read_speed,
			t.benchmark_disk_write_speed,
			t.benchmark_rocks_ops,
			t.speedtest_download_speed,
			t.speedtest_upload_speed,
			t.speedtest_ping,
			t.country,
			t.isp
		FROM providers.providers p
			LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
		WHERE p.public_key = ANY($1::text[])`

	rows, err := r.db.Query(ctx, query, pubkeys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp, err = scanProviderDBRows(rows)
	if err != nil {
		return
	}

	return
}

func (r *repository) GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) (resp []db.ProviderDB, err error) {
	query := `
		SELECT 
			p.public_key,
			p.uptime,
			p.rating,
			p.max_span,
			p.rate_per_mb_per_day,
			p.min_span,
			0,                  -- p.max_bag_size_bytes ???
			p.registered_at,
			coalesce(p.is_send_telemetry, false) as is_send_telemetry,
			t.storage_git_hash,
			t.provider_git_hash,
			t.total_provider_space,
			t.total_provider_space - t.used_provider_space as free_provider_space,
			t.cpu_name,
			t.cpu_number,
			t.cpu_is_virtual,
			t.total_ram,
			t.free_ram,
			t.benchmark_disk_read_speed,
			t.benchmark_disk_write_speed,
			t.benchmark_rocks_ops,
			t.speedtest_download_speed,
			t.speedtest_upload_speed,
			t.speedtest_ping,
			t.country,
			t.isp
		FROM providers.providers p
		    JOIN providers.statuses s ON p.public_key = s.public_key
			LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
		WHERE p.is_initialized 
			AND s.is_online 
			AND s.check_time > NOW() - INTERVAL '1 hour'
		LIMIT $1
		OFFSET $2;
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return
	}
	defer rows.Close()

	resp, err = scanProviderDBRows(rows)
	if err != nil {
		return
	}

	return
}

func (r *repository) UpdateTelemetry(ctx context.Context, telemetry []db.TelemetryUpdate) (err error) {
	if len(telemetry) == 0 {
		return nil
	}

	query := `
		WITH upd_providers AS (
			UPDATE providers.providers
			SET
				is_send_telemetry = true
			WHERE public_key = ANY(SELECT t->>'public_key' FROM jsonb_array_elements($1::jsonb) t)
		)
		INSERT INTO providers.telemetry (
			public_key,
			storage_git_hash,
			provider_git_hash,
			cpu_name,
			country,
			isp,
			pings,
			benchmarks,
			cpu_product_name,
			uname_sysname,
			uname_release,
			uname_version,
			uname_machine,
			disk_name,
			cpu_load,
			total_space,
			free_space,
			used_space,
			benchmark_disk_read_speed,
			benchmark_disk_write_speed,
			benchmark_rocks_ops,
			speedtest_download_speed,
			speedtest_upload_speed,
			speedtest_ping,
			used_provider_space,
			total_provider_space,
			total_swap,
			free_swap,
			swap_usage_percent,
			free_ram,
			total_ram,
			ram_usage_percent,
			cpu_number,
			cpu_is_virtual
		)
		SELECT 
			t->>'public_key',
			t->>'storage_git_hash',
			t->>'provider_git_hash',
			t->>'cpu_name',
			t->>'country',
			t->>'isp',
			t->>'pings',
			t->>'benchmarks',
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
			(t->>'benchmark_disk_read_speed')::bigint,
			(t->>'benchmark_disk_write_speed')::bigint,
			(t->>'benchmark_rocks_ops')::bigint,
			(t->>'speedtest_download_speed')::float8,
			(t->>'speedtest_upload_speed')::float8,
			(t->>'speedtest_ping')::float8,
			(t->>'used_provider_space')::float8,
			(t->>'total_provider_space')::float8,
			(t->>'total_swap')::float4,
			(t->>'free_swap')::float4,
			(t->>'swap_usage_percent')::float4,
			(t->>'free_ram')::float4,
			(t->>'total_ram')::float4,
			(t->>'ram_usage_percent')::float4,
			(t->>'cpu_number')::int4,
			(t->>'cpu_is_virtual')::boolean
		FROM jsonb_array_elements($1::jsonb) t
		ON CONFLICT (public_key) DO UPDATE SET
			storage_git_hash = EXCLUDED.storage_git_hash,
			provider_git_hash = EXCLUDED.provider_git_hash,
			cpu_name = EXCLUDED.cpu_name,
			country = EXCLUDED.country,
			isp = EXCLUDED.isp,
			pings = EXCLUDED.pings,
			benchmarks = EXCLUDED.benchmarks,
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
			benchmark_disk_read_speed = EXCLUDED.benchmark_disk_read_speed,
			benchmark_disk_write_speed = EXCLUDED.benchmark_disk_write_speed,
			benchmark_rocks_ops = EXCLUDED.benchmark_rocks_ops,
			speedtest_download_speed = EXCLUDED.speedtest_download_speed,
			speedtest_upload_speed = EXCLUDED.speedtest_upload_speed,
			speedtest_ping = EXCLUDED.speedtest_ping,
			used_provider_space = EXCLUDED.used_provider_space,
			total_provider_space = EXCLUDED.total_provider_space,
			total_swap = EXCLUDED.total_swap,
			free_swap = EXCLUDED.free_swap,
			swap_usage_percent = EXCLUDED.swap_usage_percent,
			free_ram = EXCLUDED.free_ram,
			total_ram = EXCLUDED.total_ram,
			ram_usage_percent = EXCLUDED.ram_usage_percent,
			cpu_number = EXCLUDED.cpu_number,
			cpu_is_virtual = EXCLUDED.cpu_is_virtual
	`

	_, err = r.db.Exec(ctx, query, telemetry)
	return err
}

func (r *repository) AddStatuses(ctx context.Context, providers []db.ProviderStatusUpdate) (err error) {
	if len(providers) == 0 {
		return nil
	}

	query := `
		INSERT INTO providers.statuses (public_key, is_online, check_time)
		SELECT
			p->>'public_key',
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
		RETURNING p.public_key, p.uptime
	`

	_, err = r.db.Query(ctx, query)
	if err != nil {
		return
	}

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
				t.benchmark_disk_write_speed,
				t.benchmark_disk_read_speed,
				t.benchmark_rocks_ops,
				t.speedtest_download_speed,
				t.speedtest_upload_speed,
				t.speedtest_ping
			FROM providers.providers p
				LEFT JOIN providers.telemetry t ON p.public_key = t.public_key
			WHERE p.is_initialized
		)
		UPDATE providers.providers p
		SET rating = (
			(
				0.01 * (EXTRACT(EPOCH FROM pr.registered_at) * COALESCE(pr.uptime, 0)) +
				0.00002 * (COALESCE(pr.max_span, 0) - COALESCE(pr.min_span, 0)) +
				0.00000000008 * COALESCE(pr.max_bag_size_bytes, 0) +
				0.000000004 * COALESCE(pr.total_provider_space, 0) +
				1.9 * COALESCE(pr.cpu_number, 0) +
				0.0000006 * COALESCE(pr.total_ram, 0) +
				0.00008 * COALESCE(pr.benchmark_disk_write_speed, 0) +
				0.00008 * COALESCE(pr.benchmark_disk_read_speed, 0) +
				0.0002 * COALESCE(pr.benchmark_rocks_ops, 0) +
				0.00001 * COALESCE(pr.speedtest_download_speed, 0) +
				0.00004 * COALESCE(pr.speedtest_upload_speed, 0) +
				CASE WHEN COALESCE(pr.speedtest_ping, 0) > 0 THEN 400 / pr.speedtest_ping ELSE 0 END
			)
			/ NULLIF(COALESCE(pr.rate_per_mb_per_day, 1), 0)
		) / 2000.0
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
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pubkey string
		if err := rows.Scan(&pubkey); err != nil {
			return nil, err
		}
		pubkeys = append(pubkeys, pubkey)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return
}

func (r *repository) UpdateProviders(ctx context.Context, providers []db.ProviderUpdate) (err error) {
	if len(providers) == 0 {
		return nil
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
		return nil
	}

	query := `
		INSERT INTO providers.providers (public_key, address, registered_at, is_initialized)
		SELECT 
			p->>'public_key',
			p->>'address',
			(p->>'registered_at')::timestamptz,
			false
		FROM jsonb_array_elements($1::jsonb) AS p
		ON CONFLICT DO NOTHING
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func scanProviderDBRows(rows pgx.Rows) (providers []db.ProviderDB, err error) {
	for rows.Next() {
		var regTime time.Time
		var provider db.ProviderDB
		if err := rows.Scan(
			&provider.PubKey,
			&provider.UpTime,
			&provider.Rating,
			&provider.MaxSpan,
			&provider.Price,
			&provider.MinSpan,
			&provider.MaxBagSizeBytes, // TODO: where to get this value?
			&regTime,
			&provider.IsSendTelemetry,
			&provider.Telemetry.StorageGitHash,
			&provider.Telemetry.ProviderGitHash,
			&provider.Telemetry.TotalProviderSpace,
			&provider.Telemetry.FreeProviderSpace,
			&provider.Telemetry.CPUName,
			&provider.Telemetry.CPUNumber,
			&provider.Telemetry.CPUIsVirtual,
			&provider.Telemetry.TotalRAM,
			&provider.Telemetry.FreeRAM,
			&provider.Telemetry.BenchmarkDiskReadSpeed,
			&provider.Telemetry.BenchmarkDiskWriteSpeed,
			&provider.Telemetry.BenchmarkRocksOps,
			&provider.Telemetry.SpeedtestDownloadSpeed,
			&provider.Telemetry.SpeedtestUploadSpeed,
			&provider.Telemetry.SpeedtestPing,
			&provider.Telemetry.Country,
			&provider.Telemetry.ISP); err != nil {
			return nil, err
		}

		provider.RegTime = uint64(regTime.Unix())
		providers = append(providers, provider)
	}

	if rErr := rows.Err(); rErr != nil {
		err = rErr
		return
	}

	return
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
	}
}
