package rpc

import (
	"net/http"
	"time"
)

type AlchemyClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewAlchemyClient(baseURL, apiKey string) *AlchemyClient {
	return &AlchemyClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}
