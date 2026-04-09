# Editor Schema

`incus-apply` ships a JSON Schema for editor validation and autocomplete.

## Schema URL

```text
https://raw.githubusercontent.com/abiosoft/incus-apply/refs/heads/main/schema/incus-apply.schema.json
```

## VS Code Setup

To enable the schema with the Red Hat YAML extension, add this to `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/abiosoft/incus-apply/refs/heads/main/schema/incus-apply.schema.json": [
      "*.yaml",
      "*.yml"
    ]
  },
  "json.schemas": [
    {
      "fileMatch": ["*.json"],
      "url": "https://raw.githubusercontent.com/abiosoft/incus-apply/refs/heads/main/schema/incus-apply.schema.json"
    }
  ]
}
```

The generated schema file lives at [../schema/incus-apply.schema.json](../schema/incus-apply.schema.json).
