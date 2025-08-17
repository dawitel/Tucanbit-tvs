-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id VARCHAR(50) NOT NULL,
    chain_type VARCHAR(20) NOT NULL CHECK (chain_type IN ('ethereum', 'solana', 'bitcoin')),
    tx_hash VARCHAR(100) NOT NULL,
    from_address VARCHAR(100) NOT NULL,
    to_address VARCHAR(100) NOT NULL,
    amount VARCHAR(50) NOT NULL DEFAULT '0',
    fee VARCHAR(50) DEFAULT '0',
    block_number BIGINT,
    block_hash VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verified', 'failed', 'processing')),
    confirmations INTEGER NOT NULL DEFAULT 0,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    processor_id VARCHAR(50),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_transactions_chain_tx_hash ON transactions(chain_id, tx_hash);
CREATE INDEX IF NOT EXISTS idx_transactions_from_address ON transactions(from_address);
CREATE INDEX IF NOT EXISTS idx_transactions_to_address ON transactions(to_address);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_block_number ON transactions(block_number);
CREATE INDEX IF NOT EXISTS idx_transactions_chain_id ON transactions(chain_id);

-- Create unique constraint on chain_id and tx_hash combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_unique_chain_tx ON transactions(chain_id, tx_hash);
