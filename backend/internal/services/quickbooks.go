package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"syncflow-backend/internal/models"
	"time"
)

type QuickBooksService struct {
	AccessToken string
	RefreshToken string
	RealmID     string
	BaseURL     string
	Client      *http.Client
}

type QuickBooksAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	XRefreshTokenExpiresIn int `json:"x_refresh_token_expires_in"`
}

type QuickBooksQueryResponse struct {
	QueryResponse struct {
		Customer []models.QuickBooksCustomer `json:"Customer"`
		Item     []models.QuickBooksItem     `json:"Item"`
		Invoice  []models.QuickBooksInvoice  `json:"Invoice"`
	} `json:"QueryResponse"`
	Time string `json:"time"`
}

func NewQuickBooksService() *QuickBooksService {
	return &QuickBooksService{
		BaseURL: "https://sandbox-quickbooks.api.intuit.com",
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (q *QuickBooksService) Authenticate(config map[string]interface{}) error {
	// Use provided configuration
	q.AccessToken, _ = config["access_token"].(string)
	q.RefreshToken, _ = config["refresh_token"].(string)
	q.RealmID, _ = config["realm_id"].(string)

	return nil
}

func (q *QuickBooksService) RefreshAccessToken() error {
	// Implementation for refreshing access token
	// This would involve calling QuickBooks OAuth2 refresh endpoint
	return nil
}

func (q *QuickBooksService) QueryCustomers() ([]models.QuickBooksCustomer, error) {
	if q.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/v3/company/%s/query", q.BaseURL, q.RealmID)
	soql := "SELECT * FROM Customer"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+q.AccessToken)
	req.Header.Set("Accept", "application/json")
	q.setQueryParam(req, "query", soql)

	resp, err := q.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResp QuickBooksQueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, err
	}

	return queryResp.QueryResponse.Customer, nil
}

func (q *QuickBooksService) CreateCustomer(customerData map[string]interface{}) (*models.QuickBooksCustomer, error) {
	if q.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/v3/company/%s/customer", q.BaseURL, q.RealmID)
	
	jsonData, err := json.Marshal(customerData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+q.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := q.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		QueryResponse struct {
			Customer []models.QuickBooksCustomer `json:"Customer"`
		} `json:"QueryResponse"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.QueryResponse.Customer) > 0 {
		return &response.QueryResponse.Customer[0], nil
	}

	return nil, fmt.Errorf("no customer created")
}

func (q *QuickBooksService) UpdateCustomer(customerID string, customerData map[string]interface{}) (*models.QuickBooksCustomer, error) {
	if q.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/v3/company/%s/customer", q.BaseURL, q.RealmID)
	
	jsonData, err := json.Marshal(customerData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+q.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := q.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		QueryResponse struct {
			Customer []models.QuickBooksCustomer `json:"Customer"`
		} `json:"QueryResponse"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.QueryResponse.Customer) > 0 {
		return &response.QueryResponse.Customer[0], nil
	}

	return nil, fmt.Errorf("no customer updated")
}

func (q *QuickBooksService) setQueryParam(req *http.Request, key, value string) {
	query := req.URL.Query()
	query.Add(key, value)
	req.URL.RawQuery = query.Encode()
}
