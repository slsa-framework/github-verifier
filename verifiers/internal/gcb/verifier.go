package gcb

import (
	"context"
	"strings"

	serrors "github.com/slsa-framework/slsa-verifier/errors"
	"github.com/slsa-framework/slsa-verifier/options"
	register "github.com/slsa-framework/slsa-verifier/register"
	_ "github.com/slsa-framework/slsa-verifier/verifiers/internal/gcb/keys"
)

const VerifierName = "GCB"

//nolint:gochecknoinits
func init() {
	register.RegisterVerifier(VerifierName, GCBVerifierNew())
}

type GCBVerifier struct{}

func GCBVerifierNew() *GCBVerifier {
	return &GCBVerifier{}
}

// IsAuthoritativeFor returns true of the verifier can verify provenance
// generated by the builderID.
func (v *GCBVerifier) IsAuthoritativeFor(builderID string) bool {
	// This verifier only supports the GCB builders.
	return strings.HasPrefix(builderID, "https://cloudbuild.googleapis.com/GoogleHostedWorker@")
}

// VerifyArtifact verifies provenance for an artifact.
func (v *GCBVerifier) VerifyArtifact(ctx context.Context,
	provenance []byte, artifactHash string,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
) ([]byte, string, error) {
	return nil, "todo", serrors.ErrorNotSupported
}

// VerifyImage verifies provenance for an OCI image.
func (v *GCBVerifier) VerifyImage(ctx context.Context,
	provenance []byte, artifactImage string,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
) ([]byte, string, error) {
	prov, err := ProvenanceFromBytes(provenance)
	if err != nil {
		return nil, "", err
	}

	// Verify signature on the intoto attestation.
	if err = prov.VerifySignature(); err != nil {
		return nil, "", err
	}

	// Verify intoto header.
	if err = prov.VerifyIntotoHeaders(); err != nil {
		return nil, "", err
	}

	// Verify the builder.
	builderID, err := prov.VerifyBuilderID(builderOpts)
	if err != nil {
		return nil, "", err
	}

	// Verify subject digest.
	if err = prov.VerifySubjectDigest(provenanceOpts.ExpectedDigest); err != nil {
		return nil, "", err
	}

	// Verify source.
	if err = prov.VerifySourceURI(provenanceOpts.ExpectedSourceURI); err != nil {
		return nil, "", err
	}

	// Verify branch.
	if provenanceOpts.ExpectedBranch != nil {
		if err = prov.VerifyBranch(*provenanceOpts.ExpectedBranch); err != nil {
			return nil, "", err
		}
	}

	// Verify the tag.
	if provenanceOpts.ExpectedTag != nil {
		if err := prov.VerifyTag(*provenanceOpts.ExpectedTag); err != nil {
			return nil, "", err
		}
	}

	// Verify the versioned tag.
	if provenanceOpts.ExpectedVersionedTag != nil {
		if err := prov.VerifyVersionedTag(*provenanceOpts.ExpectedVersionedTag); err != nil {
			return nil, "", err
		}
	}

	// TODO: verify summary information:
	// - kind: BUILD
	// - resourceUri against image_summary and subject data
	// - text is the same as what is verified.

	content, err := prov.GetVerifiedIntotoStatement()
	if err != nil {
		return nil, "", err
	}
	return content, builderID, nil
}
