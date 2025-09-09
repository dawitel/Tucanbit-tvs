-- name: CreateTransaction :exec
INSERT INTO transactions (
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
);

-- name: GetTransactionByHash :one
SELECT
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
FROM transactions
WHERE chain_id = $1 AND tx_hash = $2;

-- name: GetTransactionByID :one
SELECT
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
FROM transactions
WHERE id = $1;

-- name: UpdateTransaction :execrows
UPDATE transactions SET
    deposit_session_id = $2, withdrawal_id = $3, chain_id = $4,
    network = $5, crypto_currency = $6, tx_hash = $7,
    from_address = $8, to_address = $9, amount = $10,
    usd_amount_cents = $11, exchange_rate = $12, fee = $13,
    block_number = $14, block_hash = $15, status = $16,
    confirmations = $17, timestamp = $18, verified_at = $19,
    processor = $20, transaction_type = $21, metadata = $22,
    updated_at = $23
WHERE id = $1;

-- name: UpdateTransactionStatus :execrows
UPDATE transactions SET
    status = $2,
    verified_at = $3,
    metadata = COALESCE($4, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetTransactionsByAddress :many
SELECT
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
FROM transactions
WHERE chain_id = $1 AND (from_address = $2 OR to_address = $2)
ORDER BY timestamp DESC
LIMIT $3 OFFSET $4;

-- name: GetPendingTransactions :many
SELECT
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
FROM transactions
WHERE status IN ('pending', 'processing')
ORDER BY created_at ASC
LIMIT $1;

-- name: GetTransactionsByStatus :many
SELECT
    id, deposit_session_id, withdrawal_id, chain_id, network, crypto_currency,
    tx_hash, from_address, to_address, amount, usd_amount_cents, exchange_rate,
    fee, block_number, block_hash, status, confirmations, timestamp,
    verified_at, processor, transaction_type, metadata, created_at, updated_at
FROM transactions
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteTransaction :execrows
UPDATE transactions SET
    status = 'deleted',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetTransactionStats :one
SELECT
    COUNT(*) as total_transactions,
    COUNT(CASE WHEN status = 'verified' THEN 1 END) as verified_count,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing_count,
    COALESCE(SUM(CAST(amount AS NUMERIC)), 0) as total_amount,
    COALESCE(AVG(confirmations), 0) as avg_confirmations
FROM transactions
WHERE chain_id = $1;

-- name: ListPendingDepositSessions :many
SELECT * FROM deposit_sessions
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateDepositSessionStatus :exec
UPDATE deposit_sessions
SET
    status = $2,
    error_message = COALESCE($3, error_message),
    updated_at = CURRENT_TIMESTAMP
WHERE session_id = $1;

-- name: CompleteDepositSession :exec
UPDATE deposit_sessions
SET
    status = $1,
    metadata = $2,
    error_message = COALESCE($3, error_message),
    updated_at = CURRENT_TIMESTAMP
WHERE session_id = $4;

-- name: GetUserBalances :many
SELECT * FROM balances
WHERE user_id = $1;

-- name: GetBalance :one
SELECT * FROM balances
WHERE user_id = $1 AND currency_code = $2;

-- name: ReserveBalance :exec
UPDATE balances
SET reserved_cents = reserved_cents + $3,
    amount_cents = amount_cents - $3,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND currency_code = $2 AND amount_cents >= $3;

-- name: LogBalanceChange :exec
INSERT INTO balance_logs (
    id, user_id, component, currency_code, change_cents, change_units,
    description, timestamp, balance_after_cents, balance_after_units,
    transaction_id, status
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
);

-- name: UpdateBalance :exec
UPDATE balances
SET
    amount_cents = $3,
    amount_units = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND currency_code = $2;

-- name: ReleaseReservedBalance :exec
UPDATE balances
SET
    reserved_cents = reserved_cents - $3,
    amount_cents = amount_cents + $3,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND currency_code = $2 AND reserved_cents >= $3;

-- name: ListPendingWithdrawals :many
SELECT * FROM withdrawals
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateWithdrawal :exec
UPDATE withdrawals
SET
    tx_hash = COALESCE($1, tx_hash),
    status = COALESCE($2, status),
    error_message = COALESCE($3, error_message),
    updated_at = CURRENT_TIMESTAMP
WHERE withdrawal_id = $4;

-- name: GetTransactionByDepositSessionID :one
SELECT * FROM transactions
WHERE deposit_session_id = $1;

-- name: GetTransactionByWithdrawalID :one
SELECT * FROM transactions
WHERE withdrawal_id = $1;

-- name: GetDepositSessionByID :one
SELECT * FROM deposit_sessions
WHERE session_id = $1;

-- name: GetWithdrawalByID :one
SELECT * FROM withdrawals
WHERE withdrawal_id = $1;
