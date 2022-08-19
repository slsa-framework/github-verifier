package gha

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	serrors "github.com/slsa-framework/slsa-verifier/errors"
	"github.com/slsa-framework/slsa-verifier/options"
)

func Test_VerifyWorkflowIdentity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		workflow  *WorkflowIdentity
		buildOpts *options.BuilderOpts
		builderID string
		defaults  map[string]bool
		source    string
		err       error
	}{
		{
			name: "invalid job workflow ref",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: "random/workflow/ref",
				Trigger:           "workflow_dispatch",
				Issuer:            "https://token.actions.githubusercontent.com",
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorMalformedURI,
		},
		{
			name: "untrusted job workflow ref",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: "/malicious/slsa-go/.github/workflows/builder.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            "https://token.actions.githubusercontent.com",
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name: "untrusted job workflow ref for general repos",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            "https://bad.issuer.com",
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorInvalidRef,
		},
		{
			name: "valid main ref for trusted builder",
			workflow: &WorkflowIdentity{
				CallerRepository:  trustedBuilderRepository,
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            "https://token.actions.githubusercontent.com",
			},
			source:    trustedBuilderRepository,
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid main ref for e2e test",
			workflow: &WorkflowIdentity{
				CallerRepository:  e2eTestRepository,
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:    e2eTestRepository,
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid main ref for e2e test - match builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  e2eTestRepository,
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: e2eTestRepository,
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml"),
			},
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid main ref for e2e test - mismatch builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  e2eTestRepository,
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: e2eTestRepository,
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("some-other-builderID"),
			},
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name: "unexpected source for e2e test",
			workflow: &WorkflowIdentity{
				CallerRepository:  e2eTestRepository,
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:    "malicious/source",
			err:       serrors.ErrorMismatchSource,
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid main ref for builder",
			workflow: &WorkflowIdentity{
				CallerRepository:  trustedBuilderRepository,
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:   "malicious/source",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorMismatchSource,
		},
		{
			name: "unexpected source",
			workflow: &WorkflowIdentity{
				CallerRepository:  "malicious/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorMismatchSource,
		},
		{
			name: "valid workflow identity",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:    "asraa/slsa-on-github-test",
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid workflow identity - match builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "asraa/slsa-on-github-test",
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml"),
			},
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid workflow identity - mismatch builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "asraa/slsa-on-github-test",
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("some-other-builderID"),
			},
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name: "invalid workflow identity with prerelease",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3-alpha",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:    "asraa/slsa-on-github-test",
			err:       serrors.ErrorInvalidRef,
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "invalid workflow identity with build",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3+123",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorInvalidRef,
		},
		{
			name: "invalid workflow identity with metadata",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3-alpha+123",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:   "asraa/slsa-on-github-test",
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorInvalidRef,
		},
		{
			name: "valid workflow identity with fully qualified source",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:    "github.com/asraa/slsa-on-github-test",
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid workflow identity with fully qualified source - no default",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "github.com/asraa/slsa-on-github-test",
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml"),
			},
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid workflow identity with fully qualified source - match builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "github.com/asraa/slsa-on-github-test",
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml"),
			},
			defaults:  defaultArtifactTrustedReusableWorkflows,
			builderID: "https://github.com/" + trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml",
		},
		{
			name: "valid workflow identity with fully qualified source - mismatch builderID",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "github.com/asraa/slsa-on-github-test",
			buildOpts: &options.BuilderOpts{
				ExpectedID: asStringPointer("some-other-builderID"),
			},
			defaults: defaultArtifactTrustedReusableWorkflows,
			err:      serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name: "valid workflow identity with fully qualified source - mismatch defaults",
			workflow: &WorkflowIdentity{
				CallerRepository:  "asraa/slsa-on-github-test",
				CallerHash:        "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.3",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source:   "github.com/asraa/slsa-on-github-test",
			defaults: defaultContainerTrustedReusableWorkflows,
			err:      serrors.ErrorUntrustedReusableWorkflow,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := tt.buildOpts
			if opts == nil {
				opts = &options.BuilderOpts{}
			}
			id, err := VerifyWorkflowIdentity(tt.workflow, opts, tt.source,
				tt.defaults)
			if !errCmp(err, tt.err) {
				t.Errorf(cmp.Diff(err, tt.err, cmpopts.EquateErrors()))
			}
			if err != nil {
				return
			}
			if id != tt.builderID {
				t.Errorf(cmp.Diff(id, tt.builderID))
			}
		})
	}
}

func asStringPointer(s string) *string {
	return &s
}

func Test_verifyTrustedBuilderID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		id       *string
		path     string
		defaults map[string]bool
		expected error
	}{
		{
			name:     "default trusted",
			path:     trustedBuilderRepository + "/.github/workflows/generator_generic_slsa3.yml",
			defaults: defaultArtifactTrustedReusableWorkflows,
		},
		{
			name:     "default mismatch against container defaults",
			path:     trustedBuilderRepository + "/.github/workflows/generator_generic_slsa3.yml",
			defaults: defaultContainerTrustedReusableWorkflows,
			expected: serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name: "valid ID for GitHub builder",
			path: "some/repo/someBuilderID",
			id:   asStringPointer("https://github.com/some/repo/someBuilderID"),
		},
		{
			name:     "non GitHub builder ID",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://not-github.com/some/repo/someBuilderID"),
			expected: serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch org GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/other/repo/someBuilderID"),
			expected: serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch name GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/some/other/someBuilderID"),
			expected: serrors.ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch id GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/some/repo/ID"),
			expected: serrors.ErrorUntrustedReusableWorkflow,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := verifyTrustedBuilderID(tt.path, tt.id, tt.defaults)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected, cmpopts.EquateErrors()))
			}
			if err != nil {
				return
			}
			expectedID := "https://github.com/" + tt.path
			if id != expectedID {
				t.Errorf(cmp.Diff(id, expectedID))
			}
		})
	}
}

func Test_verifyTrustedBuilderRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		callerRepo string
		builderRef string
		expected   error
	}{
		// Trusted repo.
		{
			name:       "main allowed for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/heads/main",
		},
		{
			name:       "full semver for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3",
		},
		{
			name:       "no patch semver for other builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "no min semver for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with prerelease for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3+123",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   serrors.ErrorInvalidRef,
		},
		// E2e tests repo.
		{
			name:       "main allowed for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/heads/main",
		},
		{
			name:       "full semver for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3",
		},
		{
			name:       "no patch semver for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "no min semver for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with prerelease for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3+123",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   serrors.ErrorInvalidRef,
		},
		// Other repos.
		{
			name:       "main not allowed for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/heads/main",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3",
		},
		{
			name:       "no patch semver for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "no min semver for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with prerelease for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3+123",
			expected:   serrors.ErrorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   serrors.ErrorInvalidRef,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wf := WorkflowIdentity{
				CallerRepository: tt.callerRepo,
			}

			err := verifyTrustedBuilderRef(&wf, tt.builderRef)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected, cmpopts.EquateErrors()))
			}
		})
	}
}
