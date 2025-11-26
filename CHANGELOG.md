## [1.4.0](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.3.0...v1.4.0) (2025-11-26)


### Features

* Add default tags CLI option [K8SPCORE-1707] ([e9bd1f3](https://github.com/goto-opensource/k8s-aws-operator/commit/e9bd1f3215dce068ab790c3fd3eff4f8e4fc19a3))

## [1.3.0](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.2.3...v1.3.0) (2024-08-20)


### Features

* allow deployment annotations in Helm Chart ([5dd46ef](https://github.com/goto-opensource/k8s-aws-operator/commit/5dd46efd3a5230f8547d55480acf5377121ca472))

## [1.2.3](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.2.2...v1.2.3) (2024-08-19)


### Bug Fixes

* strip digest from image tag in version label in Helm chart ([#23](https://github.com/goto-opensource/k8s-aws-operator/issues/23)) ([b6a9fa0](https://github.com/goto-opensource/k8s-aws-operator/commit/b6a9fa085b794271864dd26a81c2ef7e48393d47))

## [1.2.2](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.2.1...v1.2.2) (2024-08-19)


### Bug Fixes

* use non-caching client to get pod in `getPodPrivateIP` also in EIP controller ([#22](https://github.com/goto-opensource/k8s-aws-operator/issues/22)) ([9db1162](https://github.com/goto-opensource/k8s-aws-operator/commit/9db1162d5aa34dcbbdc96fdad8572ea2cdf30170))

## [1.2.1](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.2.0...v1.2.1) (2024-07-01)


### Bug Fixes

* use non-caching client to get pod in `getPodPrivateIP` in ENI controller ([#21](https://github.com/goto-opensource/k8s-aws-operator/issues/21)) ([317b48e](https://github.com/goto-opensource/k8s-aws-operator/commit/317b48e32416db3115bbc10a0b8c8cb27ca1412d))

## [1.2.0](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.1.1...v1.2.0) (2024-04-30)


### Features

* add recommended labels to the Helm Chart ([#18](https://github.com/goto-opensource/k8s-aws-operator/issues/18)) ([17e1f37](https://github.com/goto-opensource/k8s-aws-operator/commit/17e1f37025385926c12f1ba6566e02b986ffe1be))

## [1.1.1](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.1.0...v1.1.1) (2023-10-25)


### Bug Fixes

* **chart:** indention of pod scheduling configurations ([72ea0df](https://github.com/goto-opensource/k8s-aws-operator/commit/72ea0dff373c191249719d6cc95cf0d35386dc90))

## [1.1.0](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.0.1...v1.1.0) (2023-07-04)


### Features

* adding EIPAssociation CRD and controller to allow static EIP unassignment ([#16](https://github.com/goto-opensource/k8s-aws-operator/issues/16)) ([4ffd565](https://github.com/goto-opensource/k8s-aws-operator/commit/4ffd565aa5f834b59de5f80aca9db9a492eecac8))

## [1.0.1](https://github.com/goto-opensource/k8s-aws-operator/compare/v1.0.0...v1.0.1) (2023-05-09)


### Bug Fixes

* use newer buildx in workflows to get correct image digest for Helm chart app version ([#17](https://github.com/goto-opensource/k8s-aws-operator/issues/17)) ([d0e5acb](https://github.com/goto-opensource/k8s-aws-operator/commit/d0e5acb492c873603486f92ded4fc7ef4b2a811d))

## [1.0.0](https://github.com/goto-opensource/k8s-aws-operator/compare/v0.0.5...v1.0.0) (2023-04-05)


### ⚠ BREAKING CHANGES

* not really a breaking change, just bumping to v1.0.0

### Features

* add semantic-release and Helm chart; push Docker image and Helm chart to ghcr.io ([5ef6c3e](https://github.com/goto-opensource/k8s-aws-operator/commit/5ef6c3efb7908c4be524d29b4ac7042d16a62d18))
