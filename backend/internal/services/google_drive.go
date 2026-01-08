package services

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type GoogleDriveService struct {
	AccessToken  string
	RefreshToken string
	Client       *http.Client
}

type GoogleDriveFile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MimeType     string    `json:"mimeType"`
	ModifiedTime time.Time `json:"modifiedTime"`
	Size         string    `json:"size"`
}

type GoogleDriveListResponse struct {
	Files []GoogleDriveFile `json:"files"`
}

func NewGoogleDriveService() *GoogleDriveService {
	return &GoogleDriveService{
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate sets the access token from connection
func (g *GoogleDriveService) Authenticate(accessToken, refreshToken string) {
	g.AccessToken = accessToken
	g.RefreshToken = refreshToken
}

// RefreshAccessToken refreshes the access token using refresh token
func (g *GoogleDriveService) RefreshAccessToken(clientID, clientSecret string) error {
	// Implementation for refreshing Google OAuth token
	// This would call Google's token endpoint
	url := "https://oauth2.googleapis.com/token"

	data := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": g.RefreshToken,
		"grant_type":    "refresh_token",
	}

	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	g.AccessToken = tokenResp.AccessToken
	return nil
}

// GetFileMetadata gets metadata for a specific file
func (g *GoogleDriveService) GetFileMetadata(fileID string) (*GoogleDriveFile, error) {
	if g.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?fields=id,name,mimeType,modifiedTime,size", fileID)

	req, err := http.NewRequest("GET", url, nil)
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
		return nil, err
	}

	var file GoogleDriveFile
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

// DownloadFile downloads a file from Google Drive
func (g *GoogleDriveService) DownloadFile(fileID string) ([]byte, error) {
	if g.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)
	fmt.Printf("DEBUG: Downloading file from Google Drive, fileID: %s, URL: %s\n", fileID, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: Failed to make request to Google Drive: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: Google Drive API response status: %d %s\n", resp.StatusCode, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: Failed to read response body: %v\n", err)
		return nil, err
	}

	fmt.Printf("DEBUG: Response body length: %d bytes\n", len(body))

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse as JSON error
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			fmt.Printf("ERROR: Google Drive API error response: %+v\n", errorResp)
			return nil, fmt.Errorf("Google Drive API error: %s (code: %d, status: %s)",
				errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
		}
		// If not JSON, log the raw response
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		fmt.Printf("ERROR: Google Drive API returned status %d. Response: %s\n", resp.StatusCode, bodyStr)
		return nil, fmt.Errorf("Google Drive API returned status %d: %s", resp.StatusCode, bodyStr)
	}

	// Log first 200 characters of response for debugging
	if len(body) > 0 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("DEBUG: Response preview (first 200 chars): %q\n", preview)
	}

	return body, nil
}

// ReadCSVFromDrive reads a CSV file from Google Drive and returns rows
func (g *GoogleDriveService) ReadCSVFromDrive(fileID string, sheetName string) ([][]string, error) {
	// Download the file
	fileData, err := g.DownloadFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}

	// Log the raw response for debugging
	fmt.Printf("DEBUG: Downloaded file data length: %d bytes\n", len(fileData))
	if len(fileData) > 0 {
		// Log first 500 characters to see what we got
		preview := string(fileData)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("DEBUG: File data preview (first 500 chars): %q\n", preview)
	}

	// Check if the response is actually JSON (error response)
	if len(fileData) > 0 {
		trimmed := bytes.TrimSpace(fileData)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			// Try to parse as JSON to see if it's an error
			var jsonData map[string]interface{}
			if err := json.Unmarshal(fileData, &jsonData); err == nil {
				fmt.Printf("DEBUG: Response appears to be JSON, not CSV. JSON content: %+v\n", jsonData)
				// Check if it's an error response
				if errorData, ok := jsonData["error"].(map[string]interface{}); ok {
					errorMsg := fmt.Sprintf("%v", errorData)
					return nil, fmt.Errorf("Google Drive API returned error response instead of CSV: %s", errorMsg)
				}
				return nil, fmt.Errorf("Google Drive returned JSON instead of CSV data. This might be an error response or wrong file format")
			}
		}
	}

	// Parse CSV line by line to capture problematic rows
	reader := csv.NewReader(bytes.NewReader(fileData))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true // Allow unquoted quotes in quoted fields
	reader.ReuseRecord = true

	var records [][]string
	lineNum := 0

	for {
		lineNum++
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			// Try to get the raw line content for debugging
			lines := bytes.Split(fileData, []byte("\n"))
			var rawLine string
			if lineNum-1 < len(lines) {
				rawLine = string(lines[lineNum-1])
				// Limit line length for error message
				if len(rawLine) > 500 {
					rawLine = rawLine[:500] + "..."
				}
			}

			// Log the error with full context
			fmt.Printf("ERROR: CSV parsing failed at line %d: %v\n", lineNum, err)
			fmt.Printf("ERROR: Problematic row content: %q\n", rawLine)
			fmt.Printf("ERROR: Full file data (first 1000 chars): %q\n", string(fileData[:min(len(fileData), 1000)]))

			// Include the problematic row in the error message
			if rawLine != "" {
				return nil, fmt.Errorf("failed to parse CSV: %v (line %d, row: %q)", err, lineNum, rawLine)
			}
			return nil, fmt.Errorf("failed to parse CSV: %v (line %d)", err, lineNum)
		}

		// Make a copy of the record since ReuseRecord is enabled
		recordCopy := make([]string, len(record))
		copy(recordCopy, record)
		records = append(records, recordCopy)
	}

	if len(records) == 0 {
		fmt.Printf("WARNING: CSV file is empty after parsing\n")
		return nil, fmt.Errorf("CSV file is empty")
	}

	fmt.Printf("DEBUG: Successfully parsed %d rows from CSV\n", len(records))
	return records, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ReadExcelFromDrive reads an Excel file from Google Drive (would need excelize library)
// For now, we'll focus on CSV support
func (g *GoogleDriveService) ReadExcelFromDrive(fileID string, sheetName string) ([][]string, error) {
	// This would require the excelize library
	// For now, return error indicating it needs implementation
	return nil, fmt.Errorf("Excel support not yet implemented")
}

// CheckFileModified checks if file has been modified since last sync
func (g *GoogleDriveService) CheckFileModified(fileID string, lastSyncTime time.Time) (bool, error) {
	metadata, err := g.GetFileMetadata(fileID)
	if err != nil {
		return false, err
	}

	return metadata.ModifiedTime.After(lastSyncTime), nil
}

// ListGoogleSheets lists all Google Sheets files from user's Drive
func (g *GoogleDriveService) ListGoogleSheets() ([]GoogleDriveFile, error) {
	if g.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	// Build URL with properly encoded query parameters
	// Google Drive API query syntax: mimeType='application/vnd.google-apps.spreadsheet'
	baseURL := "https://www.googleapis.com/drive/v3/files"
	params := url.Values{}
	// Use proper query format for Google Drive API (single quotes are part of the query syntax)
	params.Set("q", "mimeType='application/vnd.google-apps.spreadsheet'")
	params.Set("fields", "files(id,name,mimeType,modifiedTime,size)")
	params.Set("orderBy", "modifiedTime desc")
	params.Set("pageSize", "100") // Limit results

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

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse as JSON error first
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("Google Drive API error: %s (code: %d, status: %s)",
				errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
		}
		// If not JSON, return the raw response (might be HTML error page)
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return nil, fmt.Errorf("Google Drive API returned status %d. Response: %s", resp.StatusCode, bodyStr)
	}

	var listResp GoogleDriveListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return listResp.Files, nil
}
