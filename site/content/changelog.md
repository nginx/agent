---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [v2.32.0](https//github.com/nginx/agent/releases/tag/v2.32.0)

### 🚀 Features
- feat: added the new OS support for NGINX R31 by [@oliveromahony](https://github.com/oliveromahony) in [#538](https://github.com/nginx/agent/pull/538)

---
## Release [v2.31.2](https//github.com/nginx/agent/releases/tag/v2.31.2)

### 🐛 Bug Fixes

- chore: rename hugo folder to site, fix product naming by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#527](https://github.com/nginx/agent/pull/527)

### 📝 Documentation

- Update upgrade documentation by [@dhurley](https://github.com/dhurley) in [#526](https://github.com/nginx/agent/pull/526)

- Bump the versions of containerd and go-git dependencies by [@dhurley](https://github.com/dhurley) in [#533](https://github.com/nginx/agent/pull/533)

- updated dependencies by [@oliveromahony](https://github.com/oliveromahony) in [#536](https://github.com/nginx/agent/pull/536)

### 🔨 Maintenance

- Bump crypto dependency from 0.14.0 to 0.17.0 by [@dhurley](https://github.com/dhurley) in [#532](https://github.com/nginx/agent/pull/532)

---
## Release [v2.31.1](https//github.com/nginx/agent/releases/tag/v2.31.1)

### 🐛 Bug Fixes

- Fix permissions for log file and dynamic config directory by [@aphralG](https://github.com/aphralG) in [#517](https://github.com/nginx/agent/pull/517)

- Fix server example in sdk to have timeout by [@aphralG](https://github.com/aphralG) in [#518](https://github.com/nginx/agent/pull/518)

### 📝 Documentation

- Update SELinux Readme by [@aphralG](https://github.com/aphralG) in [#522](https://github.com/nginx/agent/pull/522)

### 🔨 Maintenance

- Replace mockgen by [@oliveromahony](https://github.com/oliveromahony) in [#524](https://github.com/nginx/agent/pull/524)

- Restrict config apply directory permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#519](https://github.com/nginx/agent/pull/519)

- Restrict NAP file/dir permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#516](https://github.com/nginx/agent/pull/516)

---
## Release [v2.31.0](https//github.com/nginx/agent/releases/tag/v2.31.0)

### 🐛 Bug Fixes

- Fix otelcontrib version by [@oliveromahony](https://github.com/oliveromahony) in [#504](https://github.com/nginx/agent/pull/504)

- Fix user agent request header to have the correct agent version by [@dhurley](https://github.com/dhurley) in [#498](https://github.com/nginx/agent/pull/498)

- Fix alpine plus dockerfile on alpine>=3.17 by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#511](https://github.com/nginx/agent/pull/511)

- fix: avoid stopping nginx-agent service on package upgrade by [@defanator](https://github.com/defanator) in [#352](https://github.com/nginx/agent/pull/352)

- Fix SELinux Policy by [@aphralG](https://github.com/aphralG) in [#520](https://github.com/nginx/agent/pull/520)

### 📝 Documentation

- Add CLI arg to set dynamic config path by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#490](https://github.com/nginx/agent/pull/490)

### 🔨 Maintenance

- crossplane version bump by [@oliveromahony](https://github.com/oliveromahony) in [#512](https://github.com/nginx/agent/pull/512)

- Add commander retry lock by [@dhurley](https://github.com/dhurley) in [#502](https://github.com/nginx/agent/pull/502)

- Bump otel dependency version and fix github workflow for dependabot PRs by [@dhurley](https://github.com/dhurley) in [#515](https://github.com/nginx/agent/pull/515)

---
## Release [v2.30.3](https//github.com/nginx/agent/releases/tag/v2.30.3)

### 🐛 Bug Fixes

- Fix dependabot issues by [@oliveromahony](https://github.com/oliveromahony) in [#503](https://github.com/nginx/agent/pull/503)

---
## Release [v2.30.1](https//github.com/nginx/agent/releases/tag/v2.30.1)

### 🐛 Bug Fixes

- fix: Tolerate additional fields in App Protect yaml files by [@edarzins](https://github.com/edarzins) in [#494](https://github.com/nginx/agent/pull/494)

### 🔨 Maintenance

- Update nginx-plus-go-client to stop 404 errors in NGINX access logs by [@dhurley](https://github.com/dhurley) in [#495](https://github.com/nginx/agent/pull/495)

---
## Release [v2.30.0](https//github.com/nginx/agent/releases/tag/v2.30.0)

### 🐛 Bug Fixes

- Fix version for forked repo by [@dhurley](https://github.com/dhurley) in [#468](https://github.com/nginx/agent/pull/468)

- Fix integration tests by [@aphralG](https://github.com/aphralG) in [#478](https://github.com/nginx/agent/pull/478)

- Fix config apply by [@oliveromahony](https://github.com/oliveromahony) in [#480](https://github.com/nginx/agent/pull/480)

- deprecate system.mem.used.all metric by [@aphralG](https://github.com/aphralG) in [#485](https://github.com/nginx/agent/pull/485)

### 📝 Documentation

- Update CLI flags documentation by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#476](https://github.com/nginx/agent/pull/476)

- Update NGINX plugin to read NGINX config on startup by [@dhurley](https://github.com/dhurley) in [#489](https://github.com/nginx/agent/pull/489)

### 🔨 Maintenance

- Update file watcher to ignore .swx files by [@dhurley](https://github.com/dhurley) in [#466](https://github.com/nginx/agent/pull/466)

- Check Simplemetrics is not empty  by [@aphralG](https://github.com/aphralG) in [#474](https://github.com/nginx/agent/pull/474)

- Add error log if duplicate NGINX IDs are found by [@dhurley](https://github.com/dhurley) in [#477](https://github.com/nginx/agent/pull/477)

- Add tests for additional SSL directives and key algorithms. [#276](https://github.com/nginx/agent/issues/276) by [@arsenalzp](https://github.com/arsenalzp) in [#469](https://github.com/nginx/agent/pull/469)

- call underlying os.Hostname instead of the entire hostInfo gopsutil call by [@oliveromahony](https://github.com/oliveromahony) in [#479](https://github.com/nginx/agent/pull/479)

- Add grpc integration tests by [@dhurley](https://github.com/dhurley) in [#475](https://github.com/nginx/agent/pull/475)

- remove error log causing failures  by [@aphralG](https://github.com/aphralG) in [#488](https://github.com/nginx/agent/pull/488)

- Use singleflight for caching environment.go calls by [@oliveromahony](https://github.com/oliveromahony) in [#481](https://github.com/nginx/agent/pull/481)

- Reduce the number of times env.Processes gets called by [@dhurley](https://github.com/dhurley) in [#482](https://github.com/nginx/agent/pull/482)

- add additional check to nginxProcesses by [@aphralG](https://github.com/aphralG) in [#483](https://github.com/nginx/agent/pull/483)

- profile.cgo by [@oliveromahony](https://github.com/oliveromahony) in [#493](https://github.com/nginx/agent/pull/493)

---
## Release [v2.29.0](https//github.com/nginx/agent/releases/tag/v2.29.0)

### 🚀 Features
- Add metric sender feature by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#453](https://github.com/nginx/agent/pull/453)

### 🐛 Bug Fixes

- fix: fix logic for parsing absolute path to nginx config file by [@sylwang](https://github.com/sylwang) in [#445](https://github.com/nginx/agent/pull/445)

- Fix SELinux Policy & Fix SELinux README by [@aphralG](https://github.com/aphralG) in [#467](https://github.com/nginx/agent/pull/467)

- fix: ensure fullpath by [@nginx-nickc](https://github.com/nginx-nickc) in [#471](https://github.com/nginx/agent/pull/471)

### 📝 Documentation

- Remove Ubuntu 18.04, Alpine 3.13 and Alpine 3.14 OS support by [@dhurley](https://github.com/dhurley) in [#444](https://github.com/nginx/agent/pull/444)

- Go 1.21 by [@oliveromahony](https://github.com/oliveromahony) in [#459](https://github.com/nginx/agent/pull/459)

- Add Alpine 3.18 support by [@dhurley](https://github.com/dhurley) in [#443](https://github.com/nginx/agent/pull/443)

- Update selinux readme by [@dhurley](https://github.com/dhurley) in [#449](https://github.com/nginx/agent/pull/449)

- Add proto-buf definitions for php-fpm metrics by [@achawla2012](https://github.com/achawla2012) in [#452](https://github.com/nginx/agent/pull/452)

- Performance tests for loading of plugins and different feature combinations by [@oliveromahony](https://github.com/oliveromahony) in [#463](https://github.com/nginx/agent/pull/463)

### 🔨 Maintenance

- Add Debian 12 support by [@dhurley](https://github.com/dhurley) in [#442](https://github.com/nginx/agent/pull/442)

- Revert "Merge release-2.28.1 back into main ([#455](https://github.com/nginx/agent/pull/455))" by [@oliveromahony](https://github.com/oliveromahony) in [#457](https://github.com/nginx/agent/pull/457)

- Release 2.28.1 by [@oliveromahony](https://github.com/oliveromahony) in [#458](https://github.com/nginx/agent/pull/458)

- More benchmark tests by [@oliveromahony](https://github.com/oliveromahony) in [#462](https://github.com/nginx/agent/pull/462)

- Only create metrics sender if a metric reporter is already created by [@dhurley](https://github.com/dhurley) in [#465](https://github.com/nginx/agent/pull/465)

- Register php-fpm metrics as extension plugin by [@achawla2012](https://github.com/achawla2012) in [#451](https://github.com/nginx/agent/pull/451)

- Add support for file actions during a config apply by [@dhurley](https://github.com/dhurley) in [#464](https://github.com/nginx/agent/pull/464)

- Add TLS upstream and TLS server_zone metrics by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#470](https://github.com/nginx/agent/pull/470)

- Added cgo profile to build NGINX Agent packages by [@oliveromahony](https://github.com/oliveromahony) in [#472](https://github.com/nginx/agent/pull/472)

- Add worker conn metrics by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#461](https://github.com/nginx/agent/pull/461)

---
## Release [v2.28.1](https//github.com/nginx/agent/releases/tag/v2.28.1)

### 🐛 Bug Fixes

- fix for dash during upgrade by [@oliveromahony](https://github.com/oliveromahony) in [#450](https://github.com/nginx/agent/pull/450)

- Improve status API detection and validation by [@dhurley](https://github.com/dhurley) in [#447](https://github.com/nginx/agent/pull/447)

- Rebuild selinux policy on RHEL 8 by [@dhurley](https://github.com/dhurley) in [#448](https://github.com/nginx/agent/pull/448)
