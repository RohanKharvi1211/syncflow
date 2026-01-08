package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/models"
	"time"
)

type TallyService struct {
	Config     models.TallyConfig
	Client     *http.Client
}

type TallyResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewTallyService(companyID uint) (*TallyService, error) {
	var tallyConfig models.TallyConfig
	result := config.DB.Where("company_id = ? AND is_active = ?", companyID, true).First(&tallyConfig)
	
	if result.Error != nil {
		return nil, fmt.Errorf("tally configuration not found for company %d", companyID)
	}

	return &TallyService{
		Config: tallyConfig,
		Client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *TallyService) CreateCustomer(customerData map[string]interface{}) (*models.TallyCustomer, error) {
	url := fmt.Sprintf("http://%s:%d/api/customers", t.Config.ServerURL, t.Config.Port)
	
	jsonData, err := json.Marshal(customerData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.getAuthToken())

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TallyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("tally error: %s", response.Message)
	}

	// Convert response data to TallyCustomer
	customerJSON, _ := json.Marshal(response.Data)
	var customer models.TallyCustomer
	if err := json.Unmarshal(customerJSON, &customer); err != nil {
		return nil, err
	}

	return &customer, nil
}

func (t *TallyService) UpdateCustomer(customerID string, customerData map[string]interface{}) (*models.TallyCustomer, error) {
	url := fmt.Sprintf("http://%s:%d/api/customers/%s", t.Config.ServerURL, t.Config.Port, customerID)
	
	jsonData, err := json.Marshal(customerData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.getAuthToken())

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TallyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("tally error: %s", response.Message)
	}

	// Convert response data to TallyCustomer
	customerJSON, _ := json.Marshal(response.Data)
	var customer models.TallyCustomer
	if err := json.Unmarshal(customerJSON, &customer); err != nil {
		return nil, err
	}

	return &customer, nil
}

func (t *TallyService) CreateItem(itemData map[string]interface{}) (*models.TallyItem, error) {
	url := fmt.Sprintf("http://%s:%d/api/items", t.Config.ServerURL, t.Config.Port)
	
	jsonData, err := json.Marshal(itemData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.getAuthToken())

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TallyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("tally error: %s", response.Message)
	}

	// Convert response data to TallyItem
	itemJSON, _ := json.Marshal(response.Data)
	var item models.TallyItem
	if err := json.Unmarshal(itemJSON, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (t *TallyService) CreateVoucher(voucherData map[string]interface{}) (*models.TallyVoucher, error) {
	url := fmt.Sprintf("http://%s:%d/api/vouchers", t.Config.ServerURL, t.Config.Port)
	
	jsonData, err := json.Marshal(voucherData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.getAuthToken())

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response TallyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("tally error: %s", response.Message)
	}

	// Convert response data to TallyVoucher
	voucherJSON, _ := json.Marshal(response.Data)
	var voucher models.TallyVoucher
	if err := json.Unmarshal(voucherJSON, &voucher); err != nil {
		return nil, err
	}

	return &voucher, nil
}

func (t *TallyService) getAuthToken() string {
	// In a real implementation, you'd implement proper authentication
	// For now, return a placeholder
	return "placeholder-token"
}

func (t *TallyService) TestConnection() error {
	url := fmt.Sprintf("http://%s:%d/api/health", t.Config.ServerURL, t.Config.Port)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+t.getAuthToken())

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("tally connection test failed with status: %d", resp.StatusCode)
	}

	return nil
}
