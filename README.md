# Verification of SLSA provenance
This repository contains the implementation for verifying [SLSA provenance](https://slsa.dev/). It currently supports verifying provenance generated by the [SLSA generator for Go projects](https://github.com/slsa-framework/slsa-github-generator/blob/main/.github/workflows/builder_go_slsa3.yml). We are working on support for verifying provenance for other ecosystems.

________
[Installation](#installation)
- [Compilation from source](#compilation-from-source)
- [Download the binary](#download-the-binary)
- [Install in a GitHub Workflow](#install-in-a-github-workflow)

[Verification of provenance](#verification-of-provenance)
- [Available options](#available-options)
- [Example](#example)

[Technical design](#technial-design)
- [Blog posts](#blog-posts)
- [Specifications](#specifications)
________

## Installation

You have two options to install the verifier.

### Compilation from source

#### Option 1: Install via go
```
$ go install github.com/slsa-framework/slsa-verifier/cli/slsa-verifier@v1.3.0
$ slsa-verifier <options>
```

#### Option 2: Compile manually
```
$ git clone git@github.com:slsa-framework/slsa-verifier.git
$ cd slsa-verifier && git checkout v1.3.0
$ go run ./cli/slsa-verifier <options>
```

### Download the binary

Download the binary from the latest release at [https://github.com/slsa-framework/slsa-verifier/releases/tag/v1.3.0](https://github.com/slsa-framework/slsa-verifier/releases/tag/v1.3.0)

Download the [SHA256SUM.md](https://github.com/slsa-framework/slsa-verifier/blob/main/SHA256SUM.md).

Verify the checksum:

```
$ sha256sum -c --strict SHA256SUM.md
  slsa-verifier-linux-amd64: OK
```

### Install in a GitHub Workflow

Using the GitHub Action ```slsa-framework/actions/install``` to install ```slsa-verifier``` on a GitHub Workflow. For full details, see [its documentation](https://github.com/slsa-framework/slsa-verifier/actions/install/README.md).

## Verification of Provenance

### Available options

Below is a list of options currently supported. Note that signature verification is handled seamlessly without the need for developers to manipulate public keys.

```bash
$ git clone git@github.com:slsa-framework/slsa-verifier.git
$ go run ./cli/slsa-verifier --help
 Usage of ./slsa-verifier:
  -artifact-path string
    	path to an artifact to verify
  -branch string
    	expected branch the binary was compiled from (default "main")
  -print-provenance
    	output the verified provenance
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
$ go run ./cli/slsa-verifier -artifact-path ~/Downloads/slsa-verifier-linux-amd64 -provenance ~/Downloads/slsa-verifier-linux-amd64.intoto.jsonl -source github.com/slsa-framework/slsa-verifier -tag v1.3.0
Verified signature against tlog entry index 3189970 at URL: https://rekor.sigstore.dev/api/v1/log/entries/206071d5ca7a2346e4db4dcb19a648c7f13b4957e655f4382b735894059bd199
Verified build using builder https://github.com/slsa-framework/slsa-github-generator/.github/workflows/builder_go_slsa3.yml@refs/tags/v1.2.0 at commit 5bb13ef508b2b8ded49f9264d7712f1316830d10
PASSED: Verified SLSA provenance
```

The verified in-toto statement may be written to stdout with the `--print-provenance` flag to pipe into policy engines.

## Technical design

### Blog post
Find our blog post series [here](https://security.googleblog.com/2022/04/improving-software-supply-chain.html).

### Specifications
For a more in-depth technical dive, read the [SPECIFICATIONS.md](https://github.com/slsa-framework/slsa-github-generator/blob/main/SPECIFICATIONS.md).
