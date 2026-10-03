package profile

import "strings"

// AddressProblem says why an address can not be saved ("" means none).
type AddressProblem string

// The address problems, named as in the Dart enum, in the editor's field order.
const (
	AddrLabel     AddressProblem = "label"
	AddrRecipient AddressProblem = "recipient"
	AddrPhone     AddressProblem = "phone"
	AddrLine      AddressProblem = "line"
	AddrPlace     AddressProblem = "place"
)

// Address is a saved address as the app sends and receives it (SavedAddressModel).
type Address struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line      string `json:"line"`
	Upazila   string `json:"upazila"`
	District  string `json:"district"`
	Division  string `json:"division"`
	IsDefault bool   `json:"isDefault"`
}

// CheckAddress is AddressRules.check: the first problem, in the editor's field order.
func CheckAddress(a Address) AddressProblem {
	if strings.TrimSpace(a.Label) == "" {
		return AddrLabel
	}
	if strings.TrimSpace(a.Recipient) == "" {
		return AddrRecipient
	}
	if _, ok := Mobile(a.Phone); !ok {
		return AddrPhone
	}
	if strings.TrimSpace(a.Line) == "" {
		return AddrLine
	}
	if a.Division == "" || a.District == "" || a.Upazila == "" {
		return AddrPlace
	}
	return ""
}

// TidyAddress is AddressRules.tidy: trimmed fields and the phone as 01.... Call after CheckAddress.
func TidyAddress(a Address) Address {
	a.Label = strings.TrimSpace(a.Label)
	a.Recipient = strings.TrimSpace(a.Recipient)
	a.Line = strings.TrimSpace(a.Line)
	if m, ok := Mobile(a.Phone); ok {
		a.Phone = m
	}
	return a
}
