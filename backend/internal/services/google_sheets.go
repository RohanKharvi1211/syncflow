package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoogleSheetsService struct {
	AccessToken  string
	RefreshToken string
	Client       *http.Client
}

func NewGoogleSheetsService() *GoogleSheetsService {
	return &GoogleSheetsService{
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate sets the access token from connection
func (g *GoogleSheetsService) Authenticate(accessToken, refreshToken string) {
	g.AccessToken = accessToken
	g.RefreshToken = refreshToken
}

// GetSheetHeaders fetches the first row (headers) from a Google Sheet
func (g *GoogleSheetsService) GetSheetHeaders(spreadsheetID string, range_ string) ([]string, error) {
	if g.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	// Default to first sheet if range not specified
	// Use A1:Z1 format (without sheet name) - Google Sheets API will use the first sheet by default
	if range_ == "" {
		range_ = "A1:Z1"
	}
	
	// Remove sheet name prefix if present (e.g., "Sheet1!A1:Z1" -> "A1:Z1")
	// Google Sheets API v4 uses the range directly, and we can specify sheet in the range
	// But for simplicity, let's use just the range without sheet name for now
	if strings.Contains(range_, "!") {
		parts := strings.Split(range_, "!")
		if len(parts) > 1 {
			range_ = parts[1] // Use the range part after "!"
		}
	}

	// Build URL for Google Sheets API v4
	// Format: https://sheets.googleapis.com/v4/spreadsheets/{spreadsheetId}/values/{range}
	// Range should be URL encoded
	encodedRange := url.QueryEscape(range_)
	baseURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s", spreadsheetID, encodedRange)
	params := url.Values{}
	params.Set("majorDimension", "ROWS")
	
	fullURL := baseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("Google Sheets API error: %s (code: %d, status: %s)",
				errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
		}
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return nil, fmt.Errorf("Google Sheets API returned status %d. Response: %s", resp.StatusCode, bodyStr)
	}

	var response struct {
		Values [][]interface{} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Values) == 0 {
		return []string{}, nil
	}

	// Convert first row to string array
	headers := make([]string, 0, len(response.Values[0]))
	for _, val := range response.Values[0] {
		if str, ok := val.(string); ok {
			headers = append(headers, str)
		} else {
			headers = append(headers, fmt.Sprintf("%v", val))
		}
	}

	return headers, nil
}

// GetSheetData fetches all rows from a Google Sheet within a range
func (g *GoogleSheetsService) GetSheetData(spreadsheetID string, sheetName string, startRow int) ([][]string, error) {
	if g.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	// Build range.
	// IMPORTANT:
	// We've seen Google Sheets API return "Unable to parse range: Sheet1!A:Z"
	// for some sheets (sheet name mismatches, special chars, etc.).
	// To make this robust, we intentionally DO NOT prefix with sheet name here
	// and rely on the first sheet in the spreadsheet, using a simple "A:Z" range.
	//
	// Later, if we need exact-sheet selection, we can extend this to:
	//  - list sheets
	//  - validate sheetName
	//  - and only then include it in the range.
	range_ := "A:Z" // Default to columns A–Z, all rows on the first sheet

	// Build URL for Google Sheets API v4
	encodedRange := url.QueryEscape(range_)
	baseURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s", spreadsheetID, encodedRange)
	params := url.Values{}
	params.Set("majorDimension", "ROWS")
	
	fullURL := baseURL + "?" + params.Encode()

	fmt.Printf("DEBUG: Fetching Google Sheet data, spreadsheetID: %s, sheetName: %s, range: %s\n", spreadsheetID, sheetName, range_)
	fmt.Printf("DEBUG: Full URL: %s\n", fullURL)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: Failed to make request to Google Sheets API: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: Google Sheets API response status: %d %s\n", resp.StatusCode, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			fmt.Printf("ERROR: Google Sheets API error: %+v\n", errorResp)
			return nil, fmt.Errorf("Google Sheets API error: %s (code: %d, status: %s)",
				errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
		}
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		fmt.Printf("ERROR: Google Sheets API returned status %d. Response: %s\n", resp.StatusCode, bodyStr)
		return nil, fmt.Errorf("Google Sheets API returned status %d. Response: %s", resp.StatusCode, bodyStr)
	}

	var response struct {
		Values [][]interface{} `json:"values"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("ERROR: Failed to parse Google Sheets API response: %v\n", err)
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Values) == 0 {
		fmt.Printf("DEBUG: Google Sheet is empty\n")
		return [][]string{}, nil
	}

	// Convert all rows to string arrays
	rows := make([][]string, 0, len(response.Values))
	for i, row := range response.Values {
		stringRow := make([]string, 0, len(row))
		for _, val := range row {
			if str, ok := val.(string); ok {
				stringRow = append(stringRow, str)
			} else if val == nil {
				stringRow = append(stringRow, "")
			} else {
				stringRow = append(stringRow, fmt.Sprintf("%v", val))
			}
		}
		rows = append(rows, stringRow)
		
		// Log first few rows for debugging
		if i < 3 {
			fmt.Printf("DEBUG: Row %d: %v\n", i, stringRow)
		}
	}

	fmt.Printf("DEBUG: Successfully fetched %d rows from Google Sheet\n", len(rows))
	return rows, nil
}

