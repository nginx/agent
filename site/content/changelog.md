---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

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

- change log updated for last release by [@oliveromahony](https://github.com/oliveromahony) in [#583](https://github.com/nginx/agent/pull/583)- Restore agent container information from nms docs  by [@jputrino](https://github.com/jputrino) in [#584](https://github.com/nginx/agent/pull/584)- fix: add additional container checks during instance registration by [@mattdesmarais](https://github.com/mattdesmarais) in [#592](https://github.com/nginx/agent/pull/592)

---
## Release [v2.32.1](https//github.com/nginx/agent/releases/tag/v2.32.1)

### 🚀 Features

This release introduces the following new features:

- feat: Agent Docs IA refactor by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#548](https://github.com/nginx/agent/pull/548)- feat: move NMS agent docs by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#553](https://github.com/nginx/agent/pull/553)- feat: import changelog from github by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#570](https://github.com/nginx/agent/pull/570)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix runners and bump go version by [@oliveromahony](https://github.com/oliveromahony) in [#550](https://github.com/nginx/agent/pull/550)- Fix artifact name by [@oliveromahony](https://github.com/oliveromahony) in [#558](https://github.com/nginx/agent/pull/558)- fix: add missing catalog entry by [@Jcahilltorre](https://github.com/Jcahilltorre) in [#572](https://github.com/nginx/agent/pull/572)

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

- Update upgrade documentation by [@dhurley](https://github.com/dhurley) in [#526](https://github.com/nginx/agent/pull/526)- Bump the versions of containerd and go-git dependencies by [@dhurley](https://github.com/dhurley) in [#533](https://github.com/nginx/agent/pull/533)- updated dependencies by [@oliveromahony](https://github.com/oliveromahony) in [#536](https://github.com/nginx/agent/pull/536)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Bump crypto dependency from 0.14.0 to 0.17.0 by [@dhurley](https://github.com/dhurley) in [#532](https://github.com/nginx/agent/pull/532)
---
## Release [v2.31.1](https//github.com/nginx/agent/releases/tag/v2.31.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix permissions for log file and dynamic config directory by [@aphralG](https://github.com/aphralG) in [#517](https://github.com/nginx/agent/pull/517)- Fix server example in sdk to have timeout by [@aphralG](https://github.com/aphralG) in [#518](https://github.com/nginx/agent/pull/518)

### 📝 Documentation

We have made the following updates to the documentation:

- Update SELinux Readme by [@aphralG](https://github.com/aphralG) in [#522](https://github.com/nginx/agent/pull/522)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Replace mockgen by [@oliveromahony](https://github.com/oliveromahony) in [#524](https://github.com/nginx/agent/pull/524)- Restrict config apply directory permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#519](https://github.com/nginx/agent/pull/519)- Restrict NAP file/dir permissions by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#516](https://github.com/nginx/agent/pull/516)
---
## Release [v2.31.0](https//github.com/nginx/agent/releases/tag/v2.31.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix otelcontrib version by [@oliveromahony](https://github.com/oliveromahony) in [#504](https://github.com/nginx/agent/pull/504)- Fix user agent request header to have the correct agent version by [@dhurley](https://github.com/dhurley) in [#498](https://github.com/nginx/agent/pull/498)- Fix alpine plus dockerfile on alpine>=3.17 by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#511](https://github.com/nginx/agent/pull/511)- fix: avoid stopping nginx-agent service on package upgrade by [@defanator](https://github.com/defanator) in [#352](https://github.com/nginx/agent/pull/352)- Fix SELinux Policy by [@aphralG](https://github.com/aphralG) in [#520](https://github.com/nginx/agent/pull/520)

### 📝 Documentation

We have made the following updates to the documentation:

- Add CLI arg to set dynamic config path by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#490](https://github.com/nginx/agent/pull/490)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- crossplane version bump by [@oliveromahony](https://github.com/oliveromahony) in [#512](https://github.com/nginx/agent/pull/512)- Add commander retry lock by [@dhurley](https://github.com/dhurley) in [#502](https://github.com/nginx/agent/pull/502)- Bump otel dependency version and fix github workflow for dependabot PRs by [@dhurley](https://github.com/dhurley) in [#515](https://github.com/nginx/agent/pull/515)
---
## Release [v2.30.3](https//github.com/nginx/agent/releases/tag/v2.30.3)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix dependabot issues by [@oliveromahony](https://github.com/oliveromahony) in [#503](https://github.com/nginx/agent/pull/503)

---
## Release [v2.30.1](https//github.com/nginx/agent/releases/tag/v2.30.1)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- fix: Tolerate additional fields in App Protect yaml files by [@edarzins](https://github.com/edarzins) in [#494](https://github.com/nginx/agent/pull/494)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Update nginx-plus-go-client to stop 404 errors in NGINX access logs by [@dhurley](https://github.com/dhurley) in [#495](https://github.com/nginx/agent/pull/495)
---
## Release [v2.30.0](https//github.com/nginx/agent/releases/tag/v2.30.0)

### 🐛 Bug Fixes

In this release we have resolved the following issues:

- Fix version for forked repo by [@dhurley](https://github.com/dhurley) in [#468](https://github.com/nginx/agent/pull/468)- Fix integration tests by [@aphralG](https://github.com/aphralG) in [#478](https://github.com/nginx/agent/pull/478)- Fix config apply by [@oliveromahony](https://github.com/oliveromahony) in [#480](https://github.com/nginx/agent/pull/480)- deprecate system.mem.used.all metric by [@aphralG](https://github.com/aphralG) in [#485](https://github.com/nginx/agent/pull/485)

### 📝 Documentation

We have made the following updates to the documentation:

- Update CLI flags documentation by [@Dean-Coakley](https://github.com/Dean-Coakley) in [#476](https://github.com/nginx/agent/pull/476)- Update NGINX plugin to read NGINX config on startup by [@dhurley](https://github.com/dhurley) in [#489](https://github.com/nginx/agent/pull/489)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Update file watcher to ignore .swx files by [@dhurley](https://github.com/dhurley) in [#466](https://github.com/nginx/agent/pull/466)- Check Simplemetrics is not empty  by [@aphralG](https://github.com/aphralG) in [#474](https://github.com/nginx/agent/pull/474)- Add error log if duplicate NGINX IDs are found by [@dhurley](https://github.com/dhurley) in [#477](https://github.com/nginx/agent/pull/477)- Add tests for additional SSL directives and key algorithms. [#276](https://github.com/nginx/agent/issues/276) by [@arsenalzp](https://github.com/arsenalzp) in [#469](https://github.com/nginx/agent/pull/469)- call underlying os.Hostname instead of the entire hostInfo gopsutil call by [@oliveromahony](https://github.com/oliveromahony) in [#479](https://github.com/nginx/agent/pull/479)- Add grpc integration tests by [@dhurley](https://github.com/dhurley) in [#475](https://github.com/nginx/agent/pull/475)- remove error log causing failures  by [@aphralG](https://github.com/aphralG) in [#488](https://github.com/nginx/agent/pull/488)- Use singleflight for caching environment.go calls by [@oliveromahony](https://github.com/oliveromahony) in [#481](https://github.com/nginx/agent/pull/481)- Reduce the number of times env.Processes gets called by [@dhurley](https://github.com/dhurley) in [#482](https://github.com/nginx/agent/pull/482)- add additional check to nginxProcesses by [@aphralG](https://github.com/aphralG) in [#483](https://github.com/nginx/agent/pull/483)- profile.cgo by [@oliveromahony](https://github.com/oliveromahony) in [#493](https://github.com/nginx/agent/pull/493)

