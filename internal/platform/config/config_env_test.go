package config

import (
	"os"
	"reflect"
	"regexp"
	"testing"
)

// Every Config field must have an entry in .env.example and vice versa (BACKEND_PLAN.md §18).
func TestEnvExampleMatchesConfig(t *testing.T) {
	data, err := os.ReadFile("../../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	inExample := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^([A-Z][A-Z0-9_]*)=`).FindAllStringSubmatch(string(data), -1) {
		inExample[m[1]] = true
	}
	inStruct := map[string]bool{}
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		inStruct[typ.Field(i).Tag.Get("env")] = true
	}
	for k := range inStruct {
		if !inExample[k] {
			t.Errorf("Config field %s has no entry in .env.example", k)
		}
	}
	for k := range inExample {
		if !inStruct[k] {
			t.Errorf(".env.example entry %s has no Config field", k)
		}
	}
}
