package models

import "time"

// Salesforce Account structure
type SalesforceAccount struct {
	ID                string    `json:"Id"`
	Name              string    `json:"Name"`
	Type              string    `json:"Type"`
	Industry          string    `json:"Industry"`
	Phone             string    `json:"Phone"`
	Website           string    `json:"Website"`
	BillingStreet     string    `json:"BillingStreet"`
	BillingCity       string    `json:"BillingCity"`
	BillingState      string    `json:"BillingState"`
	BillingPostalCode string    `json:"BillingPostalCode"`
	BillingCountry    string    `json:"BillingCountry"`
	CreatedDate       time.Time `json:"CreatedDate"`
	LastModifiedDate  time.Time `json:"LastModifiedDate"`
}

// Salesforce Contact structure
type SalesforceContact struct {
	ID                string    `json:"Id"`
	FirstName         string    `json:"FirstName"`
	LastName          string    `json:"LastName"`
	Email             string    `json:"Email"`
	Phone             string    `json:"Phone"`
	Title             string    `json:"Title"`
	AccountID         string    `json:"AccountId"`
	MailingStreet     string    `json:"MailingStreet"`
	MailingCity       string    `json:"MailingCity"`
	MailingState      string    `json:"MailingState"`
	MailingPostalCode string    `json:"MailingPostalCode"`
	MailingCountry    string    `json:"MailingCountry"`
	CreatedDate       time.Time `json:"CreatedDate"`
	LastModifiedDate  time.Time `json:"LastModifiedDate"`
}

// Salesforce Opportunity structure
type SalesforceOpportunity struct {
	ID                string    `json:"Id"`
	Name              string    `json:"Name"`
	AccountID         string    `json:"AccountId"`
	CloseDate         time.Time `json:"CloseDate"`
	StageName         string    `json:"StageName"`
	Amount            float64   `json:"Amount"`
	Probability       float64   `json:"Probability"`
	Type              string    `json:"Type"`
	LeadSource        string    `json:"LeadSource"`
	Description       string    `json:"Description"`
	CreatedDate       time.Time `json:"CreatedDate"`
	LastModifiedDate  time.Time `json:"LastModifiedDate"`
}
