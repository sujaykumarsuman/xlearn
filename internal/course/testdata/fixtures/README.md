# Fixture manifests (tests only)

Synthetic course manifests for sprint m1-03's per-course tests: `zz-fixture` (an `active`
second course with its own nav labels and no mock), `zz-preview` (`preview`), `zz-soon`
(`coming_soon`) and `zz-retired` (`retired`). Tests load them through
`internal/course/coursetest.Registry`, which adds them to the embedded manifests. They are
never embedded in an image: no `//go:embed` reaches this directory.
