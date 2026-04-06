-- Default campus
INSERT INTO campus (id, name, region, city, state, country)
VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Instituto Nova SOS', 'BRAZIL', 'Sao Paulo', 'SP', 'BRA');

-- Service types (fixed seed data for MVP)
INSERT INTO service_type (name, category, description) VALUES
    ('Assistencia Juridica', 'LEGAL', 'Legal assistance and guidance'),
    ('Consulta Medica', 'MEDICAL', 'Medical consultation and referrals'),
    ('Acompanhamento Nutricional', 'NUTRITIONAL', 'Nutritional assessment and guidance'),
    ('Fisioterapia', 'PHYSIOTHERAPY', 'Physical therapy sessions'),
    ('Assistencia Social', 'SOCIAL', 'Social work and support services'),
    ('Apoio Educacional', 'EDUCATIONAL', 'Educational support and tutoring'),
    ('Atendimento Psicologico', 'PSYCHOLOGICAL', 'Psychological counseling and support'),
    ('Outros Servicos', 'OTHER', 'Other services not categorized above');
