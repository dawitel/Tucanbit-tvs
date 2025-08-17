-- name: CreateTransaction :exec
INSERT INTO transactions (
    id, chain_id, chain_type, tx_hash, from_address, to_address,
    amount, fee, block_number, block_hash, status, confirmations,
    timestamp, verified_at, processor_id, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
);

-- name: GetTransactionByHash :one
SELECT id, chain_id, chain_type, tx_hash, from_address, to_address,
       amount, fee, block_number, block_hash, status, confirmations,
       timestamp, verified_at, processor_id, metadata, created_at, updated_at
FROM transactions
WHERE chain_id = $1 AND tx_hash = $2;

-- name: GetTransactionByID :one
SELECT id, chain_id, chain_type, tx_hash, from_address, to_address,
       amount, fee, block_number, block_hash, status, confirmations,
       timestamp, verified_at, processor_id, metadata, created_at, updated_at
FROM transactions
WHERE id = $1;

-- name: UpdateTransaction :execrows
UPDATE transactions SET
    chain_id = $2, chain_type = $3, tx_hash = $4,
    from_address = $5, to_address = $6,
    amount = $7, fee = $8, block_number = $9,
    block_hash = $10, status = $11,
    confirmations = $12, timestamp = $13,
    verified_at = $14, processor_id = $15,
    metadata = $16
WHERE id = $1;

-- name: UpdateTransactionStatus :execrows
UPDATE transactions SET
    status = $2, verified_at = $3,
    metadata = COALESCE($4, metadata)
WHERE id = $1;

-- name: GetTransactionsByAddress :many
SELECT id, chain_id, chain_type, tx_hash, from_address, to_address,
       amount, fee, block_number, block_hash, status, confirmations,
       timestamp, verified_at, processor_id, metadata, created_at, updated_at
FROM transactions
WHERE chain_id = $1 AND (from_address = $2 OR to_address = $2)
ORDER BY timestamp DESC
LIMIT $3 OFFSET $4;

-- name: GetPendingTransactions :many
SELECT id, chain_id, chain_type, tx_hash, from_address, to_address,
       amount, fee, block_number, block_hash, status, confirmations,
       timestamp, verified_at, processor_id, metadata, created_at, updated_at
FROM transactions
WHERE status IN ('pending', 'processing')
ORDER BY created_at ASC
LIMIT $1;

-- name: GetTransactionsByStatus :many
SELECT id, chain_id, chain_type, tx_hash, from_address, to_address,
       amount, fee, block_number, block_hash, status, confirmations,
       timestamp, verified_at, processor_id, metadata, created_at, updated_at
FROM transactions
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteTransaction :execrows
UPDATE transactions SET status = 'deleted'
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
