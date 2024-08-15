---
title: "Changelog"
weight: 1200
toc: true
docs: "DOCS-1093"
---

{{< note >}}You can find the full changelog, contributor list and assets for NGINX Agent in the [GitHub repository](https://github.com/nginx/agent/releases).{{< /note >}}

See the list of supported Operating Systems and architectures in the [Technical Specifications]({{< relref "./technical-specifications.md" >}}).

---
## Release [v1.3.0](https//github.com/nginx/agent/releases/tag/v1.3.0)

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
## Release [v1.2.0](https//github.com/nginx/agent/releases/tag/v1.2.0)

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
## Release [v1.1.0](https//github.com/nginx/agent/releases/tag/v1.1.0)

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
## Release [v1.0.0](https//github.com/nginx/agent/releases/tag/v1.0.0)

### 🌟 Highlights

- Upgrade crossplane version to prevent Agent from rolling back in the case of valid NGINX configurations by [@oliveromahony](https://github.com/oliveromahony) in [agent#746](https://github.com/nginx/agent/pull/746)

### 🔨 Maintenance

We have made the following maintenance-related minor changes:

- Added version regex to parse the logs to see if matches vsemvar format by [@oliveromahony](https://github.com/oliveromahony) in [agent#747](https://github.com/nginx/agent/pull/747)

