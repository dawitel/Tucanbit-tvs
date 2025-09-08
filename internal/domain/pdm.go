package domain

type PDMClientRequestType string
type PDMWebhookEventType string

const (
	PDMClientRequestTypeCreateWallet PDMClientRequestType = "pdm.createwallet"
	PDMClientRequestTypeGetWallet    PDMClientRequestType = "pdm.getwallet"

	PDMWebhookEventTypeTxVerify PDMWebhookEventType = "pdm.txverify"
)

type PDMWebhookRequest struct {
	EventType PDMWebhookEventType `json:"event_type"`
	Payload   map[string]string   `json:"payload"`
	Version   string              `json:"version"`
	Secret    string              `json:"secret"`
}
