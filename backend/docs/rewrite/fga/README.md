<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Crowdfunding FGA — example artifacts (suggested example only)

**These files are a suggested example, not the authoritative model and not a proposed
change to `lfx-v2-helm`.** They exist to make
[`../12-fga-authorization-model.md`](../12-fga-authorization-model.md)'s
`crowdfunding_initiative` proposal runnable against the OpenFGA CLI, the same way a
team-member-circulated alternative model was — this package is the direct response to
that suggestion, built from doc 12 and
[`../11-initiative-attribution-and-access.md`](../11-initiative-attribution-and-access.md)
rather than from the public docs alone. If the model in doc 12 changes, these files can go
stale — treat doc 12 as the source of truth for the reasoning, and these as illustration.

## Contents

| File | Purpose |
|---|---|
| `model.fga` | The `crowdfunding_initiative` type from doc 12, plus minimal stand-in types (`project`, `b2b_org`, `team`) for the ones that really live in the shared platform model |
| `tuples.yaml` | Sample tuples covering every attribution shape |
| `tests.yaml` | Regression tests for doc 12's named merge-gate scenarios, via the OpenFGA CLI |

## Why this shape, not a project-rooted one

A project-rooted model (initiative as a child of `project`, with no `personal` or `b2b_org`
case) was suggested as an alternative. It doesn't fit our model: `attributed_to_type` is
one of `personal` / `organization` / `project`
(`backend/internal/domain/models/initiative.go:114`), and doc 11 §3.4 is explicit that
non-LF initiatives — no `project` in the platform model to hang off — are meant to be the
*simple* case (owner-only, no attribution), not an edge case bolted onto a project-first
design. `tuples.yaml`/`tests.yaml` here deliberately exercise both shapes a project-rooted
model can't express:

- **`personal-draft`** — an owner-only initiative, no `project` or `b2b_org` tuple at all.
  This is what a future non-LF-attributed initiative looks like too.
- **`org-published`** — attributed to a `b2b_org`, not a `project`.

## Running the tests

Requires the [OpenFGA CLI](https://github.com/openfga/cli):

```bash
cd backend/docs/rewrite/fga
fga model test --tests tests.yaml
```

## Trying it interactively

```bash
docker run -d --name openfga \
  -p 8080:8080 -p 8081:8081 -p 3000:3000 \
  openfga/openfga run --playground-enabled --playground-addr 0.0.0.0:3000

FGA_STORE_ID=$(fga store create --name "crowdfunding-fga-example" --api-url http://localhost:8080 \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['store']['id'])")

fga model write --store-id="$FGA_STORE_ID" --api-url http://localhost:8080 --file model.fga
fga tuple import --store-id="$FGA_STORE_ID" --api-url http://localhost:8080 --file tuples.yaml

echo "Playground: http://localhost:3000/playground"
```

Teardown: `docker stop openfga && docker rm openfga`
