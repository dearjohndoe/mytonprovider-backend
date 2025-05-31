package providers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"mytonprovider-backend/pkg/models/db"
)

type repository struct {
	db *pgxpool.Pool
}

type Repository interface {
	GetProvidersByPubkeys(ctx context.Context, pubkeys []string) ([]db.Provider, error)
	GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) ([]db.Provider, error)
	UpdateTelemetry(ctx context.Context, telemetry []db.Telemetry) (err error)
	GetAllProvidersPubkeys(ctx context.Context) (pubkeys []string, err error)
	UpdateProviders(ctx context.Context, providers []db.ProviderInfo) (err error)
	DisableProviders(ctx context.Context, providers []string) (err error)
	AddProviders(ctx context.Context, providers []db.ProviderInit) (err error)
}

func (r *repository) GetProvidersByPubkeys(ctx context.Context, pubkeys []string) (resp []db.Provider, err error) {
	return
}

func (r *repository) GetProviders(ctx context.Context, filters db.ProviderFilters, sort db.ProviderSort, limit, offset int) (resp []db.Provider, err error) {
	return
}

func (r *repository) UpdateTelemetry(ctx context.Context, telemetry []db.Telemetry) (err error) {
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

func (r *repository) UpdateProviders(ctx context.Context, providers []db.ProviderInfo) (err error) {
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
			rating = p.rating,
			is_initialized = true,
			updated_at = NOW()
		FROM (
			SELECT
				p->>'public_key' AS public_key,
				(p->>'rate_per_mb_per_day')::bigint AS rate_per_mb_per_day,
				(p->>'min_bounty')::bigint AS min_bounty,
				(p->>'min_span')::int AS min_span,
				(p->>'max_span')::int AS max_span,
				(p->>'rating')::float8 AS rating
			FROM jsonb_array_elements($1::jsonb) AS p
		) AS p
		WHERE providers.providers.public_key = p.public_key
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) DisableProviders(ctx context.Context, providers []string) (err error) {
	if len(providers) == 0 {
		return nil
	}

	query := `
		UPDATE providers.providers
		SET is_available = false, updated_at = NOW()
		WHERE public_key = ANY($1::text[])
	`

	_, err = r.db.Exec(ctx, query, providers)

	return
}

func (r *repository) AddProviders(ctx context.Context, providers []db.ProviderInit) (err error) {
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

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
	}
}
