CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    code VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(256) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE products
ADD COLUMN IF NOT EXISTS category_id INTEGER NULL;


INSERT INTO categories (code, name) VALUES
('clothing', 'Clothing'),
('shoes', 'Shoes'),
('accessories', 'Accessories')
ON CONFLICT (code) DO NOTHING;

-- Backfill products with their category_id using the provided mapping

-- Clothing: PROD001, PROD004, PROD007
UPDATE products
SET category_id = (SELECT id FROM categories WHERE code = 'clothing')
WHERE code IN ('PROD001', 'PROD004', 'PROD007');

-- Shoes: PROD002, PROD006
UPDATE products
SET category_id = (SELECT id FROM categories WHERE code = 'shoes')
WHERE code IN ('PROD002', 'PROD006');

-- Accessories: PROD003, PROD005, PROD008
UPDATE products
SET category_id = (SELECT id FROM categories WHERE code = 'accessories')
WHERE code IN ('PROD003', 'PROD005', 'PROD008');

-- Ensure all rows have a category_id before enforcing constraints
-- If any row remains NULL, this will fail; fix data or adjust mapping if that happens.
-- Now enforce NOT NULL and add FK constraint.
ALTER TABLE products
    ALTER COLUMN category_id SET NOT NULL;

-- Add FK with RESTRICT semantics (delete category only if no products reference it)
-- If a constraint already exists (e.g., from previous attempts), adjust names accordingly.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'products_category_id_fkey'
    ) THEN
        ALTER TABLE products
        ADD CONSTRAINT products_category_id_fkey
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;
    END IF;
END$$;
