-- Prevent duplicate email within the same campus (active persons only)
CREATE UNIQUE INDEX uq_person_email_campus
    ON person(lower(email), campus_id)
    WHERE email IS NOT NULL AND is_active = TRUE;

-- Prevent duplicate phone within the same campus (active persons only)
CREATE UNIQUE INDEX uq_person_phone_campus
    ON person(phone, campus_id)
    WHERE phone IS NOT NULL AND is_active = TRUE;
