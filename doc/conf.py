import datetime
import subprocess

project = "incus-apply"
author = "incus-apply contributors"
copyright = f"2026-{datetime.date.today().year} {author}"

try:
    version = subprocess.check_output(
        ["git", "describe", "--tags", "--always", "--dirty"],
        text=True,
    ).strip()
except Exception:
    version = "dev"

release = version

extensions = [
    "myst_parser",
    "sphinx_copybutton",
    "sphinx_design",
    "sphinx.ext.intersphinx",
    "sphinx_reredirects",
    "sphinx_tabs.tabs",
    "sphinxext.opengraph",
]

myst_enable_extensions = ["colon_fence", "deflist", "linkify"]
myst_heading_anchors = 4
myst_linkify_fuzzy_links = False
myst_url_schemes = {
    "http": None,
    "https": None,
}

templates_path = []
exclude_patterns = ["html", ".sphinx", ".DS_Store"]
source_suffix = {
    ".md": "markdown",
}

html_theme = "furo"
html_title = "incus-apply"
html_show_sphinx = False
html_last_updated_fmt = ""
html_static_path = [".sphinx/_static"]
html_css_files = ["custom.css", "furo_colors.css"]

html_theme_options = {
    "sidebar_hide_name": True,
}

html_context = {
    "github_url": "https://github.com/abiosoft/incus-apply",
    "github_version": "main",
    "github_folder": "/doc/",
    "github_filetype": "md",
}

intersphinx_mapping = {
    "incus": ("https://linuxcontainers.org/incus/docs/main/", None),
}

redirects = {
    "reference/configuration-reference": "configuration/",
}

ogp_site_name = "incus-apply documentation"
linkcheck_ignore = [
    r"https://raw.githubusercontent.com/.+",
]
linkcheck_anchors_ignore_for_url = [r"https://github\.com/.*"]
