# Release upgrade

Base release: `v0.46.0`

What the updater of the next release must do to upgrade a database of the
base release. See `.agents/skills/release-upgrade/SKILL.md`.

- Account for typed operands in `pipelines.filter` and pipeline state
  notifications. JSON property operands now preserve their JSON kind instead
  of storing a string interpreted as both text and, optionally, a number.
  Existing JSON filter operands require an explicit decision about their
  intended kind before release. This change provides no conversion or legacy
  reader; the release must address persisted filters before using the new
  format. Ordinary integer operands remain JSON numbers at every bit size.
- Remove `workspaces.warehouse_mcp_settings` and
  `workspaces.kms_encrypted_warehouse_mcp_settings_key`, discarding the stored
  warehouse MCP credentials and their dedicated encryption key without
  conversion. Preserve access keys of type MCP. Perform the upgrade offline:
  stop all nodes before removing the columns, then restart all nodes on the
  same new version; old nodes still write both columns. MCP profile access now
  uses the main connection for every valid workspace-scoped MCP key, including
  existing keys without warehouse MCP settings and keys formerly restricted
  by read-only warehouse grants. Document this access change in the release
  notes so administrators can review and revoke existing keys as needed.
