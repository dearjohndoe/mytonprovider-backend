-- parse_speed_to_int: bigint result and units with spaces ("107.7 MiB/s").
-- integer overflow (~2048 MiB/s) aborts the whole rating update.
-- Run manually on production BEFORE deploying coordinator with the new rating formula.
-- CREATE OR REPLACE cannot change the return type, so the function is dropped first.

DROP FUNCTION IF EXISTS providers.parse_speed_to_int(text);

CREATE FUNCTION providers.parse_speed_to_int(
	speed_text text)
    RETURNS bigint
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
    unit := regexp_replace(regexp_replace(speed_text, '[0-9\.]', '', 'g'), '\s', '', 'g');

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

    RETURN (value * multiplier)::bigint;
END;
$BODY$;
