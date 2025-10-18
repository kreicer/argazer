package helm

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewChecker(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	checker, err := NewChecker(logger)
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

func TestCompareVersions(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	checker, _ := NewChecker(logger)

	tests := []struct {
		v1     string
		v2     string
		expect int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.9.9", "2.0.0", -1},
	}

	for _, test := range tests {
		result := checker.compareVersions(test.v1, test.v2)
		if result != test.expect {
			t.Errorf("compareVersions(%s, %s) = %d, expected %d", test.v1, test.v2, result, test.expect)
		}
	}
}

func TestGetLatestVersion(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	checker, _ := NewChecker(logger)

	tests := []struct {
		versions []string
		expect   string
		hasError bool
	}{
		{[]string{"1.0.0", "1.0.1", "1.0.2"}, "1.0.2", false},
		{[]string{"2.0.0", "1.0.0", "1.5.0"}, "2.0.0", false},
		{[]string{"1.0.0"}, "1.0.0", false},
		{[]string{}, "", true},
	}

	for _, test := range tests {
		result, err := checker.getLatestVersion(test.versions)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for versions %v, got none", test.versions)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for versions %v: %v", test.versions, err)
			}
			if result != test.expect {
				t.Errorf("getLatestVersion(%v) = %s, expected %s", test.versions, result, test.expect)
			}
		}
	}
}
