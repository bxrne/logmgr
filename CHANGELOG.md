## [1.1.5](https://github.com/bxrne/logmgr/compare/v1.1.4...v1.1.5) (2025-06-01)


### Bug Fixes

* Improve error handling and closure logic in file sinks ([e8cb78e](https://github.com/bxrne/logmgr/commit/e8cb78e23b00f8753591524a78e41636967f9254))


### Code Refactoring

* Clean up benchmark tests by removing unused helper function and improving logger sync handling ([48aa24e](https://github.com/bxrne/logmgr/commit/48aa24ed966c71e12b2e95e26d68647c6ccea429))
* Refactor benchmark tests to eliminate global logger usage ([9702a15](https://github.com/bxrne/logmgr/commit/9702a15e9ee4cb3cb3796c2288dd20cee49e2dbe))


### Documentation

* Add comprehensive benchmarks and design philosophy for logmgr ([cbb0848](https://github.com/bxrne/logmgr/commit/cbb08483c45830e72100403a72c4947f95702806))
* Remove performance metrics section from README ([0859a7a](https://github.com/bxrne/logmgr/commit/0859a7a0a5d8fa3bf47442e644223aa73aa54254))


### Styles

* remove benchmark ([4099c35](https://github.com/bxrne/logmgr/commit/4099c350054ea786dea6baec24be719910a655b5))


### Tests

* Enhance file sink tests with OS-specific invalid paths and ensure proper resource closure ([38cfe66](https://github.com/bxrne/logmgr/commit/38cfe66afd88e8ffa1f70e0dd260932ce1487f84))
* Update file sink tests to use OS-specific invalid paths for improved reliability ([3a7ab24](https://github.com/bxrne/logmgr/commit/3a7ab24c446359f88e3099f02caccc73b9161f20))


### Continuous Integration

* Update CI workflow to run tests based on OS type ([91657d6](https://github.com/bxrne/logmgr/commit/91657d693a075ec2412baf9e201e9d6a20fdae3f))

## [1.1.4](https://github.com/bxrne/logmgr/compare/v1.1.3...v1.1.4) (2025-06-01)


### Bug Fixes

* Ensure proper resource closure in test case ([5215d8c](https://github.com/bxrne/logmgr/commit/5215d8c9402c3c7ca61eebd1ac81e3a3c03c4937))
* Handle nil worker in global logger reset ([bfeca2b](https://github.com/bxrne/logmgr/commit/bfeca2bc7a0cccaaba9cee3dbcfea393220eae6c))
* Prevent double closure of underlying file in AsyncFileSink ([343098a](https://github.com/bxrne/logmgr/commit/343098a169d942645726c0f213aaee21c835d452))


### Styles

* Add example image to README and enhance test coverage ([e6c6f80](https://github.com/bxrne/logmgr/commit/e6c6f8055f7d068c83c42bf1c7cd1f2c4c4a98c2))


### Miscellaneous Chores

* Remove backup GolangCI configuration file ([0d289de](https://github.com/bxrne/logmgr/commit/0d289de19970a84cea6988d4ab36690e443c47a1))
* Simplify GolangCI configuration by removing unused presets ([89347a1](https://github.com/bxrne/logmgr/commit/89347a17ab0f351aaa06ae4ff930d7e113448302))
* Update GolangCI configuration and improve test error handling ([d43f460](https://github.com/bxrne/logmgr/commit/d43f460f295bc48b815aea18a071e77650a382c2))


### Continuous Integration

* Refactor CI workflow to streamline steps and update GolangCI lint action ([b1125db](https://github.com/bxrne/logmgr/commit/b1125db12a0f8514f92a3ddb6af3e65b010be0c9))
* Remove unnecessary permissions section from CI workflow ([11ed906](https://github.com/bxrne/logmgr/commit/11ed90674d908befc6213885bc93e837b1f22089))
* Update CI workflow to install dependencies and run GolangCI lint ([f9900a0](https://github.com/bxrne/logmgr/commit/f9900a06c39ea95c661d1ddb66c51eb5f47ddaf5))
* Update GolangCI lint installation to use the latest version ([54e644c](https://github.com/bxrne/logmgr/commit/54e644c36777f6c41ce3a1da73514ee726a871f5))

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
