# Markdown to PDF Converter

Converts Markdown files to styled PDFs using Pandoc and headless Chromium. Runs with network disabled for security.

## How to run it

```bash
docker run --rm -t \
  --volume=$(pwd):/data/ \
  --workdir=/data/ \
  --network=none \
  ghcr.io/nickborgers/util/md-to-pdf:latest \
  input.md output.pdf
```

Or use the shell function from the [profile](../profile):

```bash
md_to_pdf report.md
md_to_pdf report.md custom-name.pdf
```

## How it works

Two-step conversion:
1. **Markdown → HTML** using Pandoc with embedded CSS
2. **HTML → PDF** using headless Chromium

This approach produces full-width pages with proper table styling, unlike wkhtmltopdf which produces narrow layouts.

## Custom styling

Place a `print-style.css` file in your working directory to override the default styles. The container will use your CSS if present.

## Gotcha: Tables After Bold Text

Pandoc requires a blank line between paragraphs and tables:

```markdown
<!-- Bad - table won't render -->
**Header:**
| Col1 | Col2 |

<!-- Good -->
**Header:**

| Col1 | Col2 |
```
