-- Drop triggers and functions
DROP TRIGGER IF EXISTS update_transactions_updated_at ON transactions;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_transactions_unique_chain_tx;
DROP INDEX IF EXISTS idx_transactions_chain_id;
DROP INDEX IF EXISTS idx_transactions_block_number;
DROP INDEX IF EXISTS idx_transactions_timestamp;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_to_address;
DROP INDEX IF EXISTS idx_transactions_from_address;
DROP INDEX IF EXISTS idx_transactions_chain_tx_hash;

-- Drop table
DROP TABLE IF EXISTS transactions;
