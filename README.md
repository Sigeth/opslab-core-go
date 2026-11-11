# opslab-core-go

The contracts of [opslab](https://github.com/Sigeth) (working name), a self-hosted platform of hands-on DevOps security challenges on Kubernetes, plus the Go libraries shared by the operator, the CLI and the checker.

> `opslab` is a working name and will change.

## Contents

| Path | What |
|---|---|
| `api/v1alpha1` | Go types of the Kubernetes custom resources (`PlatformConfig`, `CatalogSource`, `Challenge`, `ChallengeInstance`, `CheckRun`) and of the documents authors write (`ChallengePackage` = `challenge.yaml`, `CatalogIndex`). **Source of truth** for every other repo. |
| `crds/` | Generated CustomResourceDefinitions. Other repos (Helm chart, Rust types via kopium) consume them from a tag. |
| `schema/` | Generated JSON Schemas for `challenge.yaml` and catalog indexes, for editor completion and validation. |
| `pkg/challenge` | Load, validate and default a challenge package. The operator and the CLI use the same code, so they agree on what a valid package is. |
| `pkg/checkerio` | The checker contract: environment variables, targets file, JSON-lines task results. |
| `pkg/project` | Project-wide identifiers (name, API group, owner, platform). Rename here. |
| `samples/` | Example custom resources. |

## Writing a challenge.yaml

Add this first line to get completion and validation in editors that use the YAML language server (VS Code, Neovim, JetBrains…):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/Sigeth/opslab-core-go/main/schema/challenge-package.schema.json
```

Examples: [`pkg/challenge/testdata`](pkg/challenge/testdata).

## Development

Requirements: Go (version in `go.mod`), make, and optionally kind + kubectl to try the CRDs.

```sh
make generate   # deepcopy, CRDs, JSON Schemas (run after changing api/)
make lint test
kubectl apply --server-side -f crds/ && kubectl apply -f samples/
```

Changing a contract: edit `api/v1alpha1`, run `make generate`, update `pkg/challenge` validation and its tests, then tag a release. `opslab-core-rs` regenerates its Rust types from the tagged `crds/`.

## License

[MIT](LICENSE)
