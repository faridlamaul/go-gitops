package semver

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var strictVersion = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

// IsStrictVersion checks if a tag matches strict semver.
func IsStrictVersion(v string) bool {
	return strictVersion.MatchString(v)
}

// ParseVersionTag parses a git tag into a semver.Version.
func ParseVersionTag(name string) (*semver.Version, bool) {
	if !IsStrictVersion(name) {
		return nil, false
	}
	clean := strings.TrimPrefix(name, "v")
	v, err := semver.NewVersion(clean)
	if err != nil {
		return nil, false
	}
	return v, true
}

// TagItem represents a tag name and its parsed version.
type TagItem struct {
	Name    string
	Version *semver.Version
	CommitSHA string
}

// LatestStrictTag returns the tag with the highest semantic version from a list of tag names.
func LatestStrictTag(tagNames []string) (string, *semver.Version) {
	var parsed []TagItem
	for _, name := range tagNames {
		v, ok := ParseVersionTag(name)
		if ok {
			parsed = append(parsed, TagItem{Name: name, Version: v})
		}
	}
	if len(parsed) == 0 {
		return "", nil
	}

	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].Version.GreaterThan(parsed[j].Version)
	})

	return parsed[0].Name, parsed[0].Version
}

// NextVersion calculates the next version string based on bump type (major, minor, patch).
// Preserves the "v" prefix if present in the base tag.
func NextVersion(baseTag string, bump string) (string, error) {
	hasV := strings.HasPrefix(baseTag, "v")
	var current *semver.Version
	var err error

	if baseTag == "" {
		current, _ = semver.NewVersion("0.0.0")
		hasV = true
	} else {
		clean := strings.TrimPrefix(baseTag, "v")
		current, err = semver.NewVersion(clean)
		if err != nil {
			return "", fmt.Errorf("invalid base version %q: %w", baseTag, err)
		}
	}

	var next semver.Version
	switch strings.ToLower(bump) {
	case "major":
		next = current.IncMajor()
	case "minor":
		next = current.IncMinor()
	case "patch":
		next = current.IncPatch()
	default:
		return "", fmt.Errorf("unknown bump type: %s (must be major, minor, or patch)", bump)
	}

	if hasV {
		return "v" + next.String(), nil
	}
	return next.String(), nil
}

// AutoBumpType inspects change types to determine if bump should be major, minor, or patch.
func AutoBumpType(hasBreaking bool, hasFeatures bool) string {
	if hasBreaking {
		return "major"
	}
	if hasFeatures {
		return "minor"
	}
	return "patch"
}
