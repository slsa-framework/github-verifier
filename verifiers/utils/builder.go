package utils

import (
	"fmt"
	"strings"

	serrors "github.com/slsa-framework/slsa-verifier/v2/errors"
)

type TrustedBuilderID struct {
	name, version string
}

// TrustedBuilderIDNew creates a new BuilderID structure.
func TrustedBuilderIDNew(builderID string) (*TrustedBuilderID, error) {
	name, version, err := ParseBuilderID(builderID, true)
	if err != nil {
		return nil, err
	}

	return &TrustedBuilderID{
		name:    name,
		version: version,
	}, nil
}

// Matches matches the builderID string against the reference builderID.
// If the builderID contains a semver, the full builderID must match.
// Otherwise, only the name needs to match.
// `allowRef: true` indicates that the matching need not be an eaxct
// match. In this case, if the BuilderID version is a GitHub ref
// `refs/tags/name`, we will consider it equal to user-provided
// builderID `name`.
func (b *TrustedBuilderID) Matches(builderID string, allowRef bool) error {
	name, version, err := ParseBuilderID(builderID, false)
	if err != nil {
		return err
	}

	if name != b.name {
		return fmt.Errorf("%w: expected name '%s', got '%s'", serrors.ErrorMismatchBuilderID,
			name, b.name)
	}

	if version != "" && version != b.version {
		// If allowRef is true, try the long version `refs/tags/<name>` match.
		if allowRef &&
			"refs/tags/"+version == b.version {
			return nil
		}
		return fmt.Errorf("%w: expected version '%s', got '%s'", serrors.ErrorMismatchBuilderID,
			version, b.version)
	}

	return nil
}

func (b *TrustedBuilderID) Name() string {
	return b.name
}

func (b *TrustedBuilderID) Version() string {
	return b.version
}

func (b *TrustedBuilderID) String() string {
	return fmt.Sprintf("%s@%s", b.name, b.version)
}

func ParseBuilderID(id string, needVersion bool) (string, string, error) {
	parts := strings.Split(id, "@")
	if len(parts) == 2 {
		if parts[1] == "" {
			return "", "", fmt.Errorf("%w: builderID: '%s'",
				serrors.ErrorInvalidFormat, id)
		}
		return parts[0], parts[1], nil
	}

	if len(parts) == 1 && !needVersion {
		return parts[0], "", nil
	}

	return "", "", fmt.Errorf("%w: builderID: '%s'",
		serrors.ErrorInvalidFormat, id)
}

func ValidateGitHubTagRef(tag string) error {
	if !strings.HasPrefix(tag, "refs/tags/") {
		return fmt.Errorf("%w: %s: not of the form 'refs/tags/name'", serrors.ErrorInvalidRef, tag)
	}
	return nil
}

func TagFromGitHubRef(ref string) (string, error) {
	if err := ValidateGitHubTagRef(ref); err != nil {
		return "", err
	}
	return strings.TrimPrefix(ref, "refs/tags/"), nil
}
