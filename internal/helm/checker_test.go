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

func TestFindLatestSemverWithConstraint(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name                      string
		versions                  []string
		currentVersion            string
		constraint                string
		expectedLatest            string
		expectedLatestAll         string
		expectedOutsideConstraint bool
		hasError                  bool
	}{
		{
			name:                      "major constraint - all versions",
			versions:                  []string{"1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			currentVersion:            "1.2.0",
			constraint:                "major",
			expectedLatest:            "2.1.0",
			expectedLatestAll:         "2.1.0",
			expectedOutsideConstraint: false,
			hasError:                  false,
		},
		{
			name:                      "minor constraint - same major only",
			versions:                  []string{"1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			currentVersion:            "1.2.0",
			constraint:                "minor",
			expectedLatest:            "1.5.0",
			expectedLatestAll:         "2.1.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:                      "patch constraint - same major.minor only",
			versions:                  []string{"1.2.0", "1.2.5", "1.3.0", "2.0.0"},
			currentVersion:            "1.2.3",
			constraint:                "patch",
			expectedLatest:            "1.2.5",
			expectedLatestAll:         "2.0.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:                      "patch constraint - no updates in constraint",
			versions:                  []string{"1.2.0", "1.2.1", "1.3.0", "2.0.0"},
			currentVersion:            "1.2.3",
			constraint:                "patch",
			expectedLatest:            "1.2.3",
			expectedLatestAll:         "2.0.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:                      "minor constraint - no updates in constraint",
			versions:                  []string{"1.0.0", "1.1.0", "2.0.0", "3.0.0"},
			currentVersion:            "1.5.0",
			constraint:                "minor",
			expectedLatest:            "1.5.0",
			expectedLatestAll:         "3.0.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:                      "empty constraint defaults to major",
			versions:                  []string{"1.0.0", "2.0.0", "3.0.0"},
			currentVersion:            "1.0.0",
			constraint:                "",
			expectedLatest:            "3.0.0",
			expectedLatestAll:         "3.0.0",
			expectedOutsideConstraint: false,
			hasError:                  false,
		},
		{
			name:                      "invalid current version falls back to major",
			versions:                  []string{"1.0.0", "2.0.0"},
			currentVersion:            "invalid",
			constraint:                "minor",
			expectedLatest:            "2.0.0",
			expectedLatestAll:         "2.0.0",
			expectedOutsideConstraint: false,
			hasError:                  false,
		},
		{
			name:                      "with v prefix",
			versions:                  []string{"v1.0.0", "v1.5.0", "v2.0.0"},
			currentVersion:            "v1.2.0",
			constraint:                "minor",
			expectedLatest:            "v1.5.0",
			expectedLatestAll:         "v2.0.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:                      "mixed valid and invalid versions",
			versions:                  []string{"1.0.0", "invalid", "1.5.0", "latest", "2.0.0"},
			currentVersion:            "1.2.0",
			constraint:                "minor",
			expectedLatest:            "1.5.0",
			expectedLatestAll:         "2.0.0",
			expectedOutsideConstraint: true,
			hasError:                  false,
		},
		{
			name:           "empty versions list",
			versions:       []string{},
			currentVersion: "1.0.0",
			constraint:     "major",
			hasError:       true,
		},
		{
			name:           "all invalid versions",
			versions:       []string{"invalid", "latest", "dev"},
			currentVersion: "1.0.0",
			constraint:     "major",
			hasError:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := findLatestSemverWithConstraint(test.versions, test.currentVersion, test.constraint, logger)

			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for versions %v with constraint %s, got none", test.versions, test.constraint)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.LatestVersion != test.expectedLatest {
				t.Errorf("LatestVersion = %s, expected %s", result.LatestVersion, test.expectedLatest)
			}

			if result.LatestVersionAll != test.expectedLatestAll {
				t.Errorf("LatestVersionAll = %s, expected %s", result.LatestVersionAll, test.expectedLatestAll)
			}

			if result.HasUpdateOutsideConstraint != test.expectedOutsideConstraint {
				t.Errorf("HasUpdateOutsideConstraint = %v, expected %v", result.HasUpdateOutsideConstraint, test.expectedOutsideConstraint)
			}
		})
	}
}
