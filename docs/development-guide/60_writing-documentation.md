---
title: Writing Documentation
summary: Some notes on writing documentation for this site
authors:
  - hikarukin
date: 2024-11-27
---

# Writing Documentation

Our project utilizes a structured approach to documentation to ensure clarity and
ease of access for all users. All documentation is written in Markdown and stored
within the `docs` directory of the repository. The first level of directories under
`docs` corresponds to the main sections of our documentation site:

- `overview`
- `administrators-guide`
- `user-guide`
- `development-guide`
- `about`

We use [MkDocs](https://www.mkdocs.org/) along with the [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
theme to generate a professional and user-friendly documentation site from this
structure.

## Partially generated documentation

While most of the documentation is hand written by the core team or contributors,
the [API documentation](../development-guide/00_api.md) in the development guide
section is auto-generated using the `crd-ref-docs` tool from Elastic.

In the future we'll look into running this from our pipeline, but for now it should
be manually executed.

Assuming you're located in the root of the repository, execute the following to
update the API document.

```bash
go install github.com/elastic/crd-ref-docs
crd-ref-docs --config=./crd-ref-docs-config.yml --source-path=./api --renderer=markdown --output-path=./docs/development-guide/00_api.md
```

## Writing Clear and Effective Documentation

When contributing to the documentation, please aim to keep your writing simple
and factual. Clear and concise documentation helps users understand and utilize
our project more effectively.

### Tips for Writing Good Documentation

**Be Concise and Direct**:
Use straightforward language and get to the point quickly. Avoid unnecessary words
or overly complex sentences.

**Use Active Voice**:
Write in active voice to make your writing more engaging and easier to understand.
For example, we would prefer the use of "Install the package using..." over
"The package can be installed using...".

**Organize Content Logically**:
Break down information into logical sections and use headings and subheadings to
guide the reader. This makes it easier for users to find the information they need.

Ensure that you place your documentation in the right sub-section for your intended
reader.

**Use Lists and Bullet Points**:
When presenting multiple items or steps, use lists to improve readability.

**Include Examples and Call Outs**:
Provide code snippets or command-line examples where applicable to illustrate
your points.

!!! tip
    Use call outs like this to highlight important information.
    You can use "note", "abstract", "info", "tip", "success", "question", "warning",
    "failure", "danger", "bug", "example" and "quote".
    
    See [the docs](https://squidfunk.github.io/mkdocs-material/reference/admonitions/#supported-types) for more information.

**Maintain Consistent Formatting**:
Try to follow the existing style and formatting conventions used in the rest of
the documentation. This includes heading styles, code block formatting, and emphasis.

**Proofread Your Work**:
Check for spelling and grammar errors before submitting. Reading your text aloud
can help identify awkward phrasing or mistakes.

By following these guidelines, you'll help maintain a high standard of quality in
our documentation, making it a valuable resource for everyone involved in the project.
