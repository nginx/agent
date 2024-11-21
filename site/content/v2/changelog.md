---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [v2.39.0](https://github.com/nginx/agent/releases/tag/v2.39.0)

### 🌟 Highlights

- Remove official docker images & move testing images to test folder by [@aphralG](https://github.com/aphralG) in [#838](https://github.com/nginx/agent/pull/838)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Race conditions fixes by [@oliveromahony](https://github.com/oliveromahony) in [#810](https://github.com/nginx/agent/pull/810)
- fix r30 pipeline failures by [@oliveromahony](https://github.com/oliveromahony) in [#844](https://github.com/nginx/agent/pull/844)
- Fixed make target pointing at wrong Dockerfile and renamed others to be consistent by [@oliveromahony](https://github.com/oliveromahony) in [#857](https://github.com/nginx/agent/pull/857)
- Fix broken links causing deployment failures by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#863](https://github.com/nginx/agent/pull/863)
- Fix NGINX OSS integration tests by [@dhurley](https://github.com/dhurley) in [#888](https://github.com/nginx/agent/pull/888)
- Fix docs docker failing without git context by [@nginx-jack](https://github.com/nginx-jack) in [#892](https://github.com/nginx/agent/pull/892)

### 📝 Documentation

We have made the following updates to the documentation:

- Add automatic changelog generation in release workflow by [@spencerugbo](https://github.com/spencerugbo) in [#784](https://github.com/nginx/agent/pull/784)
- Add CLA bot workflow by [@lucacome](https://github.com/lucacome) in [#828](https://github.com/nginx/agent/pull/828)
- Refactor docker images by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [#841](https://github.com/nginx/agent/pull/841)
- Docs: Add hugo version check and theme update to Makefile by [@nginx-jack](https://github.com/nginx-jack) in [#869](https://github.com/nginx/agent/pull/869)
- Change casing of docs makefile to Makefile by [@nginx-jack](https://github.com/nginx-jack) in [#884](https://github.com/nginx/agent/pull/884)
- docs: enableGitInfo config and docs-action bump by [@nginx-jack](https://github.com/nginx-jack) in [#886](https://github.com/nginx/agent/pull/886)
- Change go version to latest go 1.23.2 by [@oliveromahony](https://github.com/oliveromahony) in [#889](https://github.com/nginx/agent/pull/889)
- Remove link to github dockerfiles by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [#897](https://github.com/nginx/agent/pull/897)
- Docs: Update link to 3rd party site by [@nginx-aoife](https://github.com/nginx-aoife) in [#898](https://github.com/nginx/agent/pull/898)
- Update the changelog for v2.38 by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#901](https://github.com/nginx/agent/pull/901)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Set log level to debug for inetegration tests by [@aphralG](https://github.com/aphralG) in [#826](https://github.com/nginx/agent/pull/826)
- updated runc dependency highlighted in security scan scan by [@oliveromahony](https://github.com/oliveromahony) in [#842](https://github.com/nginx/agent/pull/842)
- Update CODEOWNERS by [@oCHRISo](https://github.com/oCHRISo) in [#851](https://github.com/nginx/agent/pull/851)
- Check version command output by [@aphralG](https://github.com/aphralG) in [#853](https://github.com/nginx/agent/pull/853)
- Bump NGINX plus go client version from v1 to v2 by [@dhurley](https://github.com/dhurley) in [#879](https://github.com/nginx/agent/pull/879)
- Allowlist Error Messages by [@aphralG](https://github.com/aphralG) in [#907](https://github.com/nginx/agent/pull/907)

---
## Release [v2.38.0](https://github.com/nginx/agent/releases/tag/v2.38.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix broken URLS in docs by [@nginx-aoife](https://github.com/nginx-aoife) in [#796](https://github.com/nginx/agent/pull/796)
- fix name of deprecated flag by [@aphralG](https://github.com/aphralG) in [#811](https://github.com/nginx/agent/pull/811)
- Fix make image targets by [@dhurley](https://github.com/dhurley) in [#812](https://github.com/nginx/agent/pull/812)
- Fix debian oss image by [@dhurley](https://github.com/dhurley) in [#819](https://github.com/nginx/agent/pull/819)

### 📝 Documentation

We have made the following updates to the documentation:

- docs: update GPG keys by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#776](https://github.com/nginx/agent/pull/776)
- Add new docker images to v2 pipeline for integration testing by [@oliveromahony](https://github.com/oliveromahony) in [#756](https://github.com/nginx/agent/pull/756)
- Update website changelog for v2.37.0 by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#790](https://github.com/nginx/agent/pull/790)
- Pass on custom error log path at the time of validating config by [@achawla2012](https://github.com/achawla2012) in [#774](https://github.com/nginx/agent/pull/774)
- Remove blocking calls in metrics framework by [@oliveromahony](https://github.com/oliveromahony) in [#788](https://github.com/nginx/agent/pull/788)
- Update broken URL in installation-plus.md by [@nginx-aoife](https://github.com/nginx-aoife) in [#808](https://github.com/nginx/agent/pull/808)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- add new plus docker images to v2 pipeline by [@aphralG](https://github.com/aphralG) in [#779](https://github.com/nginx/agent/pull/779)
- Add MaxRecvMsgSize and MaxSendMsgSize to client and server options by [@oliveromahony](https://github.com/oliveromahony) in [#795](https://github.com/nginx/agent/pull/795)
- added leak tests for agent v2 by [@oliveromahony](https://github.com/oliveromahony) in [#807](https://github.com/nginx/agent/pull/807)

---
## Release [v2.37.0](https://github.com/nginx/agent/releases/tag/v2.37.0)

### 🚀 Features

This release introduces the following new features:

- feat: Update the changelog by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#753](https://github.com/nginx/agent/pull/753)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Prevent writing outside allowed directories list from a config payload with actions by [@oliveromahony](https://github.com/oliveromahony) in [#766](https://github.com/nginx/agent/pull/766)
- The letter v is now always prepended to output of -v by [@olli-holmala](https://github.com/olli-holmala) in [#751](https://github.com/nginx/agent/pull/751)
- Fix backoff to drop Metrics Reports from buffer after max_elapsed_time has been reached by [@oliveromahony](https://github.com/oliveromahony) in [#752](https://github.com/nginx/agent/pull/752)
- Fix Post Install Script Issues by [@spencerugbo](https://github.com/spencerugbo) in [#739](https://github.com/nginx/agent/pull/739)
- docs: fix github links in changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#770](https://github.com/nginx/agent/pull/770)
- Fix post install script for when no nginx instance is installed by [@dhurley](https://github.com/dhurley) in [#773](https://github.com/nginx/agent/pull/773)

### 📝 Documentation

We have made the following updates to the documentation:

- Upgrade prometheus exporter version to latest by [@oliveromahony](https://github.com/oliveromahony) in [#749](https://github.com/nginx/agent/pull/749)
- Add badges for Go version, release, license, contributions, and Slack… by [@oCHRISo](https://github.com/oCHRISo) in [#763](https://github.com/nginx/agent/pull/763)
- Add instructions for Amazon Linux 2023 by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [#759](https://github.com/nginx/agent/pull/759)
- Add docs-build-push github workflow by [@nginx-jack](https://github.com/nginx-jack) in [#765](https://github.com/nginx/agent/pull/765)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Increase timeout period for collecting metrics by [@oliveromahony](https://github.com/oliveromahony) in [#755](https://github.com/nginx/agent/pull/755)

---
## Release [v2.36.1](https://github.com/nginx/agent/releases/tag/v2.36.1)

### 🌟 Highlights

- Upgrade crossplane version to prevent Agent from rolling back in the case of valid NGINX configurations by [@oliveromahony](https://github.com/oliveromahony) in [#746](https://github.com/nginx/agent/pull/746)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Added version regex to parse the logs to see if matches vsemvar format by [@oliveromahony](https://github.com/oliveromahony) in [#747](https://github.com/nginx/agent/pull/747)

---
## Release [v2.36.0](https://github.com/nginx/agent/releases/tag/v2.36.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix incorrect bold tag in heading by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [#715](https://github.com/nginx/agent/pull/715)
- URL fix for building docker image in README.md by [@y82](https://github.com/y82) in [#720](https://github.com/nginx/agent/pull/720)
- Fix for version by [@oliveromahony](https://github.com/oliveromahony) in [#732](https://github.com/nginx/agent/pull/732)

### 📝 Documentation

We have made the following updates to the documentation:

- More flexible container images for the official images by [@oliveromahony](https://github.com/oliveromahony) in [#729](https://github.com/nginx/agent/pull/729)
- Update configuration examples by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [#731](https://github.com/nginx/agent/pull/731)
- updated github.com/rs/cors version by [@oliveromahony](https://github.com/oliveromahony) in [#735](https://github.com/nginx/agent/pull/735)
- docs: update changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#736](https://github.com/nginx/agent/pull/736)
- Upgrade crossplane by [@oliveromahony](https://github.com/oliveromahony) in [#737](https://github.com/nginx/agent/pull/737)

---
## Release [v2.35.1](https://github.com/nginx/agent/releases/tag/v2.35.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: add deduplication for the same ssl cert metadata by [@mattdesmarais](https://github.com/mattdesmarais) [@oliveromahony](https://github.com/oliveromahony) in [#716](https://github.com/nginx/agent/pull/716)
- Fix release workflow by [@dhurley](https://github.com/dhurley) in [#724](https://github.com/nginx/agent/pull/724)

### 📝 Documentation

We have made the following updates to the documentation:

- Update environment variables from NMS to NGINX_AGENT by [@spencerugbo](https://github.com/spencerugbo) in [#710](https://github.com/nginx/agent/pull/710)
- Update the flag & environment table callouts by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#712](https://github.com/nginx/agent/pull/712)
- updated golang version to 1.22 by [@oliveromahony](https://github.com/oliveromahony) in [#717](https://github.com/nginx/agent/pull/717)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- More detailed test for env variables migration by [@oliveromahony](https://github.com/oliveromahony) in [#709](https://github.com/nginx/agent/pull/709)

---
## Release [v2.35.0](https://github.com/nginx/agent/releases/tag/v2.35.0)

### 🌟 Highlights

- R32 operating system support parity by [@oliveromahony](https://github.com/oliveromahony) in [#708](https://github.com/nginx/agent/pull/708)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Change environment prefix from nms to nginx_agent by [@spencerugbo](https://github.com/spencerugbo) in [#706](https://github.com/nginx/agent/pull/706)

### 📝 Documentation

We have made the following updates to the documentation:

- Consolidated CLI flag and Env Var sections by [@travisamartin](https://github.com/travisamartin) in [#701](https://github.com/nginx/agent/pull/701)
- Add Ubuntu Noble 24.04 LTS support by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#682](https://github.com/nginx/agent/pull/682)

---
## Release [v2.34.1](https://github.com/nginx/agent/releases/tag/v2.34.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix metrics reporter retry logic by [@dhurley](https://github.com/dhurley) in [#700](https://github.com/nginx/agent/pull/700)

### 📝 Documentation

We have made the following updates to the documentation:

- Update changelog for release 2.34 by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#693](https://github.com/nginx/agent/pull/693)

---
## Release [v2.34.0](https://github.com/nginx/agent/releases/tag/v2.34.0)

### 🌟 Highlights

- Bump the version of net package in golang by [@oliveromahony](https://github.com/oliveromahony) in [#645](https://github.com/nginx/agent/pull/645)

- Add health check endpoint by [@dhurley](https://github.com/dhurley) in [#665](https://github.com/nginx/agent/pull/665)

- Add pending health status by [@dhurley](https://github.com/dhurley) in [#672](https://github.com/nginx/agent/pull/672)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: fix titles case by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#674](https://github.com/nginx/agent/pull/674)
- Fix oracle linux integration test by [@dhurley](https://github.com/dhurley) in [#676](https://github.com/nginx/agent/pull/676)

### 📝 Documentation

We have made the following updates to the documentation:

- chore: add 2.33.0 changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#622](https://github.com/nginx/agent/pull/622)
- Change environment variable list to table with CLI references by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#689](https://github.com/nginx/agent/pull/689)
- Add health checks documentation by [@dhurley](https://github.com/dhurley) in [#673](https://github.com/nginx/agent/pull/673)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Keep looking for master process by [@spencerugbo](https://github.com/spencerugbo) in [#617](https://github.com/nginx/agent/pull/617)
- Bump docker dependency to version v24.0.9 by [@dhurley](https://github.com/dhurley) in [#626](https://github.com/nginx/agent/pull/626)
- Bump the version of github.com/opencontainers/runc dependency by [@dhurley](https://github.com/dhurley) in [#657](https://github.com/nginx/agent/pull/657)
- Remove unnecessary freebsd logic for finding process executable by [@dhurley](https://github.com/dhurley) in [#668](https://github.com/nginx/agent/pull/668)
- Add additional checks in chunking functionality by [@dhurley](https://github.com/dhurley) in [#671](https://github.com/nginx/agent/pull/671)

---
## Release [v2.33.0](https://github.com/nginx/agent/releases/tag/v2.33.0)

### 🚀 Features

This release introduces the following new features:

- feat: Add Support for NAP 5 by [@edarzins](https://github.com/edarzins) in [#604](https://github.com/nginx/agent/pull/604)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix  nfpm.yaml for apk packages by [@dhurley](https://github.com/dhurley) in [#597](https://github.com/nginx/agent/pull/597)
- fix unit test by [@oliveromahony](https://github.com/oliveromahony) in [#607](https://github.com/nginx/agent/pull/607)
- Fix user workflow performance tests by [@dhurley](https://github.com/dhurley) in [#612](https://github.com/nginx/agent/pull/612)
- fix Advanced Metrics  by [@aphralG](https://github.com/aphralG) in [#598](https://github.com/nginx/agent/pull/598)

### 📝 Documentation

We have made the following updates to the documentation:

- chore: Add the 2.32.2 Changelog to the docs website by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#601](https://github.com/nginx/agent/pull/601)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Bump the version of protobuf by [@oliveromahony](https://github.com/oliveromahony) in [#602](https://github.com/nginx/agent/pull/602)
- replace duplicate isContainer call by [@oliveromahony](https://github.com/oliveromahony) in [#596](https://github.com/nginx/agent/pull/596)
- Add logging to NGINX API http requests by [@dhurley](https://github.com/dhurley) in [#605](https://github.com/nginx/agent/pull/605)

