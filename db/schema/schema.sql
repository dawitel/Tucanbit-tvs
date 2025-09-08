-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Define custom types
CREATE TYPE deposit_session_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled', 'expired');
CREATE TYPE withdrawal_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled', 'awaiting_admin_review');
CREATE TYPE ProcessorType AS ENUM ('internal', 'pdm');
CREATE TYPE components AS ENUM ('real_money', 'bonus_money', 'points');
CREATE TYPE currency_type AS ENUM ('fiat', 'crypto');
CREATE TYPE conversion_type AS ENUM ('deposit', 'withdrawal', 'exchange');

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(20) UNIQUE NOT NULL CHECK (username <> ''),
    first_name VARCHAR DEFAULT '',
    last_name VARCHAR DEFAULT '',
    phone_number VARCHAR(15) UNIQUE NOT NULL CHECK (phone_number <> ''),
    password TEXT NOT NULL CHECK (password <> ''),
    email VARCHAR UNIQUE CHECK (email <> ''),
    date_of_birth VARCHAR DEFAULT '',
    profile VARCHAR,
    default_currency VARCHAR(3) CHECK (default_currency ~ '^[A-Z]{3}$'),
    source VARCHAR,
    referral_code VARCHAR UNIQUE,
    referral_type VARCHAR,
    referred_by_code VARCHAR,
    user_type VARCHAR(50) DEFAULT 'PLAYER' CHECK (user_type IN ('PLAYER', 'ADMIN', 'AFFILIATE')),
    street_address VARCHAR DEFAULT '',
    country VARCHAR DEFAULT '',
    state VARCHAR DEFAULT '',
    city VARCHAR DEFAULT '',
    postal_code VARCHAR DEFAULT '',
    kyc_status VARCHAR DEFAULT 'PENDING' CHECK (kyc_status IN ('PENDING', 'VERIFIED', 'REJECTED')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Currency configuration table
CREATE TABLE IF NOT EXISTS currency_config (
    currency_code VARCHAR(10) PRIMARY KEY,
    currency_name VARCHAR(50) NOT NULL,
    currency_type currency_type NOT NULL,
    decimal_places INTEGER NOT NULL CHECK (decimal_places BETWEEN 0 AND 18),
    smallest_unit_name VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert common currencies
INSERT INTO currency_config (currency_code, currency_name, currency_type, decimal_places, smallest_unit_name) VALUES
('USD', 'US Dollar', 'fiat', 2, 'Cent'),
('BTC', 'Bitcoin', 'crypto', 8, 'Satoshi'),
('ETH', 'Ethereum', 'crypto', 18, 'Wei'),
('USDT', 'Tether', 'crypto', 6, 'Micro-USDT'),
('USDC', 'USD Coin', 'crypto', 6, 'Micro-USDC'),
('LTC', 'Litecoin', 'crypto', 8, 'Photon')
ON CONFLICT (currency_code) DO NOTHING;

-- System configuration for withdrawal thresholds
CREATE TABLE IF NOT EXISTS system_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL CHECK (config_key <> ''),
    config_value JSONB NOT NULL,
    description TEXT,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default configuration
INSERT INTO system_config (config_key, config_value, description) VALUES
('withdrawal_threshold', '{"usd_amount_cents": 500000, "auto_approve": true}', 'Maximum USD amount in cents for automatic withdrawal processing'),
('auto_withdrawal_limit', '{"daily_usd_cents": 2000000, "weekly_usd_cents": 5000000, "monthly_usd_cents": 10000000}', 'Daily, weekly, and monthly withdrawal limits in cents'),
('conversion_settings', '{"rounding_method": "bankers", "intermediate_precision": 10}', 'Currency conversion settings')
ON CONFLICT (config_key) DO NOTHING;

-- Balances table
CREATE TABLE IF NOT EXISTS balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency_code VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    amount_cents BIGINT DEFAULT 0 CHECK (amount_cents >= 0),
    amount_units DECIMAL(36,18) DEFAULT 0 CHECK (amount_units >= 0),
    reserved_cents BIGINT DEFAULT 0 CHECK (reserved_cents >= 0),
    reserved_units DECIMAL(36,18) DEFAULT 0 CHECK (reserved_units >= 0),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_currency UNIQUE (user_id, currency_code)
);

-- Operational Groups table
CREATE TABLE IF NOT EXISTS operational_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL CHECK (name <> ''),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Operational Types table
CREATE TABLE IF NOT EXISTS operational_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES operational_groups(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL CHECK (name <> ''),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Balance Logs table
CREATE TABLE IF NOT EXISTS balance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    component components NOT NULL,
    currency_code VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    change_cents BIGINT NOT NULL DEFAULT 0,  -- For fiat currencies
    change_units DECIMAL(36,18) NOT NULL DEFAULT 0,  -- For crypto currencies
    operational_group_id UUID REFERENCES operational_groups(id) ON DELETE SET NULL,
    operational_type_id UUID REFERENCES operational_types(id) ON DELETE SET NULL,
    description TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    balance_after_cents BIGINT,  -- For fiat currencies
    balance_after_units DECIMAL(36,18),  -- For crypto currencies
    transaction_id VARCHAR(255),
    status VARCHAR(50) CHECK (status IN ('PENDING', 'COMPLETED', 'FAILED')),
    CONSTRAINT check_non_zero_change CHECK (change_cents <> 0 OR change_units <> 0),
    CONSTRAINT check_currency_representation CHECK (
        (currency_code = 'USD' AND change_units = 0 AND balance_after_units IS NULL) OR
        (currency_code != 'USD' AND change_cents = 0 AND balance_after_cents IS NULL)
    )
);

-- Manual Funds table
CREATE TABLE IF NOT EXISTS manual_funds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_id VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('DEPOSIT', 'WITHDRAWAL', 'ADJUSTMENT')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency_code VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    note TEXT NOT NULL,
    reason VARCHAR(50) NOT NULL DEFAULT 'system_restart',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Sport Bets table
CREATE TABLE IF NOT EXISTS sport_bets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255) UNIQUE NOT NULL CHECK (transaction_id <> ''),
    bet_amount_cents BIGINT NOT NULL CHECK (bet_amount_cents > 0),  -- Stored in cents
    bet_reference_num VARCHAR(255) NOT NULL CHECK (bet_reference_num <> ''),
    game_reference VARCHAR(255) NOT NULL CHECK (game_reference <> ''),
    bet_mode VARCHAR(50) NOT NULL CHECK (bet_mode <> ''),
    description TEXT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    frontend_type VARCHAR(50),
    status VARCHAR(50) CHECK (status IN ('PENDING', 'SETTLED', 'CANCELLED', 'VOID')),
    sport_ids TEXT,
    site_id VARCHAR(255) NOT NULL CHECK (site_id <> ''),
    client_ip INET,
    affiliate_user_id VARCHAR(255),
    autorecharge VARCHAR(10),
    bet_details JSONB NOT NULL,
    currency_code VARCHAR(10) NOT NULL DEFAULT 'USD' REFERENCES currency_config(currency_code),
    potential_win_cents BIGINT,
    actual_win_cents BIGINT,
    odds DECIMAL(10,4),
    placed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    settled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Exchange Rates table
CREATE TABLE IF NOT EXISTS exchange_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    to_currency VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    rate DECIMAL(15,6) NOT NULL CHECK (rate > 0),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_currency_pair UNIQUE (from_currency, to_currency)
);

-- User Sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL CHECK (token <> ''),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    refresh_token VARCHAR(255),
    refresh_token_expires_at TIMESTAMP WITH TIME ZONE
);

-- Login Attempts table
CREATE TABLE IF NOT EXISTS login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Supported Chains table
CREATE TABLE IF NOT EXISTS supported_chains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id TEXT NOT NULL CHECK (chain_id <> ''),
    name TEXT NOT NULL CHECK (name <> ''),
    networks TEXT[] NOT NULL DEFAULT '{}',
    crypto_currencies TEXT[] NOT NULL DEFAULT '{}',
    processor ProcessorType NOT NULL DEFAULT 'internal',
    is_testnet BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_chain_id UNIQUE (chain_id)
);

-- Deposit Sessions table
CREATE TABLE IF NOT EXISTS deposit_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL CHECK (session_id <> ''),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chain_id VARCHAR(50) NOT NULL REFERENCES supported_chains(chain_id) ON DELETE CASCADE,
    network VARCHAR(50) NOT NULL CHECK (network <> ''),
    wallet_address VARCHAR(255),
    amount DECIMAL(36,18) NOT NULL CHECK (amount > 0),  -- Crypto amount with full precision
    crypto_currency VARCHAR(10) NOT NULL CHECK (crypto_currency <> ''),
    status deposit_session_status NOT NULL DEFAULT 'pending',
    qr_code_data TEXT,
    payment_link TEXT,
    metadata JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) NOT NULL REFERENCES deposit_sessions(session_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chain_id VARCHAR(50) NOT NULL REFERENCES supported_chains(chain_id) ON DELETE CASCADE,
    crypto_currency VARCHAR(10) NOT NULL CHECK (crypto_currency <> ''),
    network VARCHAR(50) NOT NULL CHECK (network <> ''),
    amount DECIMAL(36,18) NOT NULL CHECK (amount > 0),  -- Crypto amount with full precision
    address VARCHAR(255) UNIQUE NOT NULL CHECK (address <> ''),
    vault_key_path VARCHAR(500) NOT NULL CHECK (vault_key_path <> ''),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP WITH TIME ZONE
);


-- Withdrawals table
CREATE TABLE IF NOT EXISTS withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    admin_id UUID REFERENCES users(id) ON DELETE SET NULL,
    withdrawal_id VARCHAR(255) UNIQUE NOT NULL CHECK (withdrawal_id <> ''),
    chain_id VARCHAR(50) NOT NULL REFERENCES supported_chains(chain_id),
    network VARCHAR(50) NOT NULL CHECK (network <> ''),
    crypto_currency VARCHAR(10) NOT NULL CHECK (crypto_currency <> ''),
    usd_amount_cents BIGINT NOT NULL CHECK (usd_amount_cents > 0),  -- Requested amount in cents
    crypto_amount DECIMAL(36,18) NOT NULL CHECK (crypto_amount > 0),  -- Crypto amount with full precision
    exchange_rate DECIMAL(15,6) NOT NULL CHECK (exchange_rate > 0),
    fee_cents BIGINT NOT NULL DEFAULT 0 CHECK (fee_cents >= 0),  -- Fee in cents
    to_address VARCHAR(255) NOT NULL CHECK (to_address <> ''),
    tx_hash VARCHAR(100),
    status withdrawal_status NOT NULL DEFAULT 'pending',
    requires_admin_review BOOLEAN NOT NULL DEFAULT FALSE,
    admin_review_deadline TIMESTAMP WITH TIME ZONE,
    processed_by_system BOOLEAN DEFAULT FALSE,
    source_wallet_address VARCHAR(255) NOT NULL CHECK (source_wallet_address <> ''),
    amount_reserved_cents BIGINT NOT NULL CHECK (amount_reserved_cents > 0),  -- Reserved amount in cents
    reservation_released BOOLEAN DEFAULT FALSE,
    reservation_released_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_session_id VARCHAR(255) REFERENCES deposit_sessions(session_id) ON DELETE CASCADE,
    withdrawal_id VARCHAR(255) REFERENCES withdrawals(withdrawal_id) ON DELETE CASCADE,
    chain_id VARCHAR(50) NOT NULL REFERENCES supported_chains(chain_id) ON DELETE CASCADE,
    network VARCHAR(50) NOT NULL CHECK (network <> ''),
    crypto_currency VARCHAR(50) NOT NULL CHECK (crypto_currency <> ''),
    tx_hash VARCHAR(100) NOT NULL CHECK (tx_hash <> ''),
    from_address VARCHAR(100) NOT NULL CHECK (from_address <> ''),
    to_address VARCHAR(100) NOT NULL CHECK (to_address <> ''),
    amount DECIMAL(36,18) NOT NULL DEFAULT 0,  -- Crypto amount with full precision
    usd_amount_cents BIGINT,  -- Converted amount in cents
    exchange_rate DECIMAL(15,6),
    fee DECIMAL(36,18) DEFAULT 0,  -- Crypto fee with full precision
    block_number BIGINT,
    block_hash VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verified', 'failed', 'processing')),
    confirmations INTEGER NOT NULL DEFAULT 0 CHECK (confirmations >= 0),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP WITH TIME ZONE,
    processor ProcessorType NOT NULL DEFAULT 'internal',
    transaction_type VARCHAR(20) NOT NULL DEFAULT 'withdrawal' CHECK (transaction_type IN ('deposit', 'withdrawal')),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Conversion remainders table
CREATE TABLE IF NOT EXISTS conversion_remainders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL,
    original_amount DECIMAL(36,18) NOT NULL,
    converted_amount BIGINT NOT NULL,
    remainder_amount DECIMAL(10,8) NOT NULL,
    currency_code VARCHAR(10) NOT NULL REFERENCES currency_config(currency_code),
    conversion_type conversion_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Admin Fund Movements table
CREATE TABLE IF NOT EXISTS admin_fund_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_address VARCHAR(255) NOT NULL CHECK (from_address <> ''),
    to_address VARCHAR(255) NOT NULL CHECK (to_address <> ''),
    chain_id VARCHAR(50) NOT NULL REFERENCES supported_chains(chain_id),
    network VARCHAR(50) NOT NULL CHECK (network <> ''),
    crypto_currency VARCHAR(10) NOT NULL CHECK (crypto_currency <> ''),
    amount DECIMAL(36,18) NOT NULL CHECK (amount > 0),  -- Crypto amount with full precision
    tx_hash VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed')),
    movement_type VARCHAR(20) NOT NULL CHECK (movement_type IN ('to_cold_storage', 'to_hot_wallet', 'rebalance')),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Audit Logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action VARCHAR(50) NOT NULL CHECK (action <> ''),
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type <> ''),
    entity_id VARCHAR(255) NOT NULL CHECK (entity_id <> ''),
    admin_id UUID REFERENCES users(id) ON DELETE SET NULL,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Service API Keys table
CREATE TABLE IF NOT EXISTS service_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issuer_service TEXT NOT NULL,
    receiver_service TEXT NOT NULL,
    key TEXT NOT NULL CHECK (key <> ''),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_service_key UNIQUE (issuer_service, key)
);


-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_phone_number ON users(phone_number);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);
CREATE INDEX IF NOT EXISTS idx_balance_logs_user_id ON balance_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_balance_logs_timestamp ON balance_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_manual_funds_user_id ON manual_funds(user_id);
CREATE INDEX IF NOT EXISTS idx_manual_funds_admin_id ON manual_funds(admin_id);
CREATE INDEX IF NOT EXISTS idx_sport_bets_user_id ON sport_bets(user_id);
CREATE INDEX IF NOT EXISTS idx_sport_bets_transaction_id ON sport_bets(transaction_id);
CREATE INDEX IF NOT EXISTS idx_sport_bets_status ON sport_bets(status);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_from_currency ON exchange_rates(from_currency);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_to_currency ON exchange_rates(to_currency);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token);
CREATE INDEX IF NOT EXISTS idx_login_attempts_user_id ON login_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_deposit_sessions_user_id ON deposit_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_deposit_sessions_session_id ON deposit_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_deposit_sessions_status ON deposit_sessions(status);
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_transactions_session_id ON transactions(deposit_session_id);
CREATE INDEX IF NOT EXISTS idx_transactions_chain_id ON transactions(chain_id);
CREATE INDEX IF NOT EXISTS idx_transactions_tx_hash ON transactions(tx_hash);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_supported_chains_chain_id ON supported_chains(chain_id);
CREATE INDEX IF NOT EXISTS idx_supported_chains_status ON supported_chains(status);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_withdrawal_id ON withdrawals(withdrawal_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_status ON withdrawals(status);
CREATE INDEX IF NOT EXISTS idx_withdrawals_requires_admin_review ON withdrawals(requires_admin_review);
CREATE INDEX IF NOT EXISTS idx_admin_fund_movements_admin_id ON admin_fund_movements(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_fund_movements_from_address ON admin_fund_movements(from_address);
CREATE INDEX IF NOT EXISTS idx_admin_fund_movements_status ON admin_fund_movements(status);
CREATE INDEX IF NOT EXISTS idx_conversion_remainders_transaction_id ON conversion_remainders(transaction_id);
