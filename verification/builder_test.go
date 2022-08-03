package verification

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_VerifyWorkflowIdentity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		workflow  *WorkflowIdentity
		buildOpts *BuilderOpts
		builderID string
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
			source: "asraa/slsa-on-github-test",
			err:    ErrorMalformedURI,
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
			source: "asraa/slsa-on-github-test",
			err:    ErrorUntrustedReusableWorkflow,
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
			source: "asraa/slsa-on-github-test",
			err:    errorInvalidRef,
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
			source: trustedBuilderRepository,
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
			source: e2eTestRepository,
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
			source: "malicious/source",
			err:    ErrorMismatchSource,
		},
		{
			name: "valid main ref for builder",
			workflow: &WorkflowIdentity{
				CallerRepository:  trustedBuilderRepository,
				JobWobWorkflowRef: trustedBuilderRepository + "/.github/workflows/builder_go_slsa3.yml@refs/heads/main",
				Trigger:           "workflow_dispatch",
				Issuer:            certOidcIssuer,
			},
			source: "malicious/source",
			err:    ErrorMismatchSource,
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
			source: "asraa/slsa-on-github-test",
			err:    ErrorMismatchSource,
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
			source: "asraa/slsa-on-github-test",
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
			source: "asraa/slsa-on-github-test",
			err:    errorInvalidRef,
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
			source: "asraa/slsa-on-github-test",
			err:    errorInvalidRef,
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
			source: "asraa/slsa-on-github-test",
			err:    errorInvalidRef,
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
			source: "github.com/asraa/slsa-on-github-test",
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// TODO: builderID
			_, err := VerifyWorkflowIdentity(tt.workflow, tt.buildOpts, tt.source)
			if !errCmp(err, tt.err) {
				t.Errorf(cmp.Diff(err, tt.err, cmpopts.EquateErrors()))
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
		expected error
	}{
		{
			name: "default trusted",
			path: trustedBuilderRepository + "/.github/workflows/generator_generic_slsa3.yml",
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
			expected: ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch org GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/other/repo/someBuilderID"),
			expected: ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch name GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/some/other/someBuilderID"),
			expected: ErrorUntrustedReusableWorkflow,
		},
		{
			name:     "mismatch id GitHub",
			path:     "some/repo/someBuilderID",
			id:       asStringPointer("https://github.com/some/repo/ID"),
			expected: ErrorUntrustedReusableWorkflow,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := verifyTrustedBuilderID(tt.path, tt.id)
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
			expected:   errorInvalidRef,
		},
		{
			name:       "no min semver for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with prerelease for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3+123",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for builder",
			callerRepo: trustedBuilderRepository,
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   errorInvalidRef,
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
			expected:   errorInvalidRef,
		},
		{
			name:       "no min semver for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with prerelease for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3+123",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for test repo",
			callerRepo: e2eTestRepository,
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   errorInvalidRef,
		},
		// Other repos.
		{
			name:       "main not allowed for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/heads/main",
			expected:   errorInvalidRef,
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
			expected:   errorInvalidRef,
		},
		{
			name:       "no min semver for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with prerelease for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3-alpha",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3+123",
			expected:   errorInvalidRef,
		},
		{
			name:       "full semver with build/prerelease for other repos",
			callerRepo: "some/repo",
			builderRef: "refs/tags/v1.2.3-alpha+123",
			expected:   errorInvalidRef,
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
