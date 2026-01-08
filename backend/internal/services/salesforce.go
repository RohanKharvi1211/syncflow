package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SalesforceService struct {
	AccessToken string
	InstanceURL string
	Client      *http.Client
}

type SalesforceAuthResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

type SalesforceQueryResponse struct {
	TotalSize int                      `json:"totalSize"`
	Done      bool                     `json:"done"`
	Records   []map[string]interface{} `json:"records"`
}

func NewSalesforceService() *SalesforceService {
	return &SalesforceService{
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SalesforceService) Authenticate(config map[string]interface{}) error {
	// Use provided configuration
	clientID, _ := config["client_id"].(string)
	clientSecret, _ := config["client_secret"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	securityToken, _ := config["security_token"].(string)
	instanceURL, _ := config["instance_url"].(string)

	// Prepare authentication request
	authURL := fmt.Sprintf("%s/services/oauth2/token", instanceURL)
	
	data := map[string]string{
		"grant_type":    "password",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"username":      username,
		"password":      password + securityToken,
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var authResp SalesforceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return err
	}

	s.AccessToken = authResp.AccessToken
	s.InstanceURL = authResp.InstanceURL

	return nil
}

func (s *SalesforceService) QueryRecords(soql string) (*SalesforceQueryResponse, error) {
	if s.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/services/data/v58.0/query/?q=%s", s.InstanceURL, soql)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResp SalesforceQueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, err
	}

	return &queryResp, nil
}

func (s *SalesforceService) GetUpdatedRecords(recordType string, lastSyncTime time.Time) ([]map[string]interface{}, error) {
	soql := fmt.Sprintf("SELECT Id, Name, LastModifiedDate FROM %s WHERE LastModifiedDate > %s ORDER BY LastModifiedDate ASC",
		recordType, lastSyncTime.Format("2006-01-02T15:04:05.000Z"))

	response, err := s.QueryRecords(soql)
	if err != nil {
		return nil, err
	}

	return response.Records, nil
}

func (s *SalesforceService) GetRecordDetails(recordType, recordID string) (map[string]interface{}, error) {
	if s.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/services/data/v58.0/sobjects/%s/%s", s.InstanceURL, recordType, recordID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, err
	}

	return record, nil
}
