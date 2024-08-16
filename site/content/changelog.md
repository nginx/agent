---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [v2.37.0](https//github.com/nginx/agent/releases/tag/v2.37.0)

### 🚀 Features

This release introduces the following new features:

- feat: Update the changelog by [@ADubhlaoich](https://github.com/ADubhlaoich) in [agent#753](https://github.com/nginx/agent/pull/753)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Prevent writing outside allowed directories list from a config payload with actions by [@oliveromahony](https://github.com/oliveromahony) in [agent#766](https://github.com/nginx/agent/pull/766)
- The letter v is now always prepended to output of -v by [@olli-holmala](https://github.com/olli-holmala) in [agent#751](https://github.com/nginx/agent/pull/751)
- Fix backoff to drop Metrics Reports from buffer after max_elapsed_time has been reached by [@oliveromahony](https://github.com/oliveromahony) in [agent#752](https://github.com/nginx/agent/pull/752)
- Fix Post Install Script Issues by [@spencerugbo](https://github.com/spencerugbo) in [agent#739](https://github.com/nginx/agent/pull/739)
- docs: fix github links in changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#770](https://github.com/nginx/agent/pull/770)
- Fix post install script for when no nginx instance is installed by [@dhurley](https://github.com/dhurley) in [agent#773](https://github.com/nginx/agent/pull/773)

### 📝 Documentation

We have made the following updates to the documentation:

- Upgrade prometheus exporter version to latest by [@oliveromahony](https://github.com/oliveromahony) in [agent#749](https://github.com/nginx/agent/pull/749)
- Add badges for Go version, release, license, contributions, and Slack… by [@oCHRISo](https://github.com/oCHRISo) in [agent#763](https://github.com/nginx/agent/pull/763)
- Add instructions for Amazon Linux 2023 by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [agent#759](https://github.com/nginx/agent/pull/759)
- Add docs-build-push github workflow by [@nginx-jack](https://github.com/nginx-jack) in [agent#765](https://github.com/nginx/agent/pull/765)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Increase timeout period for collecting metrics by [@oliveromahony](https://github.com/oliveromahony) in [agent#755](https://github.com/nginx/agent/pull/755)

---
## Release [v2.36.1](https//github.com/nginx/agent/releases/tag/v2.36.1)

### 🌟 Highlights

- Upgrade crossplane version to prevent Agent from rolling back in the case of valid NGINX configurations by [@oliveromahony](https://github.com/oliveromahony) in [agent#746](https://github.com/nginx/agent/pull/746)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Added version regex to parse the logs to see if matches vsemvar format by [@oliveromahony](https://github.com/oliveromahony) in [agent#747](https://github.com/nginx/agent/pull/747)

---
## Release [v2.36.0](https//github.com/nginx/agent/releases/tag/v2.36.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix incorrect bold tag in heading by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [agent#715](https://github.com/nginx/agent/pull/715)
- URL fix for building docker image in README.md by [@y82](https://github.com/y82) in [agent#720](https://github.com/nginx/agent/pull/720)
- Fix for version by [@oliveromahony](https://github.com/oliveromahony) in [agent#732](https://github.com/nginx/agent/pull/732)

### 📝 Documentation

We have made the following updates to the documentation:

- More flexible container images for the official images by [@oliveromahony](https://github.com/oliveromahony) in [agent#729](https://github.com/nginx/agent/pull/729)
- Update configuration examples by [@nginx-seanmoloney](https://github.com/nginx-seanmoloney) in [agent#731](https://github.com/nginx/agent/pull/731)
- updated github.com/rs/cors version by [@oliveromahony](https://github.com/oliveromahony) in [agent#735](https://github.com/nginx/agent/pull/735)
- docs: update changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#736](https://github.com/nginx/agent/pull/736)
- Upgrade crossplane by [@oliveromahony](https://github.com/oliveromahony) in [agent#737](https://github.com/nginx/agent/pull/737)

---
## Release [v2.35.1](https//github.com/nginx/agent/releases/tag/v2.35.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: add deduplication for the same ssl cert metadata by [@mattdesmarais](https://github.com/mattdesmarais) [@oliveromahony](https://github.com/oliveromahony) in [agent#716](https://github.com/nginx/agent/pull/716)
- Fix release workflow by [@dhurley](https://github.com/dhurley) in [agent#724](https://github.com/nginx/agent/pull/724)

### 📝 Documentation

We have made the following updates to the documentation:

- Update environment variables from NMS to NGINX_AGENT by [@spencerugbo](https://github.com/spencerugbo) in [agent#710](https://github.com/nginx/agent/pull/710)
- Update the flag & environment table callouts by [@ADubhlaoich](https://github.com/ADubhlaoich) in [agent#712](https://github.com/nginx/agent/pull/712)
- updated golang version to 1.22 by [@oliveromahony](https://github.com/oliveromahony) in [agent#717](https://github.com/nginx/agent/pull/717)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- More detailed test for env variables migration by [@oliveromahony](https://github.com/oliveromahony) in [agent#709](https://github.com/nginx/agent/pull/709)

---
## Release [v2.35.0](https//github.com/nginx/agent/releases/tag/v2.35.0)

### 🌟 Highlights

- R32 operating system support parity by [@oliveromahony](https://github.com/oliveromahony) in [agent#708](https://github.com/nginx/agent/pull/708)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Change environment prefix from nms to nginx_agent by [@spencerugbo](https://github.com/spencerugbo) in [agent#706](https://github.com/nginx/agent/pull/706)

### 📝 Documentation

We have made the following updates to the documentation:

- Consolidated CLI flag and Env Var sections by [@travisamartin](https://github.com/travisamartin) in [agent#701](https://github.com/nginx/agent/pull/701)
- Add Ubuntu Noble 24.04 LTS support by [@Dean-Coakley](https://github.com/Dean-Coakley) in [agent#682](https://github.com/nginx/agent/pull/682)

---
## Release [v2.34.1](https//github.com/nginx/agent/releases/tag/v2.34.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix metrics reporter retry logic by [@dhurley](https://github.com/dhurley) in [agent#700](https://github.com/nginx/agent/pull/700)

### 📝 Documentation

We have made the following updates to the documentation:

- Update changelog for release 2.34 by [@ADubhlaoich](https://github.com/ADubhlaoich) in [agent#693](https://github.com/nginx/agent/pull/693)

---
## Release [v2.34.0](https//github.com/nginx/agent/releases/tag/v2.34.0)

### 🌟 Highlights

- Bump the version of net package in golang by [@oliveromahony](https://github.com/oliveromahony) in [agent#645](https://github.com/nginx/agent/pull/645)

- Add health check endpoint by [@dhurley](https://github.com/dhurley) in [agent#665](https://github.com/nginx/agent/pull/665)

- Add pending health status by [@dhurley](https://github.com/dhurley) in [agent#672](https://github.com/nginx/agent/pull/672)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: fix titles case by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#674](https://github.com/nginx/agent/pull/674)
- Fix oracle linux integration test by [@dhurley](https://github.com/dhurley) in [agent#676](https://github.com/nginx/agent/pull/676)

### 📝 Documentation

We have made the following updates to the documentation:

- chore: add 2.33.0 changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#622](https://github.com/nginx/agent/pull/622)
- Change environment variable list to table with CLI references by [@ADubhlaoich](https://github.com/ADubhlaoich) in [agent#689](https://github.com/nginx/agent/pull/689)
- Add health checks documentation by [@dhurley](https://github.com/dhurley) in [agent#673](https://github.com/nginx/agent/pull/673)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Keep looking for master process by [@spencerugbo](https://github.com/spencerugbo) in [agent#617](https://github.com/nginx/agent/pull/617)
- Bump docker dependency to version v24.0.9 by [@dhurley](https://github.com/dhurley) in [agent#626](https://github.com/nginx/agent/pull/626)
- Bump the version of github.com/opencontainers/runc dependency by [@dhurley](https://github.com/dhurley) in [agent#657](https://github.com/nginx/agent/pull/657)
- Remove unnecessary freebsd logic for finding process executable by [@dhurley](https://github.com/dhurley) in [agent#668](https://github.com/nginx/agent/pull/668)
- Add additional checks in chunking functionality by [@dhurley](https://github.com/dhurley) in [agent#671](https://github.com/nginx/agent/pull/671)

---
## Release [v2.33.0](https//github.com/nginx/agent/releases/tag/v2.33.0)

### 🚀 Features

This release introduces the following new features:

- feat: Add Support for NAP 5 by [@edarzins](https://github.com/edarzins) in [agent#604](https://github.com/nginx/agent/pull/604)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix  nfpm.yaml for apk packages by [@dhurley](https://github.com/dhurley) in [agent#597](https://github.com/nginx/agent/pull/597)
- fix unit test by [@oliveromahony](https://github.com/oliveromahony) in [agent#607](https://github.com/nginx/agent/pull/607)
- Fix user workflow performance tests by [@dhurley](https://github.com/dhurley) in [agent#612](https://github.com/nginx/agent/pull/612)
- fix Advanced Metrics  by [@aphralG](https://github.com/aphralG) in [agent#598](https://github.com/nginx/agent/pull/598)

### 📝 Documentation

We have made the following updates to the documentation:

- chore: Add the 2.32.2 Changelog to the docs website by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#601](https://github.com/nginx/agent/pull/601)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Bump the version of protobuf by [@oliveromahony](https://github.com/oliveromahony) in [agent#602](https://github.com/nginx/agent/pull/602)
- replace duplicate isContainer call by [@oliveromahony](https://github.com/oliveromahony) in [agent#596](https://github.com/nginx/agent/pull/596)
- Add logging to NGINX API http requests by [@dhurley](https://github.com/dhurley) in [agent#605](https://github.com/nginx/agent/pull/605)

---
## Release [v2.32.2](https//github.com/nginx/agent/releases/tag/v2.32.2)

### 🌟 Highlights

- This release fixes an issue where certain container runtimes were reporting as bare-metal hosts.

### 🚀 Features

This release introduces the following new features:

- feat: improve docker docs by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#587](https://github.com/nginx/agent/pull/587)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix install-tools by [@Dean-Coakley](https://github.com/Dean-Coakley) in [agent#581](https://github.com/nginx/agent/pull/581)

### 📝 Documentation

We have made the following updates to the documentation:

- change log updated for last release by [@oliveromahony](https://github.com/oliveromahony) in [agent#583](https://github.com/nginx/agent/pull/583)
- Restore agent container information from nms docs  by [@jputrino](https://github.com/jputrino) in [agent#584](https://github.com/nginx/agent/pull/584)
- fix: add additional container checks during instance registration by [@mattdesmarais](https://github.com/mattdesmarais) in [agent#592](https://github.com/nginx/agent/pull/592)

---
## Release [v2.32.1](https//github.com/nginx/agent/releases/tag/v2.32.1)

### 🚀 Features

This release introduces the following new features:

- feat: Agent Docs IA refactor by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#548](https://github.com/nginx/agent/pull/548)
- feat: move NMS agent docs by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#553](https://github.com/nginx/agent/pull/553)
- feat: import changelog from github by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#570](https://github.com/nginx/agent/pull/570)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix runners and bump go version by [@oliveromahony](https://github.com/oliveromahony) in [agent#550](https://github.com/nginx/agent/pull/550)
- Fix artifact name by [@oliveromahony](https://github.com/oliveromahony) in [agent#558](https://github.com/nginx/agent/pull/558)
- fix: add missing catalog entry by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#572](https://github.com/nginx/agent/pull/572)

### 📝 Documentation

We have made the following updates to the documentation:

- Runc bump by [@oliveromahony](https://github.com/oliveromahony) in [agent#565](https://github.com/nginx/agent/pull/565)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- bump vulnerable version of buildkit by [@oliveromahony](https://github.com/oliveromahony) in [agent#564](https://github.com/nginx/agent/pull/564)

