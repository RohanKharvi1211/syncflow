package models

import "time"

// QuickBooks Customer structure
type QuickBooksCustomer struct {
	ID          string `json:"Id"`
	SyncToken   string `json:"SyncToken"`
	CompanyName string `json:"CompanyName"`
	DisplayName string `json:"DisplayName"`
	PrimaryPhone struct {
		FreeFormNumber string `json:"FreeFormNumber"`
	} `json:"PrimaryPhone"`
	WebAddr struct {
		URI string `json:"URI"`
	} `json:"WebAddr"`
	BillAddr struct {
		Line1                  string `json:"Line1"`
		City                   string `json:"City"`
		CountrySubDivisionCode string `json:"CountrySubDivisionCode"`
		PostalCode             string `json:"PostalCode"`
		Country                string `json:"Country"`
	} `json:"BillAddr"`
	MetaData struct {
		CreateTime      time.Time `json:"CreateTime"`
		LastUpdatedTime time.Time `json:"LastUpdatedTime"`
	} `json:"MetaData"`
}

// QuickBooks Item structure (for products/services)
type QuickBooksItem struct {
	ID          string `json:"Id"`
	SyncToken   string `json:"SyncToken"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Type        string `json:"Type"`
	UnitPrice   float64 `json:"UnitPrice"`
	IncomeAccountRef struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	} `json:"IncomeAccountRef"`
	MetaData struct {
		CreateTime      time.Time `json:"CreateTime"`
		LastUpdatedTime time.Time `json:"LastUpdatedTime"`
	} `json:"MetaData"`
}

// QuickBooks Invoice structure
type QuickBooksInvoice struct {
	ID          string `json:"Id"`
	SyncToken   string `json:"SyncToken"`
	DocNumber   string `json:"DocNumber"`
	TxnDate     string `json:"TxnDate"`
	CustomerRef struct {
		Value string `json:"value"`
		Name  string `json:"name"`
	} `json:"CustomerRef"`
	Line []struct {
		Amount     float64 `json:"Amount"`
		DetailType string  `json:"DetailType"`
		SalesItemLineDetail struct {
			ItemRef struct {
				Value string `json:"value"`
				Name  string `json:"name"`
			} `json:"ItemRef"`
			Qty float64 `json:"Qty"`
		} `json:"SalesItemLineDetail"`
	} `json:"Line"`
	TotalAmt    float64 `json:"TotalAmt"`
	Balance     float64 `json:"Balance"`
	MetaData    struct {
		CreateTime      time.Time `json:"CreateTime"`
		LastUpdatedTime time.Time `json:"LastUpdatedTime"`
	} `json:"MetaData"`
}
