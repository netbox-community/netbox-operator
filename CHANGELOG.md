# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.11] - 2025-08-18

Bump golang from 1.24.6 to 1.25.0 in the docker group (#412)

<!-- Release notes generated using configuration in .github/release.yml at v0.2.11 -->

## What's Changed
### Version Bumps
* Bump the go-dependencies group with 9 updates by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/414
* Bump actions/checkout from 4 to 5 in the github-actions group by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/413
* Bump github.com/emicklei/go-restful/v3 from 3.12.2 to 3.13.0 in the go-dependencies group by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/416
* Bump golang from 1.24.6 to 1.25.0 in the docker group by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/412

## New Contributors
* @github-actions[bot] made their first contribution in https://github.com/netbox-community/netbox-operator/pull/408

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.10...v0.2.11

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.11)

---

## [v0.2.10] - 2025-08-11

fix: update release workflow to create PR instead of pushing directly… (#407)

<!-- Release notes generated using configuration in .github/release.yml at v0.2.10 -->



**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.9...v0.2.10

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.10)

---

## [v0.2.8] - 2025-08-04

## What's Changed
### Version Bumps
* Bump github.com/prometheus/client_golang from 1.22.0 to 1.23.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/396
* Bump go.opentelemetry.io/proto/otlp from 1.7.0 to 1.7.1 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/397
* Bump github.com/sagikazarmark/locafero from 0.9.0 to 0.10.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/398
* Bump docker/metadata-action from 5.7.0 to 5.8.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/399

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.7...v0.2.8

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.8)

---

## [v0.2.7] - 2025-07-28

## What's Changed
* Fix the codespell --ignore-words-list in pre-commit by @jstudler in https://github.com/netbox-community/netbox-operator/pull/388
* Print logs in json format by @bruelea in https://github.com/netbox-community/netbox-operator/pull/389
### Version Bumps
* Bump github.com/cenkalti/backoff/v5 from 5.0.2 to 5.0.3 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/391
* Bump google.golang.org/grpc from 1.73.0 to 1.74.2 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/392
* Bump sigs.k8s.io/yaml from 1.5.0 to 1.6.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/393
* Bump github.com/onsi/gomega from 1.37.0 to 1.38.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/394

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.6...v0.2.7

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.7)

---

## [v0.2.6] - 2025-07-21

## What's Changed
* Fix iprangeclaim IpRangeName status field yaml/json name by @jstudler in https://github.com/netbox-community/netbox-operator/pull/371
* Add ownerReference to child resource on update by @jstudler in https://github.com/netbox-community/netbox-operator/pull/369
* Minor changes to align across controllers by @jstudler in https://github.com/netbox-community/netbox-operator/pull/372
* Verify prefix length when restoring PrefixClaim from NetBox by @jstudler in https://github.com/netbox-community/netbox-operator/pull/373
* Change of NetBox Operator IpAddress short name to enhance UX in Kubernetes 1.33+ by @jstudler in https://github.com/netbox-community/netbox-operator/pull/375
* Add formatting configurations and yamllint action by @jstudler in https://github.com/netbox-community/netbox-operator/pull/376
* Clean up e2e test files by @jstudler in https://github.com/netbox-community/netbox-operator/pull/382

### Version Bumps
* Bump github.com/fxamacker/cbor/v2 from 2.8.0 to 2.9.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/381
* Bump github.com/google/cel-go from 0.25.0 to 0.26.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/380
* Bump github.com/spf13/pflag from 1.0.6 to 1.0.7 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/379
* Bump k8s.io/api from 0.33.2 to 0.33.3 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/377
* Bump k8s.io/client-go from 0.33.2 to 0.33.3 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/378
* Bump github.com/swisscom/leaselocker from 0.1.0 to 0.2.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/387
* Bump k8s.io/apiserver from 0.33.2 to 0.33.3 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/386
* Bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/384
* Bump k8s.io/apiextensions-apiserver from 0.33.2 to 0.33.3 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/383


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.5...v0.2.6

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.6)

---

## [v0.2.5] - 2025-07-14

## What's Changed
* Add codespell CI job by @jstudler in https://github.com/netbox-community/netbox-operator/pull/368

### Version Bumps
* Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc from 1.36.0 to 1.37.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/364
* Bump golang.org/x/term from 0.32.0 to 0.33.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/363
* Bump go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp from 0.61.0 to 0.62.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/362
* Bump golang.org/x/net from 0.41.0 to 0.42.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/366
* Bump golang.org/x/sync from 0.15.0 to 0.16.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/365
* Bump golang from 1.24.4 to 1.24.5 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/367
* Bump golang.org/x/tools from 0.34.0 to 0.35.0 by @dependabot[bot] in https://github.com/netbox-community/netbox-operator/pull/370


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.4...v0.2.5

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.5)

---

## [v0.2.4] - 2025-07-07

## What's Changed
* Cleanup old ReplicaSets after NetBox deployment patch to prevent volume Multi-Attach errors by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/345

### Version Bumps
* Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/358
* Bump github.com/stoewer/go-strcase from 1.3.0 to 1.3.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/357
* Bump github.com/prometheus/procfs from 0.16.1 to 0.17.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/356
* Bump sigs.k8s.io/apiserver-network-proxy/konnectivity-client from 0.31.2 to 0.33.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/355
* Bump github.com/prometheus/common from 0.63.0 to 0.65.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/354


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.3...v0.2.4

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.4)

---

## [v0.2.3] - 2025-06-30

## What's Changed
* Distinguish if error occurred or if no matching prefix was found by @bruelea in https://github.com/netbox-community/netbox-operator/pull/346

### Version Bumps
* Bump go.uber.org/mock from 0.5.1 to 0.5.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/340
* Bump google.golang.org/grpc from 1.72.2 to 1.73.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/342
* Bump golang.org/x/time from 0.11.0 to 0.12.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/344
* Bump k8s.io/apiextensions-apiserver from 0.33.1 to 0.33.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/343
* Bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.26.3 to 2.27.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/341
* Bump go.opentelemetry.io/otel/trace from 1.36.0 to 1.37.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/347
* Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace from 1.36.0 to 1.37.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/348
* Bump sigs.k8s.io/yaml from 1.4.0 to 1.5.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/349
* Bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.27.0 to 2.27.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/350


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.2...v0.2.3

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.3)

---

## [v0.2.2] - 2025-06-23

## What's Changed
* Configure dependabot to run on mondays at 08:00 in the Europe/Zurich timezone by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/335
* Fix NetBox NGINX deployment for IPv4-only environments by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/333
* Add step to e2e tests to clean up resources in the test NetBox instance by @bruelea in https://github.com/netbox-community/netbox-operator/pull/321

### Version Bumps
* Bump github.com/pelletier/go-toml/v2 from 2.2.3 to 2.2.4 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/332
* Bump go.opentelemetry.io/proto/otlp from 1.6.0 to 1.7.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/330
* Bump cel.dev/expr from 0.23.1 to 0.24.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/331
* Bump go.mongodb.org/mongo-driver from 1.17.3 to 1.17.4 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/328
* Bump github.com/spf13/cobra from 1.8.1 to 1.9.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/329
* Bump NetBox e2e test version from 4.1.8 to 4.1.11 by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/242
* Bump github.com/spf13/cast from 1.7.1 to 1.9.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/338
* Bump k8s.io/apiserver from 0.33.1 to 0.33.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/337
* Bump github.com/fxamacker/cbor/v2 from 2.7.0 to 2.8.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/336
* Bump golang.org/x/tools from 0.31.0 to 0.34.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/334


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.1...v0.2.2

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.2)

---

## [v0.2.1] - 2025-06-16

## What's Changed
* Allow overriding image registry and Helm chart sources in NetBox deploy script by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/296

### Version Bumps
* Bump github.com/go-openapi/errors from 0.22.0 to 0.22.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/327
* Bump github.com/go-logr/logr from 1.4.2 to 1.4.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/326
* Bump github.com/fsnotify/fsnotify from 1.8.0 to 1.9.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/325
* Bump golang.org/x/sync from 0.14.0 to 0.15.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/324
* Bump golang from 1.24.3 to 1.24.4 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/322
* Bump sigs.k8s.io/controller-runtime from 0.20.4 to 0.21.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/323


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.2.0...v0.2.1

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.1)

---

## [v0.2.0] - 2025-06-05

## What's Changed
* Create codeql.yml workflow by @faebr in https://github.com/netbox-community/netbox-operator/pull/315

### Version Bumps
* Bump actions/setup-go from 5.4.0 to 5.5.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/308
* Bump google.golang.org/grpc from 1.71.0 to 1.72.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/307
* Bump github.com/go-openapi/jsonpointer from 0.21.0 to 0.21.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/306
* Bump golang.org/x/time from 0.10.0 to 0.11.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/305
* Bump golang.org/x/oauth2 from 0.28.0 to 0.30.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/304
* Bump sigs.k8s.io/structured-merge-diff/v4 from 4.6.0 to 4.7.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/303
* Bump golang.org/x/net from 0.38.0 to 0.40.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/314
* Bump github.com/grpc-ecosystem/grpc-gateway/v2 from 2.26.1 to 2.26.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/313
* Bump github.com/sagikazarmark/locafero from 0.7.0 to 0.9.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/311
* Bump k8s.io/apimachinery from 0.33.0 to 0.33.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/310
* Bump golang from 1.24.2 to 1.24.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/309
* Bump gomodules.xyz/jsonpatch/v2 from 2.4.0 to 2.5.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/320
* Bump k8s.io/apiextensions-apiserver from 0.33.0 to 0.33.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/316
* Bump go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp from 0.60.0 to 0.61.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/317
* Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc from 1.35.0 to 1.36.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/318


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0...v0.2.0

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.2.0)

---

## [v0.1.0] - 2025-05-12

:tada: First non-alpha release of Netbox-Operator :rocket:

## What's Changed
* Basic operational manual by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/286
* Adapting deploy-netbox.sh to a vCluster deployment by @pablogarciamiranda in https://github.com/netbox-community/netbox-operator/pull/293
* Feature/check keys in parent prefix selector by @bruelea in https://github.com/netbox-community/netbox-operator/pull/295

### Version Bumps
* Bump github.com/onsi/gomega from 1.36.2 to 1.37.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/289
* Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc from 1.34.0 to 1.35.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/291
* Bump go.uber.org/mock from 0.5.0 to 0.5.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/288
* Bump golang.org/x/term from 0.30.0 to 0.31.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/290
* Bump github.com/go-openapi/swag from 0.23.0 to 0.23.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/292
* Bump golangci/golangci-lint-action from 7.0.0 to 8.0.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/302
* Bump github.com/onsi/ginkgo/v2 from 2.23.3 to 2.23.4 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/301
* Bump github.com/prometheus/common from 0.62.0 to 0.63.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/300
* Bump go.mongodb.org/mongo-driver from 1.17.2 to 1.17.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/299
* Bump github.com/prometheus/client_model from 0.6.1 to 0.6.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/298
* Bump github.com/spf13/viper from 1.20.0 to 1.20.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/297

## New Contributors
* @pablogarciamiranda made their first contribution in https://github.com/netbox-community/netbox-operator/pull/286

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.7...v0.1.0

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0)

---

## [v0.1.0-alpha.7] - 2025-04-28

## What's Changed

* Add examples by @jstudler in https://github.com/netbox-community/netbox-operator/pull/264
* Remove duplicate kind cluster creation by @jstudler in https://github.com/netbox-community/netbox-operator/pull/257
* Improve CRD documentation by @jstudler in https://github.com/netbox-community/netbox-operator/pull/263
* Check restoration hash in NetBox before updating resource in NetBox by @bruelea in https://github.com/netbox-community/netbox-operator/pull/285
* Consolidate Status and Condition reporting by @alexandernorth in https://github.com/netbox-community/netbox-operator/pull/265

### Version Bumps
* Bump golang from 1.23.6 to 1.24.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/262
* Bump golangci/golangci-lint-action from 6.5.2 to 7.0.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/278
* Bump github.com/google/cel-go from 0.22.0 to 0.25.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/252
* Bump golang.org/x/sync from 0.11.0 to 0.12.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/258
* Bump go.opentelemetry.io/otel/metric from 1.34.0 to 1.35.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/260
* Bump golang.org/x/tools from 0.29.0 to 0.31.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/259
* Bump golangci/golangci-lint-action from 6.3.1 to 6.5.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/271
* Bump golang.org/x/oauth2 from 0.27.0 to 0.28.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/270
* Bump k8s.io/component-base from 0.32.2 to 0.32.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/269
* Bump github.com/spf13/afero from 1.12.0 to 1.14.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/268
* Bump github.com/spf13/viper from 1.19.0 to 1.20.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/267
* Bump cel.dev/expr from 0.21.2 to 0.22.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/277
* Bump go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp from 0.59.0 to 0.60.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/276
* Bump github.com/onsi/ginkgo/v2 from 2.22.2 to 2.23.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/275
* Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace from 1.34.0 to 1.35.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/274
* Bump actions/setup-go from 5.3.0 to 5.4.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/273
* Bump golangci/golangci-lint-action from 6.5.1 to 6.5.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/272
* Bump k8s.io/apiextensions-apiserver from 0.32.2 to 0.32.3 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/280
* Bump github.com/emicklei/go-restful/v3 from 3.12.1 to 3.12.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/281
* Bump sigs.k8s.io/controller-runtime from 0.20.2 to 0.20.4 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/282
* Bump github.com/prometheus/procfs from 0.15.1 to 0.16.1 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/287



**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.6...v0.1.0-alpha.7

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.7)

---

## [v0.1.0-alpha.6] - 2025-03-04

Improved testing and dependency upgrades
## What's Changed
* Bump google.golang.org/protobuf from 1.36.4 to 1.36.5 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/243
* Bump golang.org/x/time from 0.9.0 to 0.10.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/247
* Bump github.com/klauspost/compress from 1.17.11 to 1.18.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/246
* Bump k8s.io/apiextensions-apiserver from 0.32.1 to 0.32.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/245
* Bump cel.dev/expr from 0.20.0 to 0.21.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/244
* Extend e2e chainsaw tests by @jstudler in https://github.com/netbox-community/netbox-operator/pull/249
* Bump golang.org/x/oauth2 from 0.26.0 to 0.27.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/256
* Bump github.com/prometheus/client_golang from 1.20.5 to 1.21.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/254
* Bump sigs.k8s.io/controller-runtime from 0.20.1 to 0.20.2 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/255
* Bump docker/metadata-action from 5.6.1 to 5.7.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/251
* Bump github.com/google/go-cmp from 0.6.0 to 0.7.0 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/253


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.5...v0.1.0-alpha.6

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.6)

---

## [v0.1.0-alpha.5] - 2025-02-20

Improved testing, dependency upgrades and additional metrics
## What's Changed
* Add e2e tests for Prefix by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/161
* Migrate populating local demo data through database.sql to using NetBox API  by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/177
* Add part-of label to metrics monitor service labelSelector by @bruelea in https://github.com/netbox-community/netbox-operator/pull/204
* upgraded controller tools and removed rbac-proxy for metrics endpoint by @faebr in https://github.com/netbox-community/netbox-operator/pull/205
* Make metricsServerOptions configurable and add documentation by @bruelea in https://github.com/netbox-community/netbox-operator/pull/226
* Add metrics for rest requests towards netbox by @bruelea in https://github.com/netbox-community/netbox-operator/pull/231


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.4...v0.1.0-alpha.5

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.5)

---

## [v0.1.0-alpha.4] - 2024-12-05

## New Feature
* Support reservation of IpRanges by @bruelea in https://github.com/netbox-community/netbox-operator/pull/130

## What's Changed
* Add a prefix with IPv6 addresses to local demo data by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/153


**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.3...v0.1.0-alpha.4

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.4)

---

## [v0.1.0-alpha.1] - 2024-11-25

The first public NetBox Operator release!

NetBox Operator was originally developed internally at Swisscom.

This release marks the debut of NetBox Operator as an open-source project, in collaboration with the NetBox Labs.

# Contributor list

## Swisscom

@bruelea, Ashan Senevirathne @ashanvbs, Joel Studler @jstudler, Alexander North @alexandernorth, Chun-Hung (Henry) Tseng @henrybear327, Fabian Schulz @faebr, Hoang Mai @MaIT-HgA, Pablo Garcia Miranda @pablogarciamiranda, Andreas Forsten, Chema De La Sen, Jitendra Sapariya @jitendrs, Sancho Sergio @CapiSSS, Miltiadis Alexis and many more!​

## NetBox Labs

Mark Coleman, Richard Boucher, Kristopher Beevers, Jeff Gehlbach, Nat Morris

**Full Changelog**: https://github.com/netbox-community/netbox-operator/commits/v0.1.0-alpha.1

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.1)

---

## [v0.1.0-alpha.2] - 2024-11-25

## New Feature

- Implement dynamic parent prefix selection https://github.com/netbox-community/netbox-operator/issues/79

## What's Changed

* Dependabot bumps
* Generate docker images with and without prefixed v for tagged commits by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/76
* Make unlocking more reliable by @jstudler in https://github.com/netbox-community/netbox-operator/pull/59
* Add check for tenant existence by @MaIT-HgA in https://github.com/netbox-community/netbox-operator/pull/72
* Add support for IPv6 by @bruelea in https://github.com/netbox-community/netbox-operator/pull/74
* Bump github.com/klauspost/compress from 1.17.9 to 1.17.10 by @dependabot in https://github.com/netbox-community/netbox-operator/pull/80
* Add age column and fix column order by @jstudler in https://github.com/netbox-community/netbox-operator/pull/81
* Add code of conduct by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/31
* Fix Finalizers by @MaIT-HgA in https://github.com/netbox-community/netbox-operator/pull/78
* Add kustomization to top folder by @bruelea in https://github.com/netbox-community/netbox-operator/pull/98
* Reduce development-related content in README and move them to CONTRIBUTING.md by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/101
* Fix broken link in README by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/106
* removed group name from existing crd's by @faebr in https://github.com/netbox-community/netbox-operator/pull/114
* Update golang version to 1.23.3 by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/92
* Add license header lint rules by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/60
* Fix: add logic to update site of prefix in netbox by @bruelea in https://github.com/netbox-community/netbox-operator/pull/107
* Make site immutable by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/127
* Implement dynamic selection of parent prefix from a set of custom fields by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/90

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.1...v0.1.0-alpha.2

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.2)

---

## [v0.1.0-alpha.3] - 2024-11-25

## What's Changed
* Add support for built-in field `family` in `ParentPrefixSelector` by @henrybear327 in https://github.com/netbox-community/netbox-operator/pull/138

**Full Changelog**: https://github.com/netbox-community/netbox-operator/compare/v0.1.0-alpha.2...v0.1.0-alpha.3

[Full Release](https://github.com/netbox-community/netbox-operator/releases/tag/v0.1.0-alpha.3)

---


