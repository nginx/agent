---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

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

