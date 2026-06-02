# Contributing

Contributions are welcomed for the project.

## Contributing code

To contribute, you will need Go installed (see the [Go installation guide](https://golang.org/doc/install)).

### 1. Fork the Repository

First, fork the [repository](https://github.com/abiosoft/incus-apply) on GitHub. Then, clone your fork locally:

```sh
git clone https://github.com/<your-username>/incus-apply.git
cd colima
```

### 2. Create a New Branch

Create a new branch for your changes:

```sh
git checkout -b my-feature-branch
```

### 3. Commit Your Changes

Commit your changes with a DCO signoff and the required commit message format. Each commit must include a signoff line:

```
Signed-off-by: Your Name <your.email@example.com>
```

You can add this automatically with:

```sh
git commit -s -m "component: <message>"
# Example:
git commit -s -m "cli: add my-command to colima start"
```

### 4. Push Your Branch

Push your branch to your fork:

```sh
git push origin my-feature-branch
```

### 5. Open a Pull Request

Open a Pull Request against the main incus-apply repository.

### 6. DCO Signoff

All commits are required to be signed off using the [Developer Certificate of Origin (DCO)](https://developercertificate.org/). This is a simple statement that you, as a contributor, have the right to submit your changes.

### Reviewing and Merging

All PRs are subject to review.

Kindly ensure that your PR passes all CI checks. You are also obliged to respond to review comments and update your PR as needed.

## Contributing documentation

The documentation site uses Sphinx, MyST, and the Furo theme, aligned with other Incus-related projects.

Build the docs locally:

```bash
make doc
```

Serve the built docs locally:

```bash
make doc-serve
```

Then open `http://localhost:8001/`.

### Documentation contribution guidelines

- Run `make doc` before submitting changes to verify Sphinx rendering.
