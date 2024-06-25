package verification

import "errors"

var (
	ErrorInvalidDssePayload        = errors.New("invalid DSSE envelope payload")
	ErrorMismatchBranch            = errors.New("branch used to generate the binary does not match provenance")
	ErrorMismatchPackageVersion    = errors.New("package version does not match provenance")
	ErrorMismatchPackageName       = errors.New("package name does not match provenance")
	ErrorMismatchBuilderID         = errors.New("builderID does not match provenance")
	ErrorInvalidBuilderID          = errors.New("builderID is invalid")
	ErrorInvalidBuildType          = errors.New("buildType is invalid")
	ErrorMismatchSource            = errors.New("source used to generate the binary does not match provenance")
	ErrorMismatchWorkflowInputs    = errors.New("workflow input does not match")
	ErrorMalformedURI              = errors.New("URI is malformed")
	ErrorMismatchCertificate       = errors.New("certificate and provenance mismatch")
	ErrorInvalidCertificate        = errors.New("invalid certificate")
	ErrorMismatchTag               = errors.New("tag used to generate the binary does not match provenance")
	ErrorInvalidRecipe             = errors.New("the recipe is invalid")
	ErrorMismatchVersionedTag      = errors.New("tag used to generate the binary does not match provenance")
	ErrorInvalidSemver             = errors.New("invalid semantic version")
	ErrorRekorSearch               = errors.New("error searching rekor entries")
	ErrorMismatchHash              = errors.New("artifact hash does not match provenance subject")
	ErrorNonVerifiableClaim        = errors.New("provenance claim cannot be verified")
	ErrorMismatchIntoto            = errors.New("verified intoto provenance does not match text provenance")
	ErrorInvalidRef                = errors.New("invalid ref")
	ErrorUntrustedReusableWorkflow = errors.New("untrusted reusable workflow")
	ErrorNoValidRekorEntries       = errors.New("could not find a matching valid signature entry")
	ErrorVerifierNotSupported      = errors.New("no verifier support the builder")
	ErrorInvalidOIDCIssuer         = errors.New("invalid OIDC issuer")
	ErrorNotSupported              = errors.New("not supported")
	ErrorInvalidFormat             = errors.New("invalid format")
	ErrorInvalidPEM                = errors.New("invalid PEM")
	ErrorInvalidSignature          = errors.New("invalid signature")
	ErrorNoValidSignature          = errors.New("no valid signature")
	ErrorMutableImage              = errors.New("the image is mutable")
	ErrorImageHash                 = errors.New("cannot retrieve sha256 of image")
	ErrorInvalidEncoding           = errors.New("invalid encoding")
	ErrorInternal                  = errors.New("internal error")
	ErrorInvalidRekorEntry         = errors.New("invalid Rekor entry")
	ErrorRekorPubKey               = errors.New("error retrieving Rekor public keys")
	ErrorInvalidPackageName        = errors.New("invalid package name")
	ErrorInvalidSubject            = errors.New("invalid subject")
	ErrorInvalidHash               = errors.New("invalid hash")
	ErrorNotPresent                = errors.New("not present")
	ErrorInvalidPublicKey          = errors.New("invalid public key")
	ErrorInvalidHashAlgo           = errors.New("unsupported hash algorithm")
	ErrorInvalidVerificationResult = errors.New("verificationResult is not PASSED")
	ErrorMismatchVerifiedLevels    = errors.New("verified levels do not match")
	ErrorMissingSubjectDigest      = errors.New("missing subject digest")
	ErrorEmptyRequiredField        = errors.New("empty value in required field")
	ErrorMismatchResourceURI       = errors.New("resource URI does not match")
	ErrorMismatchVerifierID        = errors.New("verifier ID does not match")
)
