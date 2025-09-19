package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ProductInfo struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Qty   int64  `json:"qty"`
}

type ProductClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProductClient(baseURL string, timeout time.Duration) *ProductClient {
	return &ProductClient{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *ProductClient) GetProductById(ctx context.Context, id uint64) (*ProductInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/products/%d", c.baseURL, id), nil)
	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product service returned status %d", resp.StatusCode)
	}
	var p ProductInfo
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return  nil, err
	}

	return &p, nil
}