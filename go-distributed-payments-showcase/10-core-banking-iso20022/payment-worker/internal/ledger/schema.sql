CREATE TABLE IF NOT EXISTS accounts (
    iban          TEXT PRIMARY KEY,
    balance_cents BIGINT NOT NULL DEFAULT 1000000,
    version       BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS payment_debits (
    order_id     TEXT PRIMARY KEY,
    iban         TEXT NOT NULL REFERENCES accounts(iban),
    amount_cents BIGINT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'SETTLED', 'REVERSED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
