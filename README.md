# Verification of SLSA provenance
This repository contains the implementation for verifying [SLSA provenance](https://slsa.dev/). It currently supports verifying provenance generated by the [SLSA generator for Go projects](https://github.com/slsa-framework/slsa-github-generator-go). We are working on support for verifying provenance for other ecosystems.

________
[Verification of provenance](#verification-of-provenance)
- [Available options](#available-options)
- [Example](#example)

[Technical design](#technial-design)
- [Blog posts](#blog-posts)
- [Specifications](#specifications)
________

## Verification of Provenance

### Available options

Below is a list of options currently supported. Note that signature verification is handled seamlessly without the need for developers to manipulate public keys.

```bash
$ git clone git@github.com:slsa-framework/slsa-verifier.git
$ go run . --help
 Usage of ./slsa-verifier:
  -artifact-path string
    	path to an artifact to verify
  -branch string
    	expected branch the binary was compiled from (default "main")
  -provenance string
    	path to a provenance file
  -source string
    	expected source repository that should have produced the binary, e.g. github.com/some/repo
  -tag string
    	[optional] expected tag the binary was compiled from
  -versioned-tag string
    	[optional] expected version the binary was compiled from. Uses semantic version to match the tag
```

### Example

```bash
$ go run . --artifact-path ~/Downloads/binary-linux-amd64 --provenance ~/Downloads/binary-linux-amd64.intoto.jsonl --source github.com/origin/repo

Verified against tlog entry 1544571
verified SLSA provenance produced at 
 {
        "caller": "origin/repo",
        "commit": "0dfcd24824432c4ce587f79c918eef8fc2c44d7b",
        "job_workflow_ref": "/slsa-framework/slsa-github-generator-go/.github/workflows/builder.yml@refs/heads/main",
        "trigger": "workflow_dispatch",
        "issuer": "https://token.actions.githubusercontent.com"
}
successfully verified SLSA provenance
```
## Technical design

### Blog post
Find our blog post series [here](https://security.googleblog.com/2022/04/improving-software-supply-chain.html).

### Specifications
For a more in-depth technical dive, read the [SPECIFICATIONS.md](https://github.com/slsa-framework/slsa-github-generator-go/blob/main/SPECIFICATIONS.md).
