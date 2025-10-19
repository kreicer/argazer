package helm

import (
	"testing"

	"argazer/internal/auth"

	"github.com/sirupsen/logrus"
)

func TestNewChecker(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, err := NewChecker(authProvider, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if checker == nil {
		t.Fatal("Expected checker to be initialized")
	}

	if checker.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestFindLatestSemver(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name     string
		versions []string
		expect   string
		hasError bool
	}{
		{"simple versions", []string{"1.0.0", "1.0.1", "1.0.2"}, "1.0.2", false},
		{"mixed order", []string{"2.0.0", "1.0.0", "1.5.0"}, "2.0.0", false},
		{"single version", []string{"1.0.0"}, "1.0.0", false},
		{"empty list", []string{}, "", true},
		{"with v prefix", []string{"v1.0.0", "v1.0.1", "v2.0.0"}, "v2.0.0", false},
		{"mixed valid and invalid", []string{"1.0.0", "invalid", "2.0.0", "latest"}, "2.0.0", false},
		{"all invalid", []string{"invalid", "latest", "dev"}, "", true},
		{"pre-release versions", []string{"1.0.0", "1.0.0-alpha", "1.0.0-beta", "2.0.0"}, "2.0.0", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := findLatestSemver(test.versions, logger)
			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for versions %v, got none", test.versions)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for versions %v: %v", test.versions, err)
				}
				if result != test.expect {
					t.Errorf("findLatestSemver(%v) = %s, expected %s", test.versions, result, test.expect)
				}
			}
		})
	}
}
