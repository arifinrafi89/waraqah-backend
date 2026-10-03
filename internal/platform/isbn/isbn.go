// Package isbn reads the ISBN on the back of a book (port of lib/features/scan/domain/entities/isbn.dart).
package isbn

import (
	"regexp"
	"strings"
)

var (
	spaceDash = regexp.MustCompile(`[\s-]`)
	re13      = regexp.MustCompile(`^\d{13}$`)
	re10      = regexp.MustCompile(`^\d{9}[\dX]$`)
)

// Normalize returns the ISBN-13 for raw (spaces and dashes are fine; an ISBN-10 is turned into
// its ISBN-13), or "" and false when it is not a valid ISBN.
func Normalize(raw string) (string, bool) {
	text := strings.ToUpper(spaceDash.ReplaceAllString(raw, ""))
	if re13.MatchString(text) {
		bookland := strings.HasPrefix(text, "978") || strings.HasPrefix(text, "979")
		if bookland && check13(text[:12]) == text[12] {
			return text, true
		}
		return "", false
	}
	if re10.MatchString(text) {
		if check10(text[:9]) != text[9] {
			return "", false
		}
		twelve := "978" + text[:9]
		return twelve + string(check13(twelve)), true
	}
	return "", false
}

func check13(twelve string) byte {
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(twelve[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return byte('0' + (10-sum%10)%10)
}

func check10(nine string) byte {
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(nine[i]-'0') * (10 - i)
	}
	c := (11 - sum%11) % 11
	if c == 10 {
		return 'X'
	}
	return byte('0' + c)
}
