package models

import "time"

// Tally Customer structure
type TallyCustomer struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	Alias       string `json:"Alias"`
	Address     string `json:"Address"`
	City        string `json:"City"`
	State       string `json:"State"`
	Pincode     string `json:"Pincode"`
	Country     string `json:"Country"`
	Phone       string `json:"Phone"`
	Email       string `json:"Email"`
	Website     string `json:"Website"`
	CreatedDate time.Time `json:"CreatedDate"`
	ModifiedDate time.Time `json:"ModifiedDate"`
}

// Tally Item structure
type TallyItem struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	Alias       string `json:"Alias"`
	Description string `json:"Description"`
	Unit        string `json:"Unit"`
	Rate        float64 `json:"Rate"`
	TaxRate     float64 `json:"TaxRate"`
	CreatedDate time.Time `json:"CreatedDate"`
	ModifiedDate time.Time `json:"ModifiedDate"`
}

// Tally Voucher structure (for invoices)
type TallyVoucher struct {
	ID          string `json:"Id"`
	VoucherType string `json:"VoucherType"`
	Date        string `json:"Date"`
	PartyName   string `json:"PartyName"`
	Amount      float64 `json:"Amount"`
	Items       []TallyVoucherItem `json:"Items"`
	CreatedDate time.Time `json:"CreatedDate"`
	ModifiedDate time.Time `json:"ModifiedDate"`
}

type TallyVoucherItem struct {
	ItemName string  `json:"ItemName"`
	Quantity float64 `json:"Quantity"`
	Rate     float64 `json:"Rate"`
	Amount   float64 `json:"Amount"`
}

// Tally configuration
type TallyConfig struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	CompanyID    uint   `json:"company_id"`
	Company      Company `json:"company" gorm:"foreignKey:CompanyID"`
	ServerURL    string `json:"server_url"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	CompanyName  string `json:"company_name"`
	IsActive     bool   `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
