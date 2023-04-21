package gha

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	serrors "github.com/slsa-framework/slsa-verifier/v2/errors"
	"github.com/slsa-framework/slsa-verifier/v2/options"
	"github.com/slsa-framework/slsa-verifier/v2/verifiers/utils"
)

var (
	trustedBuilderRepository = "slsa-framework/slsa-github-generator"
	e2eTestRepository        = "slsa-framework/example-package"
	certOidcIssuer           = "https://token.actions.githubusercontent.com"
	githubCom                = "github.com/"
	httpsGithubCom           = "https://" + githubCom
	// This is used in cosign's CheckOpts for validating the certificate. We
	// do specific builder verification after this.
	certSubjectRegexp = httpsGithubCom + "*"
)

var defaultArtifactTrustedReusableWorkflows = map[string]bool{
	trustedBuilderRepository + "/.github/workflows/generator_generic_slsa3.yml":    true,
	trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml":           true,
	trustedBuilderRepository + "/.github/workflows/builder_docker-based_slsa3.yml": true,
}

var defaultContainerTrustedReusableWorkflows = map[string]bool{
	trustedBuilderRepository + "/.github/workflows/generator_container_slsa3.yml": true,
}

var delegatorGenericReusableWorkflow = trustedBuilderRepository + "/.github/workflows/delegator_generic_slsa3.yml"
var delegatorLowPermsGenericReusableWorkflow = trustedBuilderRepository + "/.github/workflows/delegator_lowperms-generic_slsa3.yml"

var defaultBYOBReusableWorkflows = map[string]bool{
	delegatorGenericReusableWorkflow:         true,
	delegatorLowPermsGenericReusableWorkflow: true,
}

// VerifyCertficateSourceRepository verifies the source repository.
func VerifyCertficateSourceRepository(id *WorkflowIdentity,
	sourceRepo string,
) error {
	// The caller repository in the x509 extension is not fully qualified. It only contains
	// {org}/{repository}.
	expectedSource := strings.TrimPrefix(sourceRepo, "git+https://")
	expectedSource = strings.TrimPrefix(expectedSource, githubCom)
	if id.SourceRepository != expectedSource {
		return fmt.Errorf("%w: expected source '%s', got '%s'", serrors.ErrorMismatchSource,
			expectedSource, id.SourceRepository)
	}
	return nil
}

// VerifyBuilderIdentity verifies the signing certificate information.
// Builder IDs are verified against an expected builder ID provided in the
// builerOpts, or against the set of defaultBuilders provided. The identiy
// in the certificate corresponds to a GitHub workflow's path.
func VerifyBuilderIdentity(id *WorkflowIdentity,
	builderOpts *options.BuilderOpts,
	defaultBuilders map[string]bool,
) (*utils.TrustedBuilderID, error) {
	// Issuer verification.
	// NOTE: this is necessary before we do any further verification.
	if id.Issuer != certOidcIssuer {
		return nil, fmt.Errorf("%w: %s", serrors.ErrorInvalidOIDCIssuer, id.Issuer)
	}

	// cert URI path is /org/repo/path/to/workflow@ref
	workflowPath := strings.SplitN(id.SubjectWorkflowRef, "@", 2)
	if len(workflowPath) < 2 {
		return nil, fmt.Errorf("%w: workflow uri: %s", serrors.ErrorMalformedURI, id.SubjectWorkflowRef)
	}

	// Verify trusted workflow.
	reusableWorkflowPath := strings.Trim(workflowPath[0], "/")
	reusableWorkflowTag := strings.Trim(workflowPath[1], "/")
	builderID, err := verifyTrustedBuilderID(reusableWorkflowPath, reusableWorkflowTag,
		builderOpts.ExpectedID, defaultBuilders)
	if err != nil {
		return nil, err
	}

	// Verify the ref is a full semantic version tag.
	if err := verifyTrustedBuilderRef(id, reusableWorkflowTag); err != nil {
		return nil, err
	}

	return builderID, nil
}

// Verifies the builder ID at path against an expected builderID.
// If an expected builderID is not provided, uses the defaultBuilders.
func verifyTrustedBuilderID(certPath, certTag string, expectedBuilderID *string, defaultBuilders map[string]bool) (*utils.TrustedBuilderID, error) {
	var trustedBuilderID *utils.TrustedBuilderID
	var err error
	certBuilderName := httpsGithubCom + certPath
	// WARNING: we don't validate the tag here, because we need to allow
	// refs/heads/main for e2e tests. See verifyTrustedBuilderRef().
	// No builder ID provided by user: use the default trusted workflows.
	if expectedBuilderID == nil || *expectedBuilderID == "" {
		if _, ok := defaultBuilders[certPath]; !ok {
			return nil, fmt.Errorf("%w: %s got %t", serrors.ErrorUntrustedReusableWorkflow, certPath, expectedBuilderID == nil)
		}
		// Construct the builderID using the certificate's builder's name and tag.
		trustedBuilderID, err = utils.TrustedBuilderIDNew(certBuilderName+"@"+certTag, true)
		if err != nil {
			return nil, err
		}
	} else {
		// Verify the builderID.
		// We only accept IDs on github.com.
		trustedBuilderID, err = utils.TrustedBuilderIDNew(certBuilderName+"@"+certTag, true)
		if err != nil {
			return nil, err
		}

		// BuilderID provided by user should match the certificate.
		// Note: the certificate builderID has the form `name@refs/tags/v1.2.3`,
		// so we pass `allowRef = true`.
		if err := trustedBuilderID.MatchesLoose(*expectedBuilderID, true); err != nil {
			return nil, fmt.Errorf("%w: %v", serrors.ErrorUntrustedReusableWorkflow, err)
		}
	}

	return trustedBuilderID, nil
}

// Only allow `@refs/heads/main` for the builder and the e2e tests that need to work at HEAD.
// This lets us use the pre-build builder binary generated during release (release happen at main).
// For other projects, we only allow semantic versions that map to a release.
func verifyTrustedBuilderRef(id *WorkflowIdentity, ref string) error {
	if (id.SourceRepository == trustedBuilderRepository ||
		id.SourceRepository == e2eTestRepository) &&
		options.TestingEnabled() {
		// Allow verification on the main branch to support e2e tests.
		if ref == "refs/heads/main" {
			return nil
		}

		// Extract the tag.
		pin, err := utils.TagFromGitHubRef(ref)
		if err != nil {
			return err
		}

		// Tags on trusted repositories should be a valid semver with version
		// core including all three parts and no build identifier.
		versionCore := strings.Split(pin, "-")[0]
		if !semver.IsValid(pin) ||
			len(strings.Split(versionCore, ".")) != 3 ||
			semver.Build(pin) != "" {
			return fmt.Errorf("%w: %s: version tag not valid", serrors.ErrorInvalidRef, pin)
		}

		return nil
	}

	// Extract the pin.
	pin, err := utils.TagFromGitHubRef(ref)
	if err != nil {
		return err
	}

	// Valid semver of the form vX.Y.Z with no metadata.
	if !(semver.IsValid(pin) &&
		len(strings.Split(pin, ".")) == 3 &&
		semver.Prerelease(pin) == "" &&
		semver.Build(pin) == "") {
		return fmt.Errorf("%w: %s: not of the form vX.Y.Z", serrors.ErrorInvalidRef, pin)
	}
	return nil
}

func getExtension(cert *x509.Certificate, oid string, encoded bool) (string, error) {
	for _, ext := range cert.Extensions {
		if strings.Contains(ext.Id.String(), oid) {
			if !encoded {
				return string(ext.Value), nil
			}

			// Decode first.
			var decoded string
			rest, err := asn1.Unmarshal(ext.Value, &decoded)
			if err != nil {
				return "", fmt.Errorf("%w", err)
			}
			if len(rest) != 0 {
				return "", fmt.Errorf("decoding has rest")
			}
			return decoded, nil
		}
	}
	return "", nil
}

type Hosted int

const (
	HostedSelf Hosted = iota
	HostedGitHub
)

// See https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md.
type WorkflowIdentity struct {
	// The source repository
	SourceRepository string
	// The commit SHA where the workflow was BuildTriggered
	SourceSha1 string
	// Ref of the source.
	SourceRef *string
	// ID of the source repository.
	SourceID *string
	//  Source owner ID of repository.
	SourceOwnerID *string

	// Workflow path OIDC subject - ref of reuseable workflow or trigger workflow.
	SubjectWorkflowRef string
	// Subject commit sha1.
	SubjectSha1 *string
	// Hosted status of the subject.
	SubjectHosted *Hosted

	// BuildTrigger
	BuildTrigger string
	// Build config path, i.e. the trigger workflow.
	BuildConfigPath *string

	// Run ID
	RunID *string
	// Issuer
	Issuer string
}

func getHosted(cert *x509.Certificate) (Hosted, error) {
	ret := HostedSelf
	runnerEnv, err := getExtension(cert, "1.3.6.1.4.1.57264.1.11", true)
	if err != nil {
		return ret, err
	}
	if runnerEnv == "github-hosted" {
		return HostedGitHub, nil
	}
	return ret, nil
}

func validateClaimsEqual(deprecated, new string) error {
	// derecated may be empty, but it more likely the cert is old and 'new' is empty.
	if deprecated != "" && new != "" && deprecated != new {
		return fmt.Errorf("%w: '%v' != '%v'", serrors.ErrorInvalidFormat, deprecated, new)
	}
	return nil
}

// GetWorkflowFromCertificate gets the workflow identity from the Fulcio authenticated content.
func GetWorkflowInfoFromCertificate(cert *x509.Certificate) (*WorkflowIdentity, error) {
	if len(cert.URIs) == 0 {
		return nil, fmt.Errorf("%w: missing URI information from certificate", serrors.ErrorInvalidFormat)
	}

	// 1.3.6.1.4.1.57264.1.2: DEPRECATED.
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#1361415726412--github-workflow-BuildTrigger-deprecated
	deprecatedBuildTrigger, err := getExtension(cert, "1.3.6.1.4.1.57264.1.2", false)
	if err != nil {
		return nil, err
	}
	// 1.3.6.1.4.1.57264.1.20 | Build Trigger
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264120--build-trigger
	buildTrigger, err := getExtension(cert, "1.3.6.1.4.1.57264.1.20", true)
	if err != nil {
		return nil, err
	}
	if err := validateClaimsEqual(deprecatedBuildTrigger, buildTrigger); err != nil {
		return nil, err
	}
	// Handle old certifcates.
	if buildTrigger == "" {
		buildTrigger = deprecatedBuildTrigger
	}

	// 1.3.6.1.4.1.57264.1.3: DEPRECATED.
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#1361415726413--github-workflow-sha-deprecated
	deprecatedSourceSha1, err := getExtension(cert, "1.3.6.1.4.1.57264.1.3", false)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.5: DEPRECATED.
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#1361415726415--github-workflow-repository-deprecated
	deprecatedSourceRepository, err := getExtension(cert, "1.3.6.1.4.1.57264.1.5", false)
	if err != nil {
		return nil, err
	}
	// 1.3.6.1.4.1.57264.1.12 | Source Repository URI
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264112--source-repository-uri
	sourceURI, err := getExtension(cert, "1.3.6.1.4.1.57264.1.12", true)
	if err != nil {
		return nil, err
	}
	if deprecatedSourceRepository != "" &&
		"https://github.com/"+deprecatedSourceRepository != sourceURI {
		return nil, fmt.Errorf("%w: '%v' != '%v'",
			serrors.ErrorInvalidFormat, "https://github.com/"+deprecatedSourceRepository, sourceURI)
	}
	sourceRepository := strings.TrimPrefix(sourceURI, "https://github.com/")
	// Handle old certifcates.
	if sourceRepository == "" {
		sourceRepository = deprecatedSourceRepository
	}

	// IssuerV1: 1.3.6.1.4.1.57264.1.8
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#1361415726411--issuer
	issuerV1, err := getExtension(cert, "1.3.6.1.4.1.57264.1.1", false)
	if err != nil {
		return nil, err
	}

	// IssuerV2: 1.3.6.1.4.1.57264.1.1
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#1361415726418--issuer-v2
	issuerV2, err := getExtension(cert, "1.3.6.1.4.1.57264.1.8", true)
	if err != nil {
		return nil, err
	}

	if issuerV1 != issuerV2 {
		return nil, fmt.Errorf("%w: issuers: '%v' != '%v'", serrors.ErrorInvalidFormat, issuerV1, issuerV2)
	}

	// 1.3.6.1.4.1.57264.1.10 | Build Signer Digest
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264110--build-signer-digest
	subjectSha1, err := getExtension(cert, "1.3.6.1.4.1.57264.1.10", true)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.11 | Runner Environment
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264111--runner-environment
	subjectHosted, err := getHosted(cert)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.13 | Source Repository Digest
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264113--source-repository-digest
	sourceSha1, err := getExtension(cert, "1.3.6.1.4.1.57264.1.13", true)
	if err != nil {
		return nil, err
	}
	if err := validateClaimsEqual(deprecatedSourceSha1, sourceSha1); err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.14 | Source Repository Ref
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264114--source-repository-ref
	sourceRef, err := getExtension(cert, "1.3.6.1.4.1.57264.1.14", true)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.15 | Source Repository Identifier
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264115--source-repository-identifier
	sourceID, err := getExtension(cert, "1.3.6.1.4.1.57264.1.15", true)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.17 | Source Repository Owner Identifier
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264117--source-repository-owner-identifier
	sourceOwnerID, err := getExtension(cert, "1.3.6.1.4.1.57264.1.17", true)
	if err != nil {
		return nil, err
	}

	// 1.3.6.1.4.1.57264.1.19 | Build Config Digest
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264119--build-config-digest
	buildConfigSha1, err := getExtension(cert, "1.3.6.1.4.1.57264.1.19", true)
	if err != nil {
		return nil, err
	}
	if buildConfigSha1 != sourceSha1 {
		return nil, fmt.Errorf("%w: '%v' != '%v'",
			serrors.ErrorInvalidFormat, buildConfigSha1, sourceSha1)
	}

	// 1.3.6.1.4.1.57264.1.21 | Run Invocation URI
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264121--run-invocation-uri
	runURI, err := getExtension(cert, "1.3.6.1.4.1.57264.1.21", true)
	if err != nil {
		return nil, err
	}
	runID := strings.TrimPrefix(runURI, fmt.Sprintf("https://github.com/%s/actions/runs/", sourceRepository))

	// 1.3.6.1.4.1.57264.1.18 | Build Config URI
	// https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md#13614157264118--build-config-uri
	buildConfigURI, err := getExtension(cert, "1.3.6.1.4.1.57264.1.18", true)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(buildConfigURI, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: '%v' != '%v'",
			serrors.ErrorInvalidFormat, buildConfigURI)
	}
	prefix := fmt.Sprintf("https://github.com/%v/", sourceRepository)
	if !strings.HasPrefix(parts[0], prefix) {
		return nil, fmt.Errorf("%w: prefix: %v",
			serrors.ErrorInvalidFormat, parts[0])
	}
	buildConfigPath := strings.TrimPrefix(parts[0], prefix)

	// Subject path.
	if !strings.HasPrefix(cert.URIs[0].Path, "/") {
		return nil, fmt.Errorf("%w: %s", serrors.ErrorInvalidFormat, cert.URIs[0].Path)
	}

	return &WorkflowIdentity{
		// Issuer.
		Issuer: issuerV2,
		// Subject
		SubjectWorkflowRef: cert.URIs[0].Path[1:], // Remove the starting '/'
		SubjectSha1:        &subjectSha1,
		SubjectHosted:      &subjectHosted,
		// Source.
		SourceRepository: sourceRepository,
		SourceSha1:       sourceSha1,
		SourceRef:        &sourceRef,
		SourceID:         &sourceID,
		SourceOwnerID:    &sourceOwnerID,
		// Build.
		BuildTrigger:    buildTrigger,
		BuildConfigPath: &buildConfigPath,
		// Other.
		RunID: &runID,
	}, nil
}
