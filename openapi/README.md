# OpenAPI bundle

`bundled.yaml` is honeybadger's generated v3 bundle (`bin/rails openapi:bundle`),
copied by `make update-spec`. Don't edit it: fix the spec in honeybadger, then
copy the new bundle into every downstream project (api-go, docs, terraform, this server).

The sha256 identifies the bundle. In honeybadger the bundle is a generated,
gitignored file, so the branch and commit are honeybadger's checkout when the
bundle was copied, not a record of what built it. update-spec refuses a bundle
older than its sources, which catches the usual stale copy.

| | |
| --- | --- |
| Source | `honeybadger/openapi/v3/bundled.yaml` |
| Checkout when copied | `master` @ `4d078e38e4` (api-go v0.14.0's vendored bundle) |
| Copied | 2026-11-24 |
| sha256 | `75a3e76a82c6309aab5812d7d479706135dabcd425c8a314ebd1336560227cf8` |
