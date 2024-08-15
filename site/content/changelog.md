---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [vundefined](https//github.com/nginx/agent/releases/tag/vundefined)

### 📝 Documentation

We have made the following updates to the documentation:

- docs: update GPG keys by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#776](https://github.com/nginx/agent/pull/776)

---
## Release [vchangelog](https//github.com/nginx/agent/releases/tag/vchangelog)

### 📝 Documentation

We have made the following updates to the documentation:

- docs: update GPG keys by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#776](https://github.com/nginx/agent/pull/776)

---
## Release [v2.37.0](https//github.com/nginx/agent/releases/tag/v2.37.0)

### 🚀 Features

This release introduces the following new features:

- feat: Update the changelog by [@ADubhlaoich](https://github.com/ADubhlaoich) in [agent#753](https://github.com/nginx/agent/pull/753)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Prevent writing outside allowed directories list from a config payload with actions by [@oliveromahony](https://github.com/oliveromahony) in [agent#766](https://github.com/nginx/agent/pull/766)
- The letter v is now always prepended to output of -v by [@olli](https://github.com/olli)-holmala in [agent#751](https://github.com/nginx/agent/pull/751)
- Fix backoff to drop Metrics Reports from buffer after max_elapsed_time has been reached by [@oliveromahony](https://github.com/oliveromahony) in [agent#752](https://github.com/nginx/agent/pull/752)
- Fix Post Install Script Issues by [@spencerugbo](https://github.com/spencerugbo) in [agent#739](https://github.com/nginx/agent/pull/739)
- docs: fix github links in changelog by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#770](https://github.com/nginx/agent/pull/770)
- Fix post install script for when no nginx instance is installed by [@dhurley](https://github.com/dhurley) in [agent#773](https://github.com/nginx/agent/pull/773)

### 📝 Documentation

We have made the following updates to the documentation:

- Upgrade prometheus exporter version to latest by [@oliveromahony](https://github.com/oliveromahony) in [agent#749](https://github.com/nginx/agent/pull/749)
- Add badges for Go version, release, license, contributions, and Slack… by [@oCHRISo](https://github.com/oCHRISo) in [agent#763](https://github.com/nginx/agent/pull/763)
- Add instructions for Amazon Linux 2023 by [@nginx](https://github.com/nginx)-seanmoloney in [agent#759](https://github.com/nginx/agent/pull/759)
- Add docs-build-push github workflow by [@nginx](https://github.com/nginx)-jack in [agent#765](https://github.com/nginx/agent/pull/765)

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

- Fix incorrect bold tag in heading by [@nginx](https://github.com/nginx)-seanmoloney in [agent#715](https://github.com/nginx/agent/pull/715)
- URL fix for building docker image in README.md by [@y82](https://github.com/y82) in [agent#720](https://github.com/nginx/agent/pull/720)
- Fix for version by [@oliveromahony](https://github.com/oliveromahony) in [agent#732](https://github.com/nginx/agent/pull/732)

### 📝 Documentation

We have made the following updates to the documentation:

- More flexible container images for the official images by [@oliveromahony](https://github.com/oliveromahony) in [agent#729](https://github.com/nginx/agent/pull/729)
- Update configuration examples by [@nginx](https://github.com/nginx)-seanmoloney in [agent#731](https://github.com/nginx/agent/pull/731)
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
- Add Ubuntu Noble 24.04 LTS support by [@Dean](https://github.com/Dean)-Coakley in [agent#682](https://github.com/nginx/agent/pull/682)

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

- Fix install-tools by [@Dean](https://github.com/Dean)-Coakley in [agent#581](https://github.com/nginx/agent/pull/581)

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

---
## Release [v2.32.0](https//github.com/nginx/agent/releases/tag/v2.32.0)

### 🚀 Features

This release introduces the following new features:

- feat: added the new OS support for NGINX R31 by [@oliveromahony](https://github.com/oliveromahony) in [agent#538](https://github.com/nginx/agent/pull/538)

---
## Release [v2.31.2](https//github.com/nginx/agent/releases/tag/v2.31.2)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- chore: rename hugo folder to site, fix product naming by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#527](https://github.com/nginx/agent/pull/527)

### 📝 Documentation

We have made the following updates to the documentation:

- Update upgrade documentation by [@dhurley](https://github.com/dhurley) in [agent#526](https://github.com/nginx/agent/pull/526)
- Bump the versions of containerd and go-git dependencies by [@dhurley](https://github.com/dhurley) in [agent#533](https://github.com/nginx/agent/pull/533)
- updated dependencies by [@oliveromahony](https://github.com/oliveromahony) in [agent#536](https://github.com/nginx/agent/pull/536)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Bump crypto dependency from 0.14.0 to 0.17.0 by [@dhurley](https://github.com/dhurley) in [agent#532](https://github.com/nginx/agent/pull/532)

---
## Release [v2.31.1](https//github.com/nginx/agent/releases/tag/v2.31.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix permissions for log file and dynamic config directory by [@aphralG](https://github.com/aphralG) in [agent#517](https://github.com/nginx/agent/pull/517)
- Fix server example in sdk to have timeout by [@aphralG](https://github.com/aphralG) in [agent#518](https://github.com/nginx/agent/pull/518)

### 📝 Documentation

We have made the following updates to the documentation:

- Update SELinux Readme by [@aphralG](https://github.com/aphralG) in [agent#522](https://github.com/nginx/agent/pull/522)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Replace mockgen by [@oliveromahony](https://github.com/oliveromahony) in [agent#524](https://github.com/nginx/agent/pull/524)
- Restrict config apply directory permissions by [@Dean](https://github.com/Dean)-Coakley in [agent#519](https://github.com/nginx/agent/pull/519)
- Restrict NAP file/dir permissions by [@Dean](https://github.com/Dean)-Coakley in [agent#516](https://github.com/nginx/agent/pull/516)

---
## Release [v2.31.0](https//github.com/nginx/agent/releases/tag/v2.31.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix otelcontrib version by [@oliveromahony](https://github.com/oliveromahony) in [agent#504](https://github.com/nginx/agent/pull/504)
- Fix user agent request header to have the correct agent version by [@dhurley](https://github.com/dhurley) in [agent#498](https://github.com/nginx/agent/pull/498)
- Fix alpine plus dockerfile on alpine>=3.17 by [@Dean](https://github.com/Dean)-Coakley in [agent#511](https://github.com/nginx/agent/pull/511)
- fix: avoid stopping nginx-agent service on package upgrade by [@defanator](https://github.com/defanator) in [agent#352](https://github.com/nginx/agent/pull/352)
- Fix SELinux Policy by [@aphralG](https://github.com/aphralG) in [agent#520](https://github.com/nginx/agent/pull/520)

### 📝 Documentation

We have made the following updates to the documentation:

- Add CLI arg to set dynamic config path by [@Dean](https://github.com/Dean)-Coakley in [agent#490](https://github.com/nginx/agent/pull/490)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- crossplane version bump by [@oliveromahony](https://github.com/oliveromahony) in [agent#512](https://github.com/nginx/agent/pull/512)
- Add commander retry lock by [@dhurley](https://github.com/dhurley) in [agent#502](https://github.com/nginx/agent/pull/502)
- Bump otel dependency version and fix github workflow for dependabot PRs by [@dhurley](https://github.com/dhurley) in [agent#515](https://github.com/nginx/agent/pull/515)

---
## Release [v2.30.3](https//github.com/nginx/agent/releases/tag/v2.30.3)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix dependabot issues by [@oliveromahony](https://github.com/oliveromahony) in [agent#503](https://github.com/nginx/agent/pull/503)

---
## Release [v2.30.2](https//github.com/nginx/agent/releases/tag/v2.30.2)

---
## Release [v2.30.1](https//github.com/nginx/agent/releases/tag/v2.30.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: Tolerate additional fields in App Protect yaml files by [@edarzins](https://github.com/edarzins) in [agent#494](https://github.com/nginx/agent/pull/494)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Update nginx-plus-go-client to stop 404 errors in NGINX access logs by [@dhurley](https://github.com/dhurley) in [agent#495](https://github.com/nginx/agent/pull/495)

---
## Release [v2.30.0](https//github.com/nginx/agent/releases/tag/v2.30.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix version for forked repo by [@dhurley](https://github.com/dhurley) in [agent#468](https://github.com/nginx/agent/pull/468)
- Fix integration tests by [@aphralG](https://github.com/aphralG) in [agent#478](https://github.com/nginx/agent/pull/478)
- Fix config apply by [@oliveromahony](https://github.com/oliveromahony) in [agent#480](https://github.com/nginx/agent/pull/480)
- deprecate system.mem.used.all metric by [@aphralG](https://github.com/aphralG) in [agent#485](https://github.com/nginx/agent/pull/485)

### 📝 Documentation

We have made the following updates to the documentation:

- Update CLI flags documentation by [@Dean](https://github.com/Dean)-Coakley in [agent#476](https://github.com/nginx/agent/pull/476)
- Update NGINX plugin to read NGINX config on startup by [@dhurley](https://github.com/dhurley) in [agent#489](https://github.com/nginx/agent/pull/489)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Update file watcher to ignore .swx files by [@dhurley](https://github.com/dhurley) in [agent#466](https://github.com/nginx/agent/pull/466)
- Check Simplemetrics is not empty  by [@aphralG](https://github.com/aphralG) in [agent#474](https://github.com/nginx/agent/pull/474)
- Add error log if duplicate NGINX IDs are found by [@dhurley](https://github.com/dhurley) in [agent#477](https://github.com/nginx/agent/pull/477)
- Add tests for additional SSL directives and key algorithms. #276 by [@arsenalzp](https://github.com/arsenalzp) in [agent#469](https://github.com/nginx/agent/pull/469)
- call underlying os.Hostname instead of the entire hostInfo gopsutil call by [@oliveromahony](https://github.com/oliveromahony) in [agent#479](https://github.com/nginx/agent/pull/479)
- Add grpc integration tests by [@dhurley](https://github.com/dhurley) in [agent#475](https://github.com/nginx/agent/pull/475)
- remove error log causing failures  by [@aphralG](https://github.com/aphralG) in [agent#488](https://github.com/nginx/agent/pull/488)
- Use singleflight for caching environment.go calls by [@oliveromahony](https://github.com/oliveromahony) in [agent#481](https://github.com/nginx/agent/pull/481)
- Reduce the number of times env.Processes gets called by [@dhurley](https://github.com/dhurley) in [agent#482](https://github.com/nginx/agent/pull/482)
- add additional check to nginxProcesses by [@aphralG](https://github.com/aphralG) in [agent#483](https://github.com/nginx/agent/pull/483)
- profile.cgo by [@oliveromahony](https://github.com/oliveromahony) in [agent#493](https://github.com/nginx/agent/pull/493)

---
## Release [v2.29.0](https//github.com/nginx/agent/releases/tag/v2.29.0)

### 🚀 Features

This release introduces the following new features:

- Add metric sender feature by [@Dean](https://github.com/Dean)-Coakley in [agent#453](https://github.com/nginx/agent/pull/453)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: fix logic for parsing absolute path to nginx config file by [@sylwang](https://github.com/sylwang) in [agent#445](https://github.com/nginx/agent/pull/445)
- Fix SELinux Policy & Fix SELinux README by [@aphralG](https://github.com/aphralG) in [agent#467](https://github.com/nginx/agent/pull/467)
- fix: ensure fullpath by [@nginx](https://github.com/nginx)-nickc in [agent#471](https://github.com/nginx/agent/pull/471)

### 📝 Documentation

We have made the following updates to the documentation:

- Remove Ubuntu 18.04, Alpine 3.13 and Alpine 3.14 OS support by [@dhurley](https://github.com/dhurley) in [agent#444](https://github.com/nginx/agent/pull/444)
- Go 1.21 by [@oliveromahony](https://github.com/oliveromahony) in [agent#459](https://github.com/nginx/agent/pull/459)
- Add Alpine 3.18 support by [@dhurley](https://github.com/dhurley) in [agent#443](https://github.com/nginx/agent/pull/443)
- Update selinux readme by [@dhurley](https://github.com/dhurley) in [agent#449](https://github.com/nginx/agent/pull/449)
- Add proto-buf definitions for php-fpm metrics by [@achawla2012](https://github.com/achawla2012) in [agent#452](https://github.com/nginx/agent/pull/452)
- Performance tests for loading of plugins and different feature combinations by [@oliveromahony](https://github.com/oliveromahony) in [agent#463](https://github.com/nginx/agent/pull/463)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Add Debian 12 support by [@dhurley](https://github.com/dhurley) in [agent#442](https://github.com/nginx/agent/pull/442)
- Revert "Merge release-2.28.1 back into main (#455)" by [@oliveromahony](https://github.com/oliveromahony) in [agent#457](https://github.com/nginx/agent/pull/457)
- Release 2.28.1 by [@oliveromahony](https://github.com/oliveromahony) in [agent#458](https://github.com/nginx/agent/pull/458)
- More benchmark tests by [@oliveromahony](https://github.com/oliveromahony) in [agent#462](https://github.com/nginx/agent/pull/462)
- Only create metrics sender if a metric reporter is already created by [@dhurley](https://github.com/dhurley) in [agent#465](https://github.com/nginx/agent/pull/465)
- Register php-fpm metrics as extension plugin by [@achawla2012](https://github.com/achawla2012) in [agent#451](https://github.com/nginx/agent/pull/451)
- Add support for file actions during a config apply by [@dhurley](https://github.com/dhurley) in [agent#464](https://github.com/nginx/agent/pull/464)
- Add TLS upstream and TLS server_zone metrics by [@Dean](https://github.com/Dean)-Coakley in [agent#470](https://github.com/nginx/agent/pull/470)
- Added cgo profile to build NGINX Agent packages by [@oliveromahony](https://github.com/oliveromahony) in [agent#472](https://github.com/nginx/agent/pull/472)
- Add worker conn metrics by [@Dean](https://github.com/Dean)-Coakley in [agent#461](https://github.com/nginx/agent/pull/461)

---
## Release [v2.28.1](https//github.com/nginx/agent/releases/tag/v2.28.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix for dash during upgrade by [@oliveromahony](https://github.com/oliveromahony) in [agent#450](https://github.com/nginx/agent/pull/450)
- Improve status API detection and validation by [@dhurley](https://github.com/dhurley) in [agent#447](https://github.com/nginx/agent/pull/447)
- Rebuild selinux policy on RHEL 8 by [@dhurley](https://github.com/dhurley) in [agent#448](https://github.com/nginx/agent/pull/448)

---
## Release [v2.28.0](https//github.com/nginx/agent/releases/tag/v2.28.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix selinux policy on RHEL 8 by [@dhurley](https://github.com/dhurley) in [agent#409](https://github.com/nginx/agent/pull/409)
- fix pipeline by [@aphralG](https://github.com/aphralG) in [agent#411](https://github.com/nginx/agent/pull/411)
- fix: use host architecture to determine FreeBSD ABI by [@defanator](https://github.com/defanator) in [agent#350](https://github.com/nginx/agent/pull/350)
- Reload NGINX after rolling back NGINX configuration changes by [@dhurley](https://github.com/dhurley) in [agent#419](https://github.com/nginx/agent/pull/419)
- Update how the SDK parses NGINX API server directive by [@dhurley](https://github.com/dhurley) in [agent#423](https://github.com/nginx/agent/pull/423)
- Update nginx access & error log metric sources to only report metrics that are available by [@dhurley](https://github.com/dhurley) in [agent#424](https://github.com/nginx/agent/pull/424)
- fix: bump hugo theme to 0.35.0 by [@Jcahilltorre](https://github.com/Jcahilltorre) in [agent#436](https://github.com/nginx/agent/pull/436)
- Fix how status API URL is determined when both stub status and NGINX Plus APIs are configured by [@dhurley](https://github.com/dhurley) in [agent#433](https://github.com/nginx/agent/pull/433)
- Fix intermittent issue where the reloadNginx function never finishes by [@dhurley](https://github.com/dhurley) in [agent#431](https://github.com/nginx/agent/pull/431)

### 📝 Documentation

We have made the following updates to the documentation:

- Bump github.com/goreleaser/nfpm/v2 from 2.31.0 to 2.32.0 by [@dependabot](https://github.com/dependabot) in [agent#407](https://github.com/nginx/agent/pull/407)
- Unit test failures by [@oliveromahony](https://github.com/oliveromahony) in [agent#412](https://github.com/nginx/agent/pull/412)
- Bump github.com/bufbuild/buf from 1.23.1 to 1.24.0 by [@dependabot](https://github.com/dependabot) in [agent#406](https://github.com/nginx/agent/pull/406)
- Bump github.com/evilmartians/lefthook from 1.4.4 to 1.4.5 by [@dependabot](https://github.com/dependabot) in [agent#404](https://github.com/nginx/agent/pull/404)
- Upgrade nginx-hugo-theme to support newer versions of Hugo by [@jputrino](https://github.com/jputrino) in [agent#420](https://github.com/nginx/agent/pull/420)
- enable log rotation by [@aphralG](https://github.com/aphralG) in [agent#414](https://github.com/nginx/agent/pull/414)
- Add documentation for docker images by [@dhurley](https://github.com/dhurley) in [agent#418](https://github.com/nginx/agent/pull/418)
- Remove memory leaks by [@oliveromahony](https://github.com/oliveromahony) in [agent#422](https://github.com/nginx/agent/pull/422)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Monitor NGINX logs for critical & error messages after NGINX reload  by [@aphralG](https://github.com/aphralG) in [agent#385](https://github.com/nginx/agent/pull/385)
- fix integration test failures  by [@aphralG](https://github.com/aphralG) in [agent#396](https://github.com/nginx/agent/pull/396)
- Merge release-2.27.1 back into main by [@dhurley](https://github.com/dhurley) in [agent#416](https://github.com/nginx/agent/pull/416)
- Bump github.com/go-swagger/go-swagger from 0.30.4 to 0.30.5 by [@dependabot](https://github.com/dependabot) in [agent#403](https://github.com/nginx/agent/pull/403)
- blank log path should not log to file by [@aphralG](https://github.com/aphralG) in [agent#427](https://github.com/nginx/agent/pull/427)
- More tidying of code and memory leaks by [@oliveromahony](https://github.com/oliveromahony) in [agent#432](https://github.com/nginx/agent/pull/432)
- Improve error logging for gRPC EOF errors by [@dhurley](https://github.com/dhurley) in [agent#429](https://github.com/nginx/agent/pull/429)
- chore: cert directives for ssl_client_certificate and ssl_trusted_certificate by [@CodeMonkeyF5](https://github.com/CodeMonkeyF5) in [agent#430](https://github.com/nginx/agent/pull/430)
- feat: Add ADM resource name dimensions by [@p](https://github.com/p)-borole in [agent#435](https://github.com/nginx/agent/pull/435)
- Send error to UI when config async is disabled  by [@aphralG](https://github.com/aphralG) in [agent#426](https://github.com/nginx/agent/pull/426)
- fix: update crossplane version to use the latest parsing logic for NGINX config files by [@sylwang](https://github.com/sylwang) in [agent#438](https://github.com/nginx/agent/pull/438)
- make deps and remove duplicate if statement by [@oliveromahony](https://github.com/oliveromahony) in [agent#440](https://github.com/nginx/agent/pull/440)
- Race condition reloading by [@oliveromahony](https://github.com/oliveromahony) in [agent#437](https://github.com/nginx/agent/pull/437)

---
## Release [v2.27.1](https//github.com/nginx/agent/releases/tag/v2.27.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix selinux policy on RHEL 8 by [@dhurley](https://github.com/dhurley) in [agent#413](https://github.com/nginx/agent/pull/413)
- Fix grpc reconnect logic by [@dhurley](https://github.com/dhurley) in [agent#401](https://github.com/nginx/agent/pull/401)

---
## Release [v2.26.2](https//github.com/nginx/agent/releases/tag/v2.26.2)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix selinux policy on RHEL 8 by [@dhurley](https://github.com/dhurley) in [agent#409](https://github.com/nginx/agent/pull/409)

---
## Release [v2.27.0](https//github.com/nginx/agent/releases/tag/v2.27.0)

### 🚀 Features

This release introduces the following new features:

- Ignore directives feature by [@u5surf](https://github.com/u5surf) in [agent#343](https://github.com/nginx/agent/pull/343)
- Remove duplicate code from enable/disable features & add tests by [@aphralG](https://github.com/aphralG) in [agent#361](https://github.com/nginx/agent/pull/361)
- Add Dockerfiles & docker-compose files for official OSS & Plus images by [@dhurley](https://github.com/dhurley) in [agent#353](https://github.com/nginx/agent/pull/353)
- Improve enable/disable features by [@aphralG](https://github.com/aphralG) in [agent#393](https://github.com/nginx/agent/pull/393)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: fixes segfault when process is TERMed by [@dekobon](https://github.com/dekobon) in [agent#335](https://github.com/nginx/agent/pull/335)
- fix: fix Linux default network interface parsing by [@dekobon](https://github.com/dekobon) in [agent#331](https://github.com/nginx/agent/pull/331)
- Fix warnings when purging agent by [@aphralG](https://github.com/aphralG) in [agent#345](https://github.com/nginx/agent/pull/345)
- Fix Packaging  by [@aphralG](https://github.com/aphralG) in [agent#341](https://github.com/nginx/agent/pull/341)
- added fix for listen directives in stub status and plus API configs by [@oliveromahony](https://github.com/oliveromahony) in [agent#348](https://github.com/nginx/agent/pull/348)
- Fix: specified predefined access log format (#340) by [@u5surf](https://github.com/u5surf) in [agent#349](https://github.com/nginx/agent/pull/349)
- Fix access log parsing for custom log format by [@nkashiv](https://github.com/nkashiv) in [agent#346](https://github.com/nginx/agent/pull/346)
- fix performance tests by [@aphralG](https://github.com/aphralG) in [agent#358](https://github.com/nginx/agent/pull/358)
- fixed tooling to pull from vendor and go.mod by [@oliveromahony](https://github.com/oliveromahony) in [agent#359](https://github.com/nginx/agent/pull/359)
- Add nil check for controller when agent is shutting down by [@dhurley](https://github.com/dhurley) in [agent#357](https://github.com/nginx/agent/pull/357)
- Fix integer overflow by [@dhurley](https://github.com/dhurley) in [agent#364](https://github.com/nginx/agent/pull/364)
- Fix worker io metrics permission issue when running in a container by [@dhurley](https://github.com/dhurley) in [agent#383](https://github.com/nginx/agent/pull/383)
- fix amazon packaging by [@aphralG](https://github.com/aphralG) in [agent#399](https://github.com/nginx/agent/pull/399)
- Fix docker issues by [@oliveromahony](https://github.com/oliveromahony) in [agent#398](https://github.com/nginx/agent/pull/398)
- Fix backoff settings by [@dhurley](https://github.com/dhurley) in [agent#382](https://github.com/nginx/agent/pull/382)

### 📝 Documentation

We have made the following updates to the documentation:

- doc: add comments for each field in the SecurityViolationEvent proto definition by [@mohamed](https://github.com/mohamed)-gougam in [agent#325](https://github.com/nginx/agent/pull/325)
- Update Go version by [@oCHRISo](https://github.com/oCHRISo) in [agent#326](https://github.com/nginx/agent/pull/326)
- Close controller before exiting by [@nkashiv](https://github.com/nkashiv) in [agent#320](https://github.com/nginx/agent/pull/320)
- Add configuration overview by [@oCHRISo](https://github.com/oCHRISo) in [agent#324](https://github.com/nginx/agent/pull/324)
- Add NGINX Agent Uninstall Doc by [@oCHRISo](https://github.com/oCHRISo) in [agent#323](https://github.com/nginx/agent/pull/323)
- Add gofumpt by [@oliveromahony](https://github.com/oliveromahony) in [agent#380](https://github.com/nginx/agent/pull/380)
- Nfpm version bump by [@oliveromahony](https://github.com/oliveromahony) in [agent#386](https://github.com/nginx/agent/pull/386)
- Bump runc by [@oliveromahony](https://github.com/oliveromahony) in [agent#387](https://github.com/nginx/agent/pull/387)
- Updates from dependabot by [@oliveromahony](https://github.com/oliveromahony) in [agent#397](https://github.com/nginx/agent/pull/397)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- chore: update deprecated API calls by [@dekobon](https://github.com/dekobon) in [agent#332](https://github.com/nginx/agent/pull/332)
- Output mount point in error message by [@dekobon](https://github.com/dekobon) in [agent#329](https://github.com/nginx/agent/pull/329)
- error_log, access_log tackle syslog (#185) by [@Retssaze](https://github.com/Retssaze) in [agent#302](https://github.com/nginx/agent/pull/302)
- Skip warnings on non-file log destinations by [@dekobon](https://github.com/dekobon) in [agent#330](https://github.com/nginx/agent/pull/330)
- Monitor NGINX logs for alert messages after NGINX reload by [@aphralG](https://github.com/aphralG) in [agent#351](https://github.com/nginx/agent/pull/351)
- Support docker install from package repository by [@Dean](https://github.com/Dean)-Coakley in [agent#354](https://github.com/nginx/agent/pull/354)
- Add timeout to integration tests setup by [@Dean](https://github.com/Dean)-Coakley in [agent#360](https://github.com/nginx/agent/pull/360)
- Add Amazon Linux 2023 support by [@dhurley](https://github.com/dhurley) in [agent#355](https://github.com/nginx/agent/pull/355)
- Average memory metrics instead of summing by [@Giners](https://github.com/Giners) in [agent#362](https://github.com/nginx/agent/pull/362)
- Auto dependency checks by [@oliveromahony](https://github.com/oliveromahony) in [agent#372](https://github.com/nginx/agent/pull/372)
- Add capture integration test docker logs by [@Dean](https://github.com/Dean)-Coakley in [agent#365](https://github.com/nginx/agent/pull/365)
- Use ParseInt instead of Atoi when converting process id string to an int by [@dhurley](https://github.com/dhurley) in [agent#384](https://github.com/nginx/agent/pull/384)
- Support ltsv by [@u5surf](https://github.com/u5surf) in [agent#363](https://github.com/nginx/agent/pull/363)
- Populate loopback interface metrics to a metric report by [@achawla2012](https://github.com/achawla2012) in [agent#381](https://github.com/nginx/agent/pull/381)

---
## Release [v2.26.1](https//github.com/nginx/agent/releases/tag/v2.26.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix segmentation fault during agent registration by [@achawla2012](https://github.com/achawla2012) in [agent#338](https://github.com/nginx/agent/pull/338)
- Update selinux policy by [@dhurley](https://github.com/dhurley) in [agent#344](https://github.com/nginx/agent/pull/344)

---
## Release [v2.26.0](https//github.com/nginx/agent/releases/tag/v2.26.0)

### 🚀 Features

This release introduces the following new features:

- Enable disable features by [@aphralG](https://github.com/aphralG) in [agent#300](https://github.com/nginx/agent/pull/300)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix replace usage of flaky epel repo by [@Dean](https://github.com/Dean)-Coakley in [agent#289](https://github.com/nginx/agent/pull/289)
- fixed vendoring by [@oliveromahony](https://github.com/oliveromahony) in [agent#295](https://github.com/nginx/agent/pull/295)
- Fix Vendoring  by [@aphralG](https://github.com/aphralG) in [agent#298](https://github.com/nginx/agent/pull/298)
- Fix empty access item when log format is changed  by [@aphralG](https://github.com/aphralG) in [agent#270](https://github.com/nginx/agent/pull/270)
- Fix handing of setting access_log off; by [@Dean](https://github.com/Dean)-Coakley in [agent#278](https://github.com/nginx/agent/pull/278)
- Fix processing error for pause message on agent config topic by [@achawla2012](https://github.com/achawla2012) in [agent#314](https://github.com/nginx/agent/pull/314)
- Fix packager script post deinstall. by [@u5surf](https://github.com/u5surf) in [agent#305](https://github.com/nginx/agent/pull/305)
- fixed bullseye docker image by [@oliveromahony](https://github.com/oliveromahony) in [agent#327](https://github.com/nginx/agent/pull/327)

### 📝 Documentation

We have made the following updates to the documentation:

- Merge docs development changes forward to main  by [@jputrino](https://github.com/jputrino) in [agent#287](https://github.com/nginx/agent/pull/287)
- Update go-crossplane version to 0.4.15 by [@dareste](https://github.com/dareste) in [agent#304](https://github.com/nginx/agent/pull/304)
- move agent-dynamic.conf to /var/lib/nginx-agent by [@aphralG](https://github.com/aphralG) in [agent#268](https://github.com/nginx/agent/pull/268)
- Bumped version of crossplane by [@oliveromahony](https://github.com/oliveromahony) in [agent#315](https://github.com/nginx/agent/pull/315)
- Packages scripts refactor by [@oCHRISo](https://github.com/oCHRISo) in [agent#316](https://github.com/nginx/agent/pull/316)
- Update OS Support Docs by [@oCHRISo](https://github.com/oCHRISo) in [agent#296](https://github.com/nginx/agent/pull/296)
- Add docs for installing nginx-agent from repository by [@Dean](https://github.com/Dean)-Coakley in [agent#309](https://github.com/nginx/agent/pull/309)
- refactor: populate key-value depending on their content by [@mohamed](https://github.com/mohamed)-gougam in [agent#308](https://github.com/nginx/agent/pull/308)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- chore: Address issues identified by linter in src/core/environment.go by [@mrajagopal](https://github.com/mrajagopal) in [agent#291](https://github.com/nginx/agent/pull/291)
- add coverage to XML parsing in NAP Monitoring by [@mohamed](https://github.com/mohamed)-gougam in [agent#288](https://github.com/nginx/agent/pull/288)
- Support GetNetOverflow in linux by [@u5surf](https://github.com/u5surf) in [agent#301](https://github.com/nginx/agent/pull/301)
- Update Agent to support backpressure from server by [@achawla2012](https://github.com/achawla2012) in [agent#299](https://github.com/nginx/agent/pull/299)
- Change default REST api config by [@oCHRISo](https://github.com/oCHRISo) in [agent#321](https://github.com/nginx/agent/pull/321)

---
## Release [v2.25.1](https//github.com/nginx/agent/releases/tag/v2.25.1)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- remove pids draining by [@oliveromahony](https://github.com/oliveromahony) in [agent#292](https://github.com/nginx/agent/pull/292)

---
## Release [v2.25.0](https//github.com/nginx/agent/releases/tag/v2.25.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix potential race by [@nickchen](https://github.com/nickchen) in [agent#279](https://github.com/nginx/agent/pull/279)
- fix: Pass correct variable for lscpu command to work by [@mrajagopal](https://github.com/mrajagopal) in [agent#280](https://github.com/nginx/agent/pull/280)

### 📝 Documentation

We have made the following updates to the documentation:

- Add Supported distributions by [@oCHRISo](https://github.com/oCHRISo) in [agent#269](https://github.com/nginx/agent/pull/269)
- refactor: extend security violation context parsing; by [@mohamed](https://github.com/mohamed)-gougam in [agent#265](https://github.com/nginx/agent/pull/265)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Add check if there are any warnings in the NGINX config validation by [@dhurley](https://github.com/dhurley) in [agent#257](https://github.com/nginx/agent/pull/257)
- Create dedicated cache and upstream metrics reports by [@karlsassenberg](https://github.com/karlsassenberg) in [agent#189](https://github.com/nginx/agent/pull/189)
- Add wrapper to execute shell commands by [@achawla2012](https://github.com/achawla2012) in [agent#261](https://github.com/nginx/agent/pull/261)
- added waitgroup done in tests by [@oliveromahony](https://github.com/oliveromahony) in [agent#271](https://github.com/nginx/agent/pull/271)
- Add oracle linux 9 support by [@aphralG](https://github.com/aphralG) in [agent#264](https://github.com/nginx/agent/pull/264)
- Monitor NGINX logs for errors & NGINX worker processes after a NGINX reload by [@dhurley](https://github.com/dhurley) in [agent#255](https://github.com/nginx/agent/pull/255)
- Reduce logging verbosity by [@Dean](https://github.com/Dean)-Coakley in [agent#275](https://github.com/nginx/agent/pull/275)
- Add alpine 3.17 support by [@Dean](https://github.com/Dean)-Coakley in [agent#273](https://github.com/nginx/agent/pull/273)
- Add building amd64 RPMs for RHEL by [@Dean](https://github.com/Dean)-Coakley in [agent#281](https://github.com/nginx/agent/pull/281)
- Update NGINX monitor function to handle monitoring multiple NGINX error logs by [@dhurley](https://github.com/dhurley) in [agent#282](https://github.com/nginx/agent/pull/282)

