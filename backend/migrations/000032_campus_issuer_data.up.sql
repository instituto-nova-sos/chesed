-- Sprint 10 (S11.5) — campus fiscal identity for donation receipts.
-- Nullable so existing campuses stay valid; the receipt renders "—" when absent.
-- campus is NOT under RLS (not in the 000028 policy set), so no policy change.
ALTER TABLE campus
    ADD COLUMN legal_name   VARCHAR(200),
    ADD COLUMN cnpj         VARCHAR(18),
    ADD COLUMN address_line VARCHAR(300),
    ADD COLUMN city_name    VARCHAR(100),
    ADD COLUMN state_code   VARCHAR(2),
    ADD COLUMN zip_code      VARCHAR(9);
