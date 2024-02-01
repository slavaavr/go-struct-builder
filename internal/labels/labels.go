package labels

import (
	"fmt"
	"strings"
)

type Feature string

func (t Feature) String() string {
	return string(t)
}

const (
	Gosb        = "gosb"
	GenerateCmd = "go:generate"

	StructTagRequired = "required"
	StructTagOptional = "optional"

	FeatureFlagPtr Feature = "ptr"
	FeatureFlagArr Feature = "arr"
	FeatureFlagOpt Feature = "opt"
)

func ParseFeatures(s string) ([]Feature, error) {
	if s == "" {
		return nil, nil
	}

	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ",")
	ss := strings.Split(s, ",")
	res := make([]Feature, 0, len(ss))

	for _, s := range ss {
		switch s {
		case FeatureFlagPtr.String():
			res = append(res, FeatureFlagPtr)

		case FeatureFlagArr.String():
			res = append(res, FeatureFlagArr)

		case FeatureFlagOpt.String():
			res = append(res, FeatureFlagOpt)

		default:
			return nil, fmt.Errorf("unable to parse feature='%s'", s)
		}
	}

	return res, nil
}
