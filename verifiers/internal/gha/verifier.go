package gha

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/rekor/pkg/client"

	serrors "github.com/slsa-framework/slsa-verifier/v2/errors"
	"github.com/slsa-framework/slsa-verifier/v2/options"
	"github.com/slsa-framework/slsa-verifier/v2/register"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/utils"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/utils/container"
)

const VerifierName = "GHA"

//nolint:gochecknoinits
func init() {
	register.RegisterVerifier(VerifierName, GHAVerifierNew())
}

type GHAVerifier struct{}

func GHAVerifierNew() *GHAVerifier {
	return &GHAVerifier{}
}

// IsAuthoritativeFor returns true of the verifier can verify provenance
// generated by the builderID.
func (v *GHAVerifier) IsAuthoritativeFor(builderID string) bool {
	// This verifier only supports builders defined on GitHub.
	return strings.HasPrefix(builderID, httpsGithubCom)
}

func verifyEnvAndCert(env *dsse.Envelope,
	cert *x509.Certificate,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
	defaultBuilders map[string]bool,
) ([]byte, *utils.TrustedBuilderID, error) {
	/* Verify properties of the signing identity. */
	// Get the workflow info given the certificate information.
	workflowInfo, err := GetWorkflowInfoFromCertificate(cert)
	if err != nil {
		return nil, nil, err
	}

	// Verify the builder identity.
	builderID, err := VerifyBuilderIdentity(workflowInfo, builderOpts, defaultBuilders)
	if err != nil {
		return nil, nil, err
	}

	// Verify the source repository from the certificate.
	if err := VerifyCertficateSourceRepository(workflowInfo, provenanceOpts.ExpectedSourceURI); err != nil {
		return nil, nil, err
	}

	// Verify properties of the SLSA provenance.
	// Unpack and verify info in the provenance, including the subject Digest.
	provenanceOpts.ExpectedBuilderID = builderID.String()
	if err := VerifyProvenance(env, provenanceOpts); err != nil {
		return nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "Verified build using builder https://github.com%s at commit %s\n",
		workflowInfo.SubjectWorkflowRef,
		workflowInfo.SourceSha1)
	// Return verified provenance.
	r, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, nil, err
	}

	return r, builderID, nil
}

func verifyNpmEnvAndCert(env *dsse.Envelope,
	cert *x509.Certificate,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
	defaultBuilders map[string]bool,
) (*utils.TrustedBuilderID, error) {
	/* Verify properties of the signing identity. */
	// Get the workflow info given the certificate information.
	workflowInfo, err := GetWorkflowInfoFromCertificate(cert)
	if err != nil {
		return nil, err
	}

	// Verify the workflow identity.
	// We verify against the delegator re-usable workflow, not the user-provided
	// builder. This is because the signing identity for delegator-based builders
	// is *always* the delegator workflow.
	expectedDelegatorWorkflow := httpsGithubCom + delegatorLowPermsGenericReusableWorkflow
	delegatorBuilderOpts := options.BuilderOpts{
		ExpectedID: &expectedDelegatorWorkflow,
	}
	trustedBuilderID, err := VerifyBuilderIdentity(workflowInfo, &delegatorBuilderOpts, defaultBuilders)
	// We accept a non-trusted builder for the default npm builder
	// that uses npm CLI.
	if err != nil && !errors.Is(err, serrors.ErrorUntrustedReusableWorkflow) {
		return nil, err
	}

	// TODO(#493): retrieve certificate information to match
	// with the provenance.
	// Today it's not possible due to lack of information in the cert.
	// Verify the source repository from the certificate.
	if err := VerifyCertficateSourceRepository(workflowInfo, provenanceOpts.ExpectedSourceURI); err != nil {
		return nil, err
	}

	// Users must always provide the builder ID.
	if builderOpts == nil || builderOpts.ExpectedID == nil {
		return nil, fmt.Errorf("builder ID is empty")
	}

	// WARNING: builderID may be empty if it's not a trusted reusable builder workflow.
	if trustedBuilderID != nil {
		// We only support builders built using the BYOB framework.
		// The builder is guaranteed to be delegatorGenericReusableWorkflow, since this is the builder
		// we compare against in the call to VerifyBuilderIdentity() above.
		// The delegator workflow will set the builder ID to the caller's path,
		// which is what users match against.
		provenanceOpts.ExpectedBuilderID = *builderOpts.ExpectedID
	} else {
		// NOTE: if the user created provenance using a re-usable workflow
		// that does not integrate with the BYOB framework, this code will be run.
		// In this case, the re-usable workflow must set the builder ID to
		// builderGitHubRunnerID. This means we treat arbitrary re-usable
		// workflows like the default GitHub Action runner. Note that
		// the SAN in the certificate is *different* from the builder ID
		// provided by users during verification.
		// We may add support for verifying provenance from arbitrary re-usable workflows
		// later; which may be useful for org-level builders.

		// TODO(https://github.com/gh-community/npm-provenance-private-beta-community/issues/9#issuecomment-1516685721):
		// update the builder ID based on self-vs-GitHub hosted status in the cert.
		// The builder.id is set to builderGitHubRunnerID by the npm CLI.
		trustedBuilderID, err = utils.TrustedBuilderIDNew(builderGitHubRunnerID, false)
		if err != nil {
			return nil, err
		}
		if err := trustedBuilderID.MatchesLoose(*builderOpts.ExpectedID, false); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
		// On GitHub we only support the default GitHub runner builder.
		provenanceOpts.ExpectedBuilderID = builderGitHubRunnerID
	}

	// Verify properties of the SLSA provenance.
	// Unpack and verify info in the provenance, including the Subject Digest.
	if err := VerifyNpmPackageProvenance(env, provenanceOpts); err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "Verified build using builder %s at commit %s\n",
		trustedBuilderID.String(),
		workflowInfo.SourceSha1)

	return trustedBuilderID, nil
}

// VerifyArtifact verifies provenance for an artifact.
func (v *GHAVerifier) VerifyArtifact(ctx context.Context,
	provenance []byte, artifactHash string,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
) ([]byte, *utils.TrustedBuilderID, error) {
	isSigstoreBundle := IsSigstoreBundle(provenance)
	if isSigstoreBundle && !options.ExperimentalEnabled() {
		return nil, nil, errors.New("sigstore bundle support is only provided in SLSA_VERIFIER_EXPERIMENTAL mode")
	}

	// This includes a default retry count of 3.
	rClient, err := client.GetRekorClient(defaultRekorAddr)
	if err != nil {
		return nil, nil, err
	}

	trustedRoot, err := GetTrustedRoot(ctx)
	if err != nil {
		return nil, nil, err
	}

	var signedAtt *SignedAttestation
	/* Verify signature on the intoto attestation. */
	if isSigstoreBundle {
		signedAtt, err = VerifyProvenanceBundle(ctx, provenance, trustedRoot)
	} else {
		signedAtt, err = VerifyProvenanceSignature(ctx, trustedRoot, rClient,
			provenance, artifactHash)
	}
	if err != nil {
		return nil, nil, err
	}

	return verifyEnvAndCert(signedAtt.Envelope, signedAtt.SigningCert,
		provenanceOpts, builderOpts,
		defaultArtifactTrustedReusableWorkflows)
}

// VerifyImage verifies provenance for an OCI image.
func (v *GHAVerifier) VerifyImage(ctx context.Context,
	provenance []byte, artifactImage string,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
) ([]byte, *utils.TrustedBuilderID, error) {
	/* Retrieve any valid signed attestations that chain up to Fulcio root CA. */
	trustedRoot, err := GetTrustedRoot(ctx)
	if err != nil {
		return nil, nil, err
	}
	opts := &cosign.CheckOpts{
		RootCerts:         trustedRoot.FulcioRoot,
		IntermediateCerts: trustedRoot.FulcioIntermediates,
		RekorPubKeys:      trustedRoot.RekorPubKeys,
		CTLogPubKeys:      trustedRoot.CTPubKeys,
	}

	atts, _, err := container.RunCosignImageVerification(ctx,
		artifactImage, opts)
	if err != nil {
		return nil, nil, err
	}

	/* Now verify properties of the attestations */
	var errs []error
	var builderID *utils.TrustedBuilderID
	var verifiedProvenance []byte
	for _, att := range atts {
		pyld, err := att.Payload()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error getting payload from OCI registry %s", err)
			continue
		}
		env, err := EnvelopeFromBytes(pyld)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error parsing envelope from OCI registry %s", err)
			continue
		}
		cert, err := att.Cert()
		if err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error getting certificate from OCI registry %s", err)
			continue
		}
		verifiedProvenance, builderID, err = verifyEnvAndCert(env,
			cert, provenanceOpts, builderOpts,
			defaultContainerTrustedReusableWorkflows)
		if err == nil {
			return verifiedProvenance, builderID, nil
		}
		errs = append(errs, err)
	}

	// Return the first error.
	if len(errs) > 0 {
		var s string
		if len(errs) > 1 {
			s = fmt.Sprintf(": %v", errs[1:])
		}
		return nil, nil, fmt.Errorf("%w%s", errs[0], s)
	}
	return nil, nil, fmt.Errorf("%w", serrors.ErrorNoValidSignature)
}

// VerifyNpmPackage verifies an npm package tarball.
func (v *GHAVerifier) VerifyNpmPackage(ctx context.Context,
	attestations []byte, tarballHash string,
	provenanceOpts *options.ProvenanceOpts,
	builderOpts *options.BuilderOpts,
) ([]byte, *utils.TrustedBuilderID, error) {
	trustedRoot, err := GetTrustedRoot(ctx)
	if err != nil {
		return nil, nil, err
	}

	npm, err := NpmNew(ctx, trustedRoot, attestations)
	if err != nil {
		return nil, nil, err
	}

	// Verify provenance signature.
	if err := npm.verifyProvenanceAttestationSignature(); err != nil {
		return nil, nil, err
	}

	// Verify publish attesttation signature.
	if err := npm.verifyPublishAttesttationSignature(); err != nil {
		return nil, nil, err
	}

	// Verify attestation headers.
	if err := npm.verifyIntotoHeaders(); err != nil {
		return nil, nil, err
	}

	// Verify package names match.
	if provenanceOpts != nil {
		if err := npm.verifyPackageName(provenanceOpts.ExpectedPackageName); err != nil {
			return nil, nil, err
		}

		if err := npm.verifyPackageVersion(provenanceOpts.ExpectedPackageVersion); err != nil {
			return nil, nil, err
		}
	}

	// Verify certificate information.
	builder, err := verifyNpmEnvAndCert(npm.ProvenanceEnvelope(),
		npm.ProvenanceLeafCertificate(),
		provenanceOpts, builderOpts,
		defaultBYOBReusableWorkflows)
	if err != nil {
		return nil, nil, err
	}

	prov, err := npm.verifiedProvenanceBytes()
	if err != nil {
		return nil, nil, err
	}

	return prov, builder, nil
}
