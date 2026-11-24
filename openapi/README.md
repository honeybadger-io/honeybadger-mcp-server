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
| Checkout when copied | `master` @ `b5c72c6763` |
| Copied | 2026-11-20 |
| sha256 | `82e8043ae536cb99155d1a6e16306008adad3d9786754238100f00bcaffab499` |
