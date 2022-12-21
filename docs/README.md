# agent repo

This folder houses the product documentation for the NGINX Agent:

Directory Tree:

```shell
└──docs
```

## Where to Find Docs Online

The public docs will be available online at [docs.nginx.com/nginx-agent](https://docs.nginx.com/nginx-agent).

## Contributing

Docs are written in Markdown. We build the docs using [Hugo](https://gohugo.io) and host them on [Netlify](https://www.netlify.com/).

We use a set of pre-defined content types and templates, which can help you get started when working on new docs:

The content templates are based on the [Kubernetes docs templates](https://kubernetes.io/docs/contribute/style/page-content-types/).

- Task: Informs the customer of the steps they must complete to achieve a specific goal.
- Concept: Teaches customers about a product, architectural design, feature, or feature group.
- Reference: Describes an API, command line tool, etc.  
- Troubleshooting Guide: Addresses a specific issue that customers commonly face.
- Tutorial: Helps customers accomplish a goal that may encompass several tasks; provides context, instructions, and detailed examples.

> Refer to [Add new docs](#add-new-docs) to learn how to create new docs using Hugo.

## Branch Management

In this repo, the "main" branch should always be releasable. This means that **only content that has been reviewed and approved should be merged into main**.

To contribute to the docs:

1. Create a feature branch from main.
2. Merge to main to publish.

## Tools 

In this directory, you will find the following files that support the tools we use to lint, build, and deploy the docs:

- configuration files for [markdownlint](https://github.com/DavidAnson/markdownlint/) and [markdown-link-check](https://github.com/tcort/markdown-link-check);
- a [`config`](./config/) directory that contains the [Hugo](https://gohugo.io) configuration. Each sub-directory represents a different Hugo build environment (e.g., staging, production);
  > [Learn more about Hugo configuration](https://gohugo.io/getting-started/configuration/#configuration-directory) 
- a [Netlify](https://netlify.com) configuration file.

### Setup

1. To install Hugo locally, refer to the [Hugo installation instructions](https://gohugo.io/getting-started/installing/).

    > **NOTE**: We don't support versions newer than v0.91 yet, so we recommend using the [Binary](https://gohugo.io/getting-started/installing/#binary-cross-platform) installation option.

2. We use markdownlint to check that Markdown files are correct. Use `npm` to install `markdownlint-cli` if you want to lint the files locally.

    ```
    npm i -g markdownlint-cli   
    ```

### Hugo Theme

The docs rely on the f5-hugo theme (hosted in GitLab) for the page layouts.
The theme is imported as Hugo module (essentially the same thing as go mods), via the [default docs config](./_default/config.toml).
The theme must be vendored (`hugo mod vendor`) every time it is updated for Netlify to be able to build the docs.

### Local Docs Development

To build the docs locally, run the desired `make` command from the docs directory:

```text
make clean          -   removes the local `public` directory, which is the default output path used by Hugo
make docs           -   runs a local hugo server so you can view docs in your browser while you work
make docs-drafts    -   runs the local hugo server and includes all docs marked with `draft: true`
make hugo-mod       -   cleans the Hugo module cache and fetches the latest version of the theme module
make docs-local     -   runs the `hugo` command to generate static HTML files into the `public` directory
```

### Linting

- To run the style and grammar check, run the following command from the docs directory:

<!-- Todo: add VALE local steps -->

- To run the markdownlint check, run the following command from the docs directory:

    ```bash
    markdownlint -c .gitlab/ci/markdown_lint_config.json content    
    ```

    **Note**: You can run this tool on an entire directory or an individual file.

## Add new docs

### Generate a new doc file using Hugo

To create a new doc file that contains all of the pre-configured Hugo front-matter and the docs task template, **run the following command in the docs directory**:

`hugo new <SECTIONNAME>/<FILENAME>.<FORMAT>`

e.g.,

`hugo new getting-started/install.md`

The default template -- task -- should be used for most docs. To docs from the other content templates, you can use the `--kind` flag:

`hugo new tutorials/deploy.md --kind tutorial`

The available content types (`kind`) are:

- concept: Helps a customer learn about a specific feature or feature set.
- tutorial: Walks a customer through an example use case scenario; results in a functional PoC environment.
- reference: Describes an API, command line tool, config options, etc.; should be generated automatically from source code. 
- troubleshooting: Helps a customer solve a specific problem.
- openapi: Contains front-matter and shortcode for rendering an openapi.yaml spec

## How to format docs

### Internal links

Format links as [Hugo refs](https://gohugo.io/content-management/cross-references/). 

- File extensions are optional.
- You can use relative paths or just the filename. (**Paths without a leading / are first resolved relative to the current page, then to the remainder of the site.**)
- Anchors are supported.

For example:

```md
To install <product>, refer to the [installation instructions]({{< ref "install" >}}).
```

### Hugo shortcodes

You can use [Hugo shortcodes](/docs/themes/f5-hugo/layouts/shortcodes/) to do things like format callouts, add images, and reuse content across different docs. 

For example, to use the note callout:

```md
{{< note >}}Provide the text of the note here. {{< /note >}}
```

The callout shortcodes also support multi-line blocks:

```md
{{< caution >}}
You should probably never do this specific thing in a production environment. 

If you do, and things break, don't say we didn't warn you.
{{< /caution >}}
```

Supported callouts:

- `caution`
- `important`
- `note`
- `see-also`
- `tip`
- `warning`

A few more fun shortcodes:

- `fa`: inserts a Font Awesome icon
- `img`: include an image and define things like alt text and dimensions
- `include`: include the content of a file in another file (requires the included file to be in the content/includes directory; will be deprecated in favor of readfile)
- `link`: makes it possible to link to a file and prepend the path with the Hugo baseUrl
- `openapi`: loads an OpenAPI spec and renders as HTML using ReDoc
- `raw-html`: makes it possible to include a block of raw HTML
- `readfile`: includes the content of another file in the current file (intended to replace `include`)
- `bootstrap-table`: formats a table using Bootstrap classes; accepts any bootstrap table classes as additional arguments, e.g. `{{< bootstrap-table "table-bordered table-hover" }}`
