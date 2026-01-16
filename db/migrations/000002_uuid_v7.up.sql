-- K-ERP v0.2 Migration: UUID v7 Function
-- UUID v7 provides time-ordered UUIDs for better index performance

CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS UUID AS $$
DECLARE
    timestamp_ms BIGINT;
    uuid_bytes BYTEA;
BEGIN
    -- Get current timestamp in milliseconds
    timestamp_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

    -- Build UUID bytes: 6 bytes timestamp + 10 bytes random
    uuid_bytes := decode(lpad(to_hex(timestamp_ms), 12, '0'), 'hex') ||
                  gen_random_bytes(10);

    -- Set version to 7 (0111 in bits 48-51)
    uuid_bytes := set_byte(uuid_bytes, 6, (get_byte(uuid_bytes, 6) & 15) | 112);

    -- Set variant to RFC 4122 (10xx in bits 64-65)
    uuid_bytes := set_byte(uuid_bytes, 8, (get_byte(uuid_bytes, 8) & 63) | 128);

    RETURN encode(uuid_bytes, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Helper function to extract timestamp from UUID v7
CREATE OR REPLACE FUNCTION uuid_v7_to_timestamp(uuid_val UUID)
RETURNS TIMESTAMPTZ AS $$
DECLARE
    uuid_hex TEXT;
    timestamp_ms BIGINT;
BEGIN
    uuid_hex := replace(uuid_val::TEXT, '-', '');
    timestamp_ms := ('x' || substring(uuid_hex, 1, 12))::BIT(48)::BIGINT;
    RETURN to_timestamp(timestamp_ms / 1000.0);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION uuid_generate_v7() IS 'Generate UUID v7 with time-ordered prefix for better indexing';
COMMENT ON FUNCTION uuid_v7_to_timestamp(UUID) IS 'Extract timestamp from UUID v7';
