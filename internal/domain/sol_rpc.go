package domain

type SPLTokenType string
type SolanaClusterType string

const (
	SPLTokenTypeUSDC SPLTokenType = "USDC"
	SPLTokenTypeUSDT SPLTokenType = "USDT"
	SPLTokenTypeSOL  SPLTokenType = "SOL"

	SolanaClusterTypeDevnet  SolanaClusterType = "devnet"
	SolanaClusterTypeTestnet SolanaClusterType = "testnet"
	SolanaClusterTypeMainnet SolanaClusterType = "mainnet-beta"
)

type GetSignaturesForAddressOptions struct {
	Limit          int    `json:"limit,omitempty"`
	Before         string `json:"before,omitempty"`
	Until          string `json:"until,omitempty"`
	Commitment     string `json:"commitment,omitempty"`
	MinContextSlot uint64 `json:"minContextSlot,omitempty"`
}

type HeliusTransaction struct {
	Description      string           `json:"description"`
	Type             string           `json:"type"`
	Source           string           `json:"source"`
	Fee              int64            `json:"fee"`
	FeePayer         string           `json:"feePayer"`
	Signature        string           `json:"signature"`
	Slot             int64            `json:"slot"`
	Timestamp        int64            `json:"timestamp"`
	NativeTransfers  []NativeTransfer `json:"nativeTransfers"`
	TokenTransfers   []TokenTransfer  `json:"tokenTransfers"`
	AccountData      []AccountData    `json:"accountData"`
	TransactionError interface{}      `json:"transactionError"`
	Instructions     []Instruction    `json:"instructions"`
	Events           interface{}      `json:"events"`
}

type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          int64  `json:"amount"`
}

type TokenTransfer struct {
	FromUserAccount  string  `json:"fromUserAccount"`
	ToUserAccount    string  `json:"toUserAccount"`
	FromTokenAccount string  `json:"fromTokenAccount"`
	ToTokenAccount   string  `json:"toTokenAccount"`
	TokenAmount      float64 `json:"tokenAmount"`
	Mint             string  `json:"mint"`
}

type AccountData struct {
	Account             string               `json:"account"`
	NativeBalanceChange int64                `json:"nativeBalanceChange"`
	TokenBalanceChanges []TokenBalanceChange `json:"tokenBalanceChanges"`
}

type TokenBalanceChange struct {
	UserAccount    string `json:"userAccount"`
	TokenAccount   string `json:"tokenAccount"`
	Mint           string `json:"mint"`
	RawTokenAmount struct {
		TokenAmount string `json:"tokenAmount"`
		Decimals    int    `json:"decimals"`
	} `json:"rawTokenAmount"`
}

type Instruction struct {
	Accounts          []string           `json:"accounts"`
	Data              string             `json:"data"`
	ProgramId         string             `json:"programId"`
	InnerInstructions []InnerInstruction `json:"innerInstructions"`
}

type InnerInstruction struct {
	Accounts  []string `json:"accounts"`
	Data      string   `json:"data"`
	ProgramId string   `json:"programId"`
}
