package v1

import (
	"fmt"

	serrors "github.com/slsa-framework/slsa-verifier/v2/errors"

	"github.com/slsa-framework/slsa-verifier/v2/verifiers/internal/gha/slsaprovenance/common"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/utils"
)

// BYOBProvenance is SLSA v1.0 provenance for the slsa-github-generator BYOB build type.
type BYOBProvenance struct {
	*provenanceV1
}

// GetBranch implements Provenance.GetBranch.
func (p *BYOBProvenance) GetBranch() (string, error) {
	sourceURI, err := p.SourceURI()
	if err != nil {
		// Get the value from the internalParameters if there is no source URI.
		sysParams, ok := p.prov.Predicate.BuildDefinition.InternalParameters.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%w: %s", serrors.ErrorInvalidDssePayload, "internal parameters type")
		}
		return common.GetBranch(sysParams, true)
	}

	// Returns the branch from the source URI if available.
	_, ref, err := utils.ParseGitURIAndRef(sourceURI)
	if err != nil {
		return "", fmt.Errorf("parsing source uri: %w", err)
	}

	if ref == "" {
		return "", fmt.Errorf("%w: unable to get ref for source %q",
			serrors.ErrorInvalidDssePayload, sourceURI)
	}

	refType, _ := utils.ParseGitRef(ref)
	switch refType {
	case "heads": // branch.
		// NOTE: We return the full git ref.
		return ref, nil
	case "tags":
		// NOTE: If the ref type is a tag we want to try to parse out the branch from the tag.
		sysParams, ok := p.prov.Predicate.BuildDefinition.InternalParameters.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%w: %s", serrors.ErrorInvalidDssePayload, "internal parameters type")
		}
		return common.GetBranch(sysParams, true)
	default:
		return "", fmt.Errorf("%w: unknown ref type %q for ref %q",
			serrors.ErrorInvalidDssePayload, refType, ref)
	}
}

// GetTag implements Provenance.GetTag.
func (p *BYOBProvenance) GetTag() (string, error) {
	sourceURI, err := p.SourceURI()
	if err != nil {
		// Get the value from the internalParameters if there is no source URI.
		sysParams, ok := p.prov.Predicate.BuildDefinition.InternalParameters.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%w: %s", serrors.ErrorInvalidDssePayload, "system parameters type")
		}

		return common.GetTag(sysParams, true)
	}

	// Returns the branch from the source URI if available.
	_, ref, err := utils.ParseGitURIAndRef(sourceURI)
	if err != nil {
		return "", fmt.Errorf("parsing source uri: %w", err)
	}

	if ref == "" {
		return "", fmt.Errorf("%w: unable to get ref for source %q",
			serrors.ErrorInvalidDssePayload, sourceURI)
	}

	refType, _ := utils.ParseGitRef(ref)
	switch refType {
	case "heads": // branch.
		return "", nil
	case "tags":
		// NOTE: We return the full git ref.
		return ref, nil
	default:
		return "", fmt.Errorf("%w: unknown ref type %q for ref %q",
			serrors.ErrorInvalidDssePayload, refType, ref)
	}
}
