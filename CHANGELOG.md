## [1.1.3](https://github.com/bxrne/logmgr/compare/v1.1.2...v1.1.3) (2025-06-01)


### Documentation

* Update README.md ([b7b0d09](https://github.com/bxrne/logmgr/commit/b7b0d09824cd076e48c5b5d79ef6d1853beb1eda))


### Continuous Integration

* Potential fix for code scanning alert no. 1: Workflow does not contain permissions ([0702883](https://github.com/bxrne/logmgr/commit/070288320cb8da64034d8a13b5f69531e7f078dd))
* Potential fix for code scanning alert no. 2: Workflow does not contain permissions ([50b2923](https://github.com/bxrne/logmgr/commit/50b292387f32c943ffeed74cbce2ededcf0b89a8))
* Potential fix for code scanning alert no. 3: Workflow does not contain permissions ([f749571](https://github.com/bxrne/logmgr/commit/f749571a7a96b807d742b0b4756960de87225078))
* Potential fix for code scanning alert no. 4: Workflow does not contain permissions ([e1e4bf5](https://github.com/bxrne/logmgr/commit/e1e4bf50f1811e3c52e187d83021b1ea282211bb))

## [1.1.2](https://github.com/bxrne/logmgr/compare/v1.1.1...v1.1.2) (2025-06-01)


### Code Refactoring

* improve log entry processing and enhance README with performance metrics ([5420420](https://github.com/bxrne/logmgr/commit/54204203460c2a4e63be2dbaeeb463a4524a4a50))

## [1.1.1](https://github.com/bxrne/logmgr/compare/v1.1.0...v1.1.1) (2025-06-01)


### Code Refactoring

* optimize entry writing in sinks by pre-allocating batch buffers ([ef7a470](https://github.com/bxrne/logmgr/commit/ef7a470636a282859b2d0e569d4770d56f467ad2))


### Continuous Integration

* Create codeql.yml ([34694fc](https://github.com/bxrne/logmgr/commit/34694fc4f314a6a69c72b9e8707a1c0c83caad00))
* update golangci-lint version format in CI workflow ([3481ac7](https://github.com/bxrne/logmgr/commit/3481ac7e0731145ff2a76454287a2d42e0400617))

## [1.1.0](https://github.com/bxrne/logmgr/compare/v1.0.2...v1.1.0) (2025-06-01)


### Features

* add SonarCloud configuration and project properties for Go project analysis ([41edaf1](https://github.com/bxrne/logmgr/commit/41edaf10679a362d04bfc70e4d3c8b804da0c961))

## [1.0.2](https://github.com/bxrne/logmgr/compare/v1.0.1...v1.0.2) (2025-06-01)


### Documentation

* update README to enhance badge visibility and include coverage metrics ([2ab0faa](https://github.com/bxrne/logmgr/commit/2ab0faae79b93c57a869491f2cac148bb543121c))


### Continuous Integration

* remove push trigger and update golangci-lint version in CI workflow ([2bc8d2f](https://github.com/bxrne/logmgr/commit/2bc8d2f04ebec1998b3aa4b47f4c85d937f0e585))

## [1.0.1](https://github.com/bxrne/logmgr/compare/v1.0.0...v1.0.1) (2025-06-01)


### Code Refactoring

* enhance error handling in sinks and benchmarks, improve resource management ([7d9105e](https://github.com/bxrne/logmgr/commit/7d9105e02ac38958df067738671e9382299f5711))
* improve error handling during sink closure in tests ([0d3bfdb](https://github.com/bxrne/logmgr/commit/0d3bfdb2398b6b414007114e618a3f5143130a39))


### Miscellaneous Chores

* add Makefile for build automation, include LICENSE file, and expand .gitignore for development artifacts ([17dc458](https://github.com/bxrne/logmgr/commit/17dc4583e1ad2473a096d2f5b1d9c64956c0ed49))
* update Go version to 1.24 in go.mod and CI workflows ([f1b566b](https://github.com/bxrne/logmgr/commit/f1b566b7f51c056b33aa7c40989d8e5c534b7fb3))
* update golangci-lint configuration to streamline linter settings and remove deprecated options ([61cba0a](https://github.com/bxrne/logmgr/commit/61cba0adc9cdeec5c0860f80994d165d349c661c))


### Tests

* adjust log level in stream sink tests based on sink type ([b557b92](https://github.com/bxrne/logmgr/commit/b557b9214d5079a465bd768b0966d245428869fb))
* enhance stdout redirection in BenchmarkConsoleSink to improve error handling and resource management ([ac38b46](https://github.com/bxrne/logmgr/commit/ac38b46cd7db35b0f2460c44400d71d3e818bcfe))


### Continuous Integration

* Create dependabot.yml ([7ccb199](https://github.com/bxrne/logmgr/commit/7ccb1998edd8d3921b3d859fb0aebbb77a72c5eb))
* update release configuration to include documentation and build types for patch releases and documented some tests too ([418543f](https://github.com/bxrne/logmgr/commit/418543f93111844de910ecfc3bde2603d688ccea))

## 1.0.0 (2025-06-01)


### Features

* initial implementation with a ring buffer and multiple sinks with benchmarks and some intial testing ([2f94a5a](https://github.com/bxrne/logmgr/commit/2f94a5a9b7757f6d7ed56470460b91763883f5ca))


### Code Refactoring

* improve shutdown handling in AsyncFileSink and Worker, add tests for logging functionality and sink behavior ([df9f1f8](https://github.com/bxrne/logmgr/commit/df9f1f8a1c4cb5502620a0dcb86623ce96038560))
