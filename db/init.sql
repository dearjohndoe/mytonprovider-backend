
-- SCHEMAS

CREATE SCHEMA providers AUTHORIZATION pguser;

CREATE SCHEMA public AUTHORIZATION pg_database_owner;

CREATE SCHEMA system AUTHORIZATION pguser;

-- TABLES

CREATE TABLE IF NOT EXISTS system.reason_codes (
    code        int         PRIMARY KEY,
    description text        NOT NULL
);
INSERT INTO system.reason_codes (code, description) VALUES
    (0, 'Valid storage proof'),
    (101, 'IP Not Found'),
    (102, 'Not Found(impossible)'),
    (103, 'Unavailable Provider'),
    (201, 'Ping Failed'),
    (202, 'Invalid Bag ID'),
    (301, 'Get Info Failed'),
    (302, 'Invalid Header'),
    (401, 'Cant Get Piece'),
    (402, 'Cant Parse BoC'),
    (403, 'Proof Check Failed');

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
);

CREATE TABLE IF NOT EXISTS providers.benchmarks_history
(
    id serial,
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
);

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
    is_initialized boolean NOT NULL DEFAULT false,
    uptime double precision NOT NULL DEFAULT 0.0,
    max_bag_size_bytes bigint NOT NULL DEFAULT 0,
    last_tx_lt bigint NOT NULL DEFAULT 0,
    ip character varying(16) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
    port integer DEFAULT 0,
    status integer,
    status_ratio real NOT NULL DEFAULT 0,
    ip_info jsonb DEFAULT '{}'::jsonb,
    storage_ip character varying(16) COLLATE pg_catalog."default",
    storage_port integer,
    statuses_reason_stats JSONB DEFAULT '[]'::JSONB,
    CONSTRAINT providers_pkey PRIMARY KEY (public_key),
    CONSTRAINT providers_address_key UNIQUE (address)
);

CREATE TABLE IF NOT EXISTS providers.providers_history
(
    id serial,
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
    is_initialized boolean NOT NULL,
    uptime double precision NOT NULL DEFAULT 0.0,
    CONSTRAINT providers_history_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS providers.statuses
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    check_time timestamp with time zone NOT NULL,
    is_online boolean NOT NULL,
    CONSTRAINT statuses_pkey PRIMARY KEY (public_key)
);

CREATE TABLE IF NOT EXISTS providers.statuses_history
(
    public_key character varying(64) COLLATE pg_catalog."default" NOT NULL,
    check_time timestamp with time zone NOT NULL,
    is_online boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS providers.storage_contracts
(
    address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    provider_address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    bag_id character varying(64) COLLATE pg_catalog."default" NOT NULL,
    owner_address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    size bigint NOT NULL,
    chunk_size bigint NOT NULL,
    last_tx_lt bigint NOT NULL,
    reason integer,
    reason_timestamp timestamp with time zone,
    CONSTRAINT storage_contracts_pkey PRIMARY KEY (address, provider_address)
);

CREATE INDEX IF NOT EXISTS idx_storage_contracts_address
    ON providers.storage_contracts USING btree
    (address COLLATE pg_catalog."default");

CREATE TABLE IF NOT EXISTS providers.storage_contracts_history
(
    address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    provider_address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    bag_id character varying(64) COLLATE pg_catalog."default" NOT NULL,
    owner_address character varying(64) COLLATE pg_catalog."default" NOT NULL,
    size bigint NOT NULL,
    chunk_size bigint NOT NULL,
    last_tx_lt bigint NOT NULL,
    reason integer,
    reason_timestamp timestamp with time zone,
    deleted_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT storage_contracts_history_pkey PRIMARY KEY (address, provider_address, deleted_at)
);

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
    updated_at timestamp with time zone DEFAULT now(),
    x_real_ip character varying(16) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
    disks_load jsonb NOT NULL DEFAULT '{}'::jsonb,
    disks_load_percent jsonb NOT NULL DEFAULT '{}'::jsonb,
    iops jsonb NOT NULL DEFAULT '{}'::jsonb,
    net_load double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    net_recv double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    net_sent double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    pps double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    CONSTRAINT telemetry_pkey PRIMARY KEY (public_key)
);

CREATE TABLE IF NOT EXISTS providers.telemetry_history
(
    id serial,
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
    x_real_ip character varying(16) COLLATE pg_catalog."default" DEFAULT NULL::character varying,
    pps double precision[][],
    iops jsonb NOT NULL DEFAULT '{}'::jsonb,
    net_sent double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    net_recv double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    net_load double precision[][] NOT NULL DEFAULT '{}'::double precision[],
    disks_load jsonb NOT NULL DEFAULT '{}'::jsonb,
    disks_load_percent jsonb NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT telemetry_history_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS system.params
(
    key character varying(256) COLLATE pg_catalog."default" NOT NULL,
    value character varying(1024) COLLATE pg_catalog."default",
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT params_pkey PRIMARY KEY (key)
);

-- FUNCTIONS AND TRIGGERS

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
    value := regexp_replace(speed_text, '[^0-9\.]', '', 'g')::numeric;
    unit := regexp_replace(speed_text, '[0-9\.]', '', 'g');

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

CREATE FUNCTION providers.archive_telemetry()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    INSERT INTO providers.telemetry_history (
        public_key, storage_git_hash, provider_git_hash, cpu_name, pings, cpu_product_name,
        uname_sysname, uname_release, uname_version, uname_machine, disk_name, cpu_load, 
        total_space, free_space, used_space, used_provider_space, total_provider_space, 
        total_swap, usage_swap, swap_usage_percent, usage_ram, total_ram, ram_usage_percent, 
        cpu_number, cpu_is_virtual, x_real_ip,
        disks_load, disks_load_percent, iops, pps, net_load, net_recv, net_sent
    ) VALUES (
        OLD.public_key, OLD.storage_git_hash, OLD.provider_git_hash, OLD.cpu_name, OLD.pings, OLD.cpu_product_name,
        OLD.uname_sysname, OLD.uname_release, OLD.uname_version, OLD.uname_machine, OLD.disk_name, OLD.cpu_load, 
        OLD.total_space, OLD.free_space, OLD.used_space, OLD.used_provider_space, OLD.total_provider_space, 
        OLD.total_swap, OLD.usage_swap, OLD.swap_usage_percent, OLD.usage_ram, OLD.total_ram, OLD.ram_usage_percent, 
        OLD.cpu_number, OLD.cpu_is_virtual, OLD.x_real_ip,
        COALESCE(CASE WHEN jsonb_typeof(OLD.disks_load) = 'object' THEN OLD.disks_load ELSE '{}'::jsonb END, '{}'::jsonb),
        COALESCE(CASE WHEN jsonb_typeof(OLD.disks_load_percent) = 'object' THEN OLD.disks_load_percent ELSE '{}'::jsonb END, '{}'::jsonb),
        COALESCE(CASE WHEN jsonb_typeof(OLD.iops) = 'object' THEN OLD.iops ELSE '{}'::jsonb END, '{}'::jsonb),
        OLD.pps, OLD.net_load, OLD.net_recv, OLD.net_sent
    );
    RETURN NEW;
END;
$BODY$;

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
            old.is_initialized
        );
    end if;
    return new;
end;
$BODY$;

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

CREATE FUNCTION providers.move_to_storage_contracts_history()
    RETURNS trigger
    LANGUAGE plpgsql
    COST 100
    VOLATILE NOT LEAKPROOF
AS $BODY$
BEGIN
    -- Insert the deleted record into the history table
    INSERT INTO providers.storage_contracts_history (
        address,
        provider_address,
        bag_id,
        owner_address,
        size,
        chunk_size,
        last_tx_lt,
        reason,
        reason_timestamp,
        deleted_at
    ) VALUES (
        OLD.address,
        OLD.provider_address,
        OLD.bag_id,
        OLD.owner_address,
        OLD.size,
        OLD.chunk_size,
        OLD.last_tx_lt,
        OLD.reason,
        OLD.reason_timestamp,
        now()
    );
    
    RETURN OLD;
END;
$BODY$;

CREATE TRIGGER benchmarks_archive_after_update 
AFTER UPDATE ON providers.benchmarks 
FOR EACH ROW EXECUTE FUNCTION providers.archive_benchmarks_after_update();

CREATE TRIGGER trg_log_provider_update 
BEFORE UPDATE ON providers.providers 
FOR EACH ROW EXECUTE FUNCTION providers.log_provider_update();

CREATE TRIGGER trg_log_status_insert 
AFTER INSERT ON providers.statuses 
FOR EACH ROW EXECUTE FUNCTION providers.log_status_history();

CREATE TRIGGER trg_log_status_update 
AFTER UPDATE ON providers.statuses 
FOR EACH ROW EXECUTE FUNCTION providers.log_status_history();

CREATE TRIGGER storage_contracts_delete_trigger 
BEFORE DELETE ON providers.storage_contracts 
FOR EACH ROW EXECUTE FUNCTION providers.move_to_storage_contracts_history();

CREATE TRIGGER telemetry_archive_before_delete 
BEFORE DELETE ON providers.telemetry 
FOR EACH ROW EXECUTE FUNCTION providers.archive_telemetry();

CREATE TRIGGER telemetry_archive_before_update 
BEFORE UPDATE ON providers.telemetry 
FOR EACH ROW EXECUTE FUNCTION providers.archive_telemetry();
