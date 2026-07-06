-- Remove the donation currency CHECK constraint.
ALTER TABLE donation DROP CONSTRAINT IF EXISTS donation_currency_check;
