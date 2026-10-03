package profile

import (
	"strings"
	"testing"
)

// Ported from test/profile_rules_test.dart.
func TestMobileAcceptsThreeForms(t *testing.T) {
	for _, in := range []string{"01712345678", "+8801712345678", "8801712345678", "017-1234 5678"} {
		if got, ok := Mobile(in); !ok || got != "01712345678" {
			t.Errorf("Mobile(%q) = %q %v", in, got, ok)
		}
	}
}

func TestMobileRefusesOthers(t *testing.T) {
	for _, in := range []string{"01212345678", "0171234567", "1712345678", ""} {
		if _, ok := Mobile(in); ok {
			t.Errorf("Mobile(%q) accepted", in)
		}
	}
}

func TestNameAndOptionalPhone(t *testing.T) {
	cases := []struct {
		name, phone string
		want        ProfileProblem
	}{
		{"Nadia", "", ""},
		{"Nadia", "+8801712345678", ""},
		{" N ", "", NameLength},
		{strings.Repeat("N", 61), "", NameLength},
		{"Nadia", "0121", PhoneBad},
	}
	for _, c := range cases {
		if got := CheckProfile(c.name, c.phone); got != c.want {
			t.Errorf("CheckProfile(%q, %q) = %q, want %q", c.name, c.phone, got, c.want)
		}
	}
}

// Ported from test/address_rules_test.dart.
var okAddress = Address{Label: "Home", Recipient: "Nadia", Phone: "+8801712345678", Line: "House 1, Road 2",
	Upazila: "Savar", District: "Dhaka", Division: "Dhaka"}

func TestCompleteAddressPassesAndIsTidied(t *testing.T) {
	if CheckAddress(okAddress) != "" {
		t.Error("complete address refused")
	}
	if TidyAddress(okAddress).Phone != "01712345678" {
		t.Error("phone not tidied")
	}
}

func TestAddressProblemsInEditorOrder(t *testing.T) {
	with := func(f func(*Address)) Address { a := okAddress; f(&a); return a }
	cases := []struct {
		a    Address
		want AddressProblem
	}{
		{with(func(a *Address) { a.Label = " " }), AddrLabel},
		{with(func(a *Address) { a.Recipient = "" }), AddrRecipient},
		{with(func(a *Address) { a.Line = "  " }), AddrLine},
		{with(func(a *Address) { a.Phone = "01212345678" }), AddrPhone},
		{with(func(a *Address) { a.Phone = "0171234567" }), AddrPhone},
		{with(func(a *Address) { a.Upazila = "" }), AddrPlace},
		{with(func(a *Address) { a.Division = "" }), AddrPlace},
	}
	for _, c := range cases {
		if got := CheckAddress(c.a); got != c.want {
			t.Errorf("CheckAddress = %q, want %q for %+v", got, c.want, c.a)
		}
	}
}
