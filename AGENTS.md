# Project Rules

## Encoding

- Read `doc/encoding.md` before editing text files in this repository.
- Keep source files, docs, JSON, YAML, SQL, and generated docs as UTF-8.
- Prefer `apply_patch` for manual edits.
- Do not rewrite text files with PowerShell `Set-Content` or `Out-File`.
- Do not pipe `Get-Content` back into `Set-Content` or `Out-File`.
- If a script must write text files, write UTF-8 without BOM explicitly.
- Be extra careful with `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` because they are generated and may contain non-ASCII text.
