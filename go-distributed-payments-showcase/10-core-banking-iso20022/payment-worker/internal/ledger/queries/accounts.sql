-- Queries for the accounts table.

-- name: UpsertAccount :exec
INSERT INTO accounts (iban, balance_cents, version)
VALUES ($1, 1000000, 0) ON CONFLICT DO NOTHING;

-- name: GetAccount :one
SELECT balance_cents, version FROM accounts WHERE iban = $1;

-- name: DebitAccount :execrows
UPDATE accounts
SET balance_cents = balance_cents - $1,
    version       = version + 1
WHERE iban = $2
  AND version = $3
  AND balance_cents >= $1;

-- name: GetAccountForConflict :one
SELECT version, balance_cents FROM accounts WHERE iban = $1;

-- name: RestoreBalance :exec
UPDATE accounts
SET balance_cents = balance_cents + $1,
    version       = version + 1
WHERE iban = $2;
