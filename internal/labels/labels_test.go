package labels

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFeatures(t *testing.T) {
	cases := []struct {
		name        string
		features    string
		expected    []Feature
		expectedErr error
	}{
		{
			name:        "empty features",
			features:    "",
			expected:    nil,
			expectedErr: nil,
		},
		{
			name:        "ptr feature",
			features:    "ptr",
			expected:    []Feature{FeatureFlagPtr},
			expectedErr: nil,
		},
		{
			name:        "arr feature",
			features:    "arr",
			expected:    []Feature{FeatureFlagArr},
			expectedErr: nil,
		},
		{
			name:        "opt feature",
			features:    "opt",
			expected:    []Feature{FeatureFlagOpt},
			expectedErr: nil,
		},
		{
			name:        "multiple features",
			features:    "opt,arr,ptr",
			expected:    []Feature{FeatureFlagOpt, FeatureFlagArr, FeatureFlagPtr},
			expectedErr: nil,
		},
		{
			name:        "multiple features",
			features:    "opt,arr,ptr",
			expected:    []Feature{FeatureFlagOpt, FeatureFlagArr, FeatureFlagPtr},
			expectedErr: nil,
		},
		{
			name:        "spaces around",
			features:    " opt,arr,ptr",
			expected:    []Feature{FeatureFlagOpt, FeatureFlagArr, FeatureFlagPtr},
			expectedErr: nil,
		},
		{
			name:        "trailing comma",
			features:    "opt,arr,ptr,,",
			expected:    []Feature{FeatureFlagOpt, FeatureFlagArr, FeatureFlagPtr},
			expectedErr: nil,
		},
		{
			name:        "invalid feature",
			features:    "opt,arr,some",
			expected:    nil,
			expectedErr: fmt.Errorf("unable to parse feature='%s'", "some"),
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual, actualErr := ParseFeatures(c.features)
			require.Equal(t, c.expectedErr, actualErr, "errors are not equal")
			assert.Equal(t, c.expected, actual, "values are not equal")
		})
	}
}
