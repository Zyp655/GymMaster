package config

import "testing"

// SWAG-02: APP_ENV không phân biệt hoa thường khi xác định môi trường production.
func TestIsProduction_CaseInsensitive(t *testing.T) {
	for env, want := range map[string]bool{
		"production": true, "Production": true, "PRODUCTION": true, " prod ": true, "Prod": true,
		"development": false, "": false, "staging": false,
	} {
		if got := (&Config{Env: env}).IsProduction(); got != want {
			t.Errorf("IsProduction(%q) = %v, muốn %v", env, got, want)
		}
	}
}
