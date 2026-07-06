-- Add an IANA timezone to campus so multi-region campuses can localize dates
-- (S09.4). Existing rows default to the São Paulo zone.
ALTER TABLE campus
    ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'America/Sao_Paulo';
