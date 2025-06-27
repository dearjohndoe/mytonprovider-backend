CREATE SCHEMA providers
    AUTHORIZATION pguser;

-- providers.benchmarks
CREATE TABLE IF NOT EXISTS providers.benchmarks
(
    public_key text COLLATE pg_catalog."default" NOT NULL,
    disk jsonb,
    network jsonb,
    qd64_disk_read_speed text COLLATE pg_catalog."default",
    qd64_disk_write_speed text COLLATE pg_catalog."default",
    benchmark_timestamp timestamp with time zone,
    speedtest_download double precision,
    speedtest_upload double precision,
    speedtest_ping double precision,
    country character varying(128) COLLATE pg_catalog."default",
    isp character varying(128) COLLATE pg_catalog."default",
    CONSTRAINT benchmarks_pkey PRIMARY KEY (public_key)
)

TABLESPACE pg_default;

ALTER TABLE providers.benchmarks
    OWNER to pguser;

COMMENT ON TABLE providers.benchmarks IS 'To store benchmarks data for providers';

-- providers.benchmarks_history
CREATE TABLE IF NOT EXISTS providers.benchmarks_history
(
    id integer NOT NULL DEFAULT nextval('providers.benchmarks_history_id_seq'::regclass),
    archived_at timestamp with time zone NOT NULL DEFAULT now(),
    public_key text COLLATE pg_catalog."default" NOT NULL,
    disk jsonb,
    network jsonb,
    qd64_disk_read_speed text COLLATE pg_catalog."default",
    qd64_disk_write_speed text COLLATE pg_catalog."default",
    benchmark_timestamp timestamp with time zone,
    speedtest_download double precision,
    speedtest_upload double precision,
    speedtest_ping double precision,
    country character varying(128) COLLATE pg_catalog."default",
    isp character varying(128) COLLATE pg_catalog."default",
    CONSTRAINT benchmarks_history_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE providers.benchmarks_history
    OWNER to pguser;

CREATE SCHEMA system
    AUTHORIZATION pguser;

-- providers.providers
CREATE TABLE IF NOT EXISTS providers.providers
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    registered_at timestamp with time zone NOT NULL,
    rating double precision,
    updated_at timestamp with time zone,
    min_bounty bigint,
    rate_per_mb_per_day bigint,
    min_span integer,
    max_span integer,
    is_send_telemetry boolean,
    is_initialized boolean NOT NULL DEFAULT false,
    uptime double precision NOT NULL DEFAULT 0.0,
    max_bag_size_bytes bigint NOT NULL DEFAULT 0,
    CONSTRAINT providers_pkey PRIMARY KEY (public_key),
    CONSTRAINT providers_address_key UNIQUE (address)
)

TABLESPACE pg_default;

ALTER TABLE providers.providers
    OWNER to pguser;

-- providers.providers_history
CREATE TABLE IF NOT EXISTS providers.providers_history
(
    id integer NOT NULL DEFAULT nextval('providers.providers_history_id_seq'::regclass),
    archived_at timestamp with time zone NOT NULL DEFAULT now(),
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    registered_at timestamp with time zone NOT NULL,
    rating double precision,
    updated_at timestamp with time zone,
    min_bounty bigint,
    rate_per_mb_per_day bigint,
    min_span integer,
    max_span integer,
    is_send_telemetry boolean,
    is_initialized boolean NOT NULL,
    uptime double precision NOT NULL DEFAULT 0.0,
    CONSTRAINT providers_history_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE providers.providers_history
    OWNER to pguser;

-- providers.statuses
CREATE TABLE IF NOT EXISTS providers.statuses
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    check_time timestamp with time zone NOT NULL,
    is_online boolean NOT NULL,
    CONSTRAINT statuses_pkey PRIMARY KEY (public_key)
)

TABLESPACE pg_default;

ALTER TABLE providers.statuses
    OWNER to pguser;

-- providers.statuses_history
CREATE TABLE IF NOT EXISTS providers.statuses_history
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    check_time timestamp with time zone NOT NULL,
    is_online boolean NOT NULL
)

TABLESPACE pg_default;

ALTER TABLE providers.statuses_history
    OWNER to pguser;

-- providers.telemetry
CREATE TABLE IF NOT EXISTS providers.telemetry
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    storage_git_hash character varying(40) COLLATE pg_catalog."default" NOT NULL,
    provider_git_hash character varying(40) COLLATE pg_catalog."default" NOT NULL,
    cpu_name character varying(255) COLLATE pg_catalog."default" NOT NULL,
    pings text COLLATE pg_catalog."default",
    cpu_product_name character varying(255) COLLATE pg_catalog."default",
    uname_sysname character varying(64) COLLATE pg_catalog."default",
    uname_release character varying(64) COLLATE pg_catalog."default",
    uname_version character varying(128) COLLATE pg_catalog."default",
    uname_machine character varying(64) COLLATE pg_catalog."default",
    disk_name character varying(255) COLLATE pg_catalog."default",
    cpu_load double precision[][],
    total_space double precision NOT NULL,
    free_space double precision NOT NULL,
    used_space double precision NOT NULL,
    used_provider_space double precision,
    total_provider_space double precision,
    total_swap real,
    usage_swap real,
    swap_usage_percent real,
    usage_ram real,
    total_ram real,
    ram_usage_percent real,
    cpu_number integer NOT NULL,
    cpu_is_virtual boolean NOT NULL,
    benchmarks text COLLATE pg_catalog."default",
    benchmark_disk_read_speed bigint NOT NULL DEFAULT 0,
    benchmark_disk_write_speed bigint NOT NULL DEFAULT 0,
    benchmark_rocks_ops bigint NOT NULL DEFAULT 0,
    speedtest_download_speed double precision NOT NULL DEFAULT 0.0,
    speedtest_upload_speed double precision NOT NULL DEFAULT 0.0,
    speedtest_ping double precision NOT NULL DEFAULT 0.0,
    country character varying(128) COLLATE pg_catalog."default" NOT NULL DEFAULT ''::character varying,
    isp character varying(128) COLLATE pg_catalog."default" NOT NULL DEFAULT ''::character varying,
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT telemetry_pkey PRIMARY KEY (public_key)
)

TABLESPACE pg_default;

ALTER TABLE providers.telemetry
    OWNER to pguser;

-- providers.telemetry_history
CREATE TABLE IF NOT EXISTS providers.telemetry_history
(
    id integer NOT NULL DEFAULT nextval('providers.telemetry_history_id_seq'::regclass),
    archived_at timestamp with time zone NOT NULL DEFAULT now(),
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    storage_git_hash character varying(40) COLLATE pg_catalog."default" NOT NULL,
    provider_git_hash character varying(40) COLLATE pg_catalog."default" NOT NULL,
    cpu_name character varying(255) COLLATE pg_catalog."default" NOT NULL,
    pings text COLLATE pg_catalog."default",
    cpu_product_name character varying(255) COLLATE pg_catalog."default",
    uname_sysname character varying(64) COLLATE pg_catalog."default",
    uname_release character varying(64) COLLATE pg_catalog."default",
    uname_version character varying(128) COLLATE pg_catalog."default",
    uname_machine character varying(64) COLLATE pg_catalog."default",
    disk_name character varying(255) COLLATE pg_catalog."default",
    cpu_load double precision[][],
    total_space double precision NOT NULL,
    free_space double precision NOT NULL,
    used_space double precision NOT NULL,
    used_provider_space double precision,
    total_provider_space double precision,
    total_swap real,
    usage_swap real,
    swap_usage_percent real,
    usage_ram real,
    total_ram real,
    ram_usage_percent real,
    cpu_number integer NOT NULL,
    cpu_is_virtual boolean NOT NULL,
    CONSTRAINT telemetry_history_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE providers.telemetry_history
    OWNER to pguser;





-- functions
-- providers.parse_speed_to_int(...)
CREATE OR REPLACE FUNCTION providers.parse_speed_to_int(
	speed_text text)
    RETURNS integer
    LANGUAGE plpgsql
    COST 100
    IMMUTABLE PARALLEL UNSAFE
AS $BODY$
DECLARE
    value numeric;
    unit text;
    multiplier integer := 1;
BEGIN
    -- Extract numeric value and unit
    value := regexp_replace(speed_text, '[^0-9\.]', '', 'g')::numeric;
    unit := regexp_replace(speed_text, '[0-9\.\/]', '', 'g');

    -- Determine multiplier based on unit
    IF unit = 'KiBps' OR unit = 'KiB/s' OR unit = 'KiB' THEN
        multiplier := 1024;
    ELSIF unit = 'MiBps' OR unit = 'MiB/s' OR unit = 'MiB' THEN
        multiplier := 1024 * 1024;
    ELSIF unit = 'GiBps' OR unit = 'GiB/s' OR unit = 'GiB' THEN
        multiplier := 1024 * 1024 * 1024;
    ELSIF unit = 'KBps' OR unit = 'KB/s' OR unit = 'KB' THEN
        multiplier := 1000;
    ELSIF unit = 'MBps' OR unit = 'MB/s' OR unit = 'MB' THEN
        multiplier := 1000 * 1000;
    ELSIF unit = 'GBps' OR unit = 'GB/s' OR unit = 'GB' THEN
        multiplier := 1000 * 1000 * 1000;
    END IF;

    RETURN (value * multiplier)::integer;
END;
$BODY$;

ALTER FUNCTION providers.parse_speed_to_int(speed_text text)
    OWNER TO pguser;

-- triggers
-- providers.archive_benchmarks()
CREATE FUNCTION providers.archive_benchmarks()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    INSERT INTO providers.benchmarks_history (
        public_key, disk, benchmark_timestamp, server, client, share, timestamp, bytes_received, bytes_sent, download, upload, ping
    ) VALUES (
        OLD.public_key, OLD.disk, OLD.benchmark_timestamp, OLD.server, OLD.client, OLD.share, OLD.timestamp, OLD.bytes_received, OLD.bytes_sent, OLD.download, OLD.upload, OLD.ping
    );
    RETURN OLD;
END;
$BODY$;

ALTER FUNCTION providers.archive_benchmarks()
    OWNER TO pguser;

-- providers.archive_benchmarks_after_update()
CREATE FUNCTION providers.archive_benchmarks_after_update()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    INSERT INTO providers.benchmarks_history (
        public_key, disk, network, qd64_disk_read_speed, qd64_disk_write_speed, 
        benchmark_timestamp, speedtest_download, speedtest_upload, speedtest_ping, country, isp
    ) VALUES (
        OLD.public_key, OLD.disk, OLD.network, OLD.qd64_disk_read_speed, OLD.qd64_disk_write_speed,
        OLD.benchmark_timestamp, OLD.speedtest_download, OLD.speedtest_upload, OLD.speedtest_ping, OLD.country, OLD.isp
    );
    RETURN NEW;
END;
$BODY$;

ALTER FUNCTION providers.archive_benchmarks_after_update()
    OWNER TO pguser;

-- providers.archive_telemetry()
CREATE FUNCTION providers.archive_telemetry()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    INSERT INTO providers.telemetry_history (
        public_key, storage_git_hash, provider_git_hash, cpu_name, pings, cpu_product_name,
        uname_sysname, uname_release, uname_version, uname_machine, disk_name, cpu_load, total_space, free_space, used_space,
        used_provider_space, total_provider_space, total_swap, usage_swap, swap_usage_percent, usage_ram, total_ram,
        ram_usage_percent, cpu_number, cpu_is_virtual
    ) VALUES (
        OLD.public_key, OLD.storage_git_hash, OLD.provider_git_hash, OLD.cpu_name, OLD.pings, OLD.cpu_product_name,
        OLD.uname_sysname, OLD.uname_release, OLD.uname_version, OLD.uname_machine, OLD.disk_name, OLD.cpu_load, OLD.total_space, OLD.free_space, OLD.used_space,
        OLD.used_provider_space, OLD.total_provider_space, OLD.total_swap, OLD.usage_swap, OLD.swap_usage_percent, OLD.usage_ram, OLD.total_ram,
        OLD.ram_usage_percent, OLD.cpu_number, OLD.cpu_is_virtual
    );
    RETURN NEW;
END;
$BODY$;

ALTER FUNCTION providers.archive_telemetry()
    OWNER TO pguser;

-- providers.log_provider_update()
CREATE FUNCTION providers.log_provider_update()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
begin
    if 
        old.public_key is distinct from new.public_key or
        old.address is distinct from new.address or
        old.registered_at is distinct from new.registered_at or
        old.uptime is distinct from new.uptime or
        old.rating is distinct from new.rating or
        old.updated_at is distinct from new.updated_at or
        old.min_bounty is distinct from new.min_bounty or
        old.rate_per_mb_per_day is distinct from new.rate_per_mb_per_day or
        old.min_span is distinct from new.min_span or
        old.max_span is distinct from new.max_span or
        old.is_initialized is distinct from new.is_initialized or
    then
        insert into providers.providers_history (
            public_key,
            address,
            registered_at,
            uptime,
            rating,
            updated_at,
            min_bounty,
            rate_per_mb_per_day,
            min_span,
            max_span,
            is_send_telemetry,
            is_initialized
        ) values (
            old.public_key,
            old.address,
            old.registered_at,
            old.uptime,
            old.rating,
            old.updated_at,
            old.min_bounty,
            old.rate_per_mb_per_day,
            old.min_span,
            old.max_span,
            old.is_send_telemetry,
            old.is_initialized
        );
    end if;
    return new;
end;
$BODY$;

ALTER FUNCTION providers.log_provider_update()
    OWNER TO pguser;

-- providers.log_status_history()
CREATE FUNCTION providers.log_status_history()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
begin
    insert into providers.statuses_history (
        public_key,
        check_time,
        is_online
    ) values (
        new.public_key,
        new.check_time,
        new.is_online
    );
    return new;
end;
$BODY$;

ALTER FUNCTION providers.log_status_history()
    OWNER TO pguser;

-- Trigger: benchmarks_archive_after_update
CREATE OR REPLACE TRIGGER benchmarks_archive_after_update
    AFTER UPDATE 
    ON providers.benchmarks
    FOR EACH ROW
    EXECUTE FUNCTION providers.archive_benchmarks_after_update();

-- Trigger: trg_log_provider_update
CREATE OR REPLACE TRIGGER trg_log_provider_update
    BEFORE UPDATE 
    ON providers.providers
    FOR EACH ROW
    EXECUTE FUNCTION providers.log_provider_update();

-- Trigger: trg_log_status_update
CREATE OR REPLACE TRIGGER trg_log_status_update
    AFTER UPDATE 
    ON providers.statuses
    FOR EACH ROW
    EXECUTE FUNCTION providers.log_status_history();

-- Trigger: telemetry_archive_before_delete
CREATE OR REPLACE TRIGGER telemetry_archive_before_delete
    BEFORE DELETE
    ON providers.telemetry
    FOR EACH ROW
    EXECUTE FUNCTION providers.archive_telemetry();
    
-- Trigger: telemetry_archive_before_update
CREATE OR REPLACE TRIGGER telemetry_archive_before_update
    BEFORE UPDATE 
    ON providers.telemetry
    FOR EACH ROW
    EXECUTE FUNCTION providers.archive_telemetry();





-- system.params
CREATE TABLE IF NOT EXISTS system.params
(
    key character varying(256) COLLATE pg_catalog."default" NOT NULL,
    value character varying(1024) COLLATE pg_catalog."default",
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT params_pkey PRIMARY KEY (key)
)

TABLESPACE pg_default;

ALTER TABLE system.params
    OWNER to pguser;

COMMENT ON TABLE system.params IS 'To store sustem settings and parameters of the application';
