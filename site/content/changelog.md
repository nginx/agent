---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [v2.35.1](https//github.com/nginx/agent/releases/tag/v2.35.1)

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
## Release [v2.35.0](https//github.com/nginx/agent/releases/tag/v2.35.0)

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
## Release [v2.34.1](https//github.com/nginx/agent/releases/tag/v2.34.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix metrics reporter retry logic by [@dhurley](https://github.com/dhurley) in [#700](https://github.com/nginx/agent/pull/700)

### 📝 Documentation

We have made the following updates to the documentation:

- Update changelog for release 2.34 by [@ADubhlaoich](https://github.com/ADubhlaoich) in [#693](https://github.com/nginx/agent/pull/693)

---
## Release [v2.34.0](https//github.com/nginx/agent/releases/tag/v2.34.0)

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
## Release [v2.33.0](https//github.com/nginx/agent/releases/tag/v2.33.0)

### 🌟 Highlights

- feat: Add Support for NAP 5 by [@edarzins](https://github.com/edarzins) in [#604](https://github.com/nginx/agent/pull/604)

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

---
## Release [v2.32.2](https//github.com/nginx/agent/releases/tag/v2.32.2)

### 🌟 Highlights

- This release fixes an issue where certain container runtimes were reporting as bare-metal hosts.

### 🚀 Features

This release introduces the following new features:

- feat: improve docker docs by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#587](https://github.com/nginx/agent/pull/587)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix install-tools by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#581](https://github.com/nginx/agent/pull/581)

### 📝 Documentation

We have made the following updates to the documentation:

- change log updated for last release by [@oliveromahony](https://github.com/oliveromahony) in [#583](https://github.com/nginx/agent/pull/583)
- Restore agent container information from nms docs  by [@jputrino](https://github.com/jputrino) in [#584](https://github.com/nginx/agent/pull/584)
- fix: add additional container checks during instance registration by [@mattdesmarais](https://github.com/mattdesmarais) in [#592](https://github.com/nginx/agent/pull/592)

---
## Release [v2.32.1](https//github.com/nginx/agent/releases/tag/v2.32.1)

### 🚀 Features

This release introduces the following new features:

- feat: Agent Docs IA refactor by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#548](https://github.com/nginx/agent/pull/548)
- feat: move NMS agent docs by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#553](https://github.com/nginx/agent/pull/553)
- feat: import changelog from github by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#570](https://github.com/nginx/agent/pull/570)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix runners and bump go version by [@oliveromahony](https://github.com/oliveromahony) in [#550](https://github.com/nginx/agent/pull/550)
- Fix artifact name by [@oliveromahony](https://github.com/oliveromahony) in [#558](https://github.com/nginx/agent/pull/558)
- fix: add missing catalog entry by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#572](https://github.com/nginx/agent/pull/572)

### 📝 Documentation

We have made the following updates to the documentation:

- Runc bump by [@oliveromahony](https://github.com/oliveromahony) in [#565](https://github.com/nginx/agent/pull/565)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- bump vulnerable version of buildkit by [@oliveromahony](https://github.com/oliveromahony) in [#564](https://github.com/nginx/agent/pull/564)

---
## Release [v2.32.0](https//github.com/nginx/agent/releases/tag/v2.32.0)

### 🚀 Features

This release introduces the following new features:

- feat: added the new OS support for NGINX R31 by [@oliveromahony](https://github.com/oliveromahony) in [#538](https://github.com/nginx/agent/pull/538)

---
## Release [v2.31.2](https//github.com/nginx/agent/releases/tag/v2.31.2)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- chore: rename hugo folder to site, fix product naming by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#527](https://github.com/nginx/agent/pull/527)

### 📝 Documentation

We have made the following updates to the documentation:

- Update upgrade documentation by [@dhurley](https://github.com/dhurley) in [#526](https://github.com/nginx/agent/pull/526)
- Bump the versions of containerd and go-git dependencies by [@dhurley](https://github.com/dhurley) in [#533](https://github.com/nginx/agent/pull/533)
- updated dependencies by [@oliveromahony](https://github.com/oliveromahony) in [#536](https://github.com/nginx/agent/pull/536)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Bump crypto dependency from 0.14.0 to 0.17.0 by [@dhurley](https://github.com/dhurley) in [#532](https://github.com/nginx/agent/pull/532)

---
## Release [v2.31.1](https//github.com/nginx/agent/releases/tag/v2.31.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix permissions for log file and dynamic config directory by [@aphralG](https://github.com/aphralG) in [#517](https://github.com/nginx/agent/pull/517)
- Fix server example in sdk to have timeout by [@aphralG](https://github.com/aphralG) in [#518](https://github.com/nginx/agent/pull/518)

### 📝 Documentation

We have made the following updates to the documentation:

- Update SELinux Readme by [@aphralG](https://github.com/aphralG) in [#522](https://github.com/nginx/agent/pull/522)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Replace mockgen by [@oliveromahony](https://github.com/oliveromahony) in [#524](https://github.com/nginx/agent/pull/524)
- Restrict config apply directory permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#519](https://github.com/nginx/agent/pull/519)
- Restrict NAP file/dir permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#516](https://github.com/nginx/agent/pull/516)

