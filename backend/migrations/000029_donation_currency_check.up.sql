-- Constrain donation.currency to the supported ISO 4217 codes (S09.3).
ALTER TABLE donation
    ADD CONSTRAINT donation_currency_check
    CHECK (currency IN ('BRL', 'USD', 'EUR'));
