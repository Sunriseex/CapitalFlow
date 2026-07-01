-- +goose Up
INSERT INTO categories (id, slug, name, is_default)
SELECT md5('capitalflow-default-category:' || slug)::uuid, slug, name, true
FROM (VALUES
    ('salary', 'Salary'),
    ('deposit_interest', 'Deposit interest'),
    ('food', 'Food'),
    ('transport', 'Transport'),
    ('subscriptions', 'Subscriptions'),
    ('housing', 'Housing'),
    ('health', 'Health'),
    ('education', 'Education'),
    ('investments', 'Investments'),
    ('emergency_fund', 'Emergency fund'),
    ('entertainment', 'Entertainment'),
    ('other', 'Other')
) AS defaults(slug, name)
ON CONFLICT (slug) DO NOTHING;

-- +goose Down
DELETE FROM categories
WHERE id IN (
    SELECT md5('capitalflow-default-category:' || slug)::uuid
    FROM (VALUES
        ('salary'),
        ('deposit_interest'),
        ('food'),
        ('transport'),
        ('subscriptions'),
        ('housing'),
        ('health'),
        ('education'),
        ('investments'),
        ('emergency_fund'),
        ('entertainment'),
        ('other')
    ) AS defaults(slug)
);
