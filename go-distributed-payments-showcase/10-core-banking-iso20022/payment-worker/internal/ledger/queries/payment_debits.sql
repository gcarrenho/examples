-- Queries for the payment_debits journal table.

-- name: InsertDebit :exec
INSERT INTO payment_debits (order_id, iban, amount_cents, status)
VALUES ($1, $2, $3, 'PENDING');

-- name: SettleDebit :exec
UPDATE payment_debits
SET status = 'SETTLED'
WHERE order_id = $1
  AND status   = 'PENDING';

-- name: ReverseDebit :one
UPDATE payment_debits
SET status = 'REVERSED'
WHERE order_id = $1
  AND status   = 'PENDING'
RETURNING iban, amount_cents;
