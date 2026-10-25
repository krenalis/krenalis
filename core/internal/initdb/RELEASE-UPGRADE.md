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
