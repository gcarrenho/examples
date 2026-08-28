-- Schema for Case 10 — core banking ledger
-- Applied automatically by Postgres on first `docker compose up`.

CREATE TABLE IF NOT EXISTS accounts (
    iban          TEXT PRIMARY KEY,
    balance_cents BIGINT NOT NULL DEFAULT 1000000,  -- 10,000.00 starting balance for demo
    version       BIGINT NOT NULL DEFAULT 0         -- OCC guard: incremented on every debit/reverse
);

-- payment_debits records every accounting entry so Settle and Reverse can find
-- the original amount and account without the caller repeating them.
CREATE TABLE IF NOT EXISTS payment_debits (
    order_id     TEXT PRIMARY KEY,
    iban         TEXT NOT NULL REFERENCES accounts(iban),
    amount_cents BIGINT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'SETTLED', 'REVERSED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
