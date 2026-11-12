#!/usr/bin/env python3
# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Generate the compiled Krenalis catalog from the supplied source archives.

Uses only the standard library and the documented-profile helpers delivered
with the property catalog. Does not modify either archive or its JSON Schemas.
"""

import argparse
import hashlib
import json
from pathlib import Path
import re
import runpy
import subprocess
import tempfile
import zipfile

from native_type_choices import TYPE_CHOICES


HERE = Path(__file__).resolve().parent
OUTPUT = HERE.parents[1] / "core/internal/state/event_catalog_data.go"
TEXTS = HERE / "event_texts_en.json"
PROPERTY_NAME = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")


def archive_files(name):
    """Check archive consistency, not authenticity, against its own manifest.

    The archives and executable helpers must already be trusted repository inputs.
    """
    with zipfile.ZipFile(HERE / name) as archive:
        files = {name: archive.read(name) for name in archive.namelist()}
    for line in files["MANIFEST.sha256"].decode().splitlines():
        expected, name = line.split(maxsplit=1)
        if hashlib.sha256(files[name.lstrip("*")]).hexdigest() != expected:
            raise ValueError(f"Source manifest mismatch: {name}")
    return files


def field(name, value, indent):
    """Wrap Go string literals without changing their contents."""
    prefix = "\t" * indent + name + ": "
    quoted = json.dumps(value, ensure_ascii=False)
    if len(prefix.expandtabs(4)) + len(quoted) + 1 <= 120:
        return prefix + quoted + ","
    parts = []
    while len(value) > 75:
        split = value.rfind(" ", 0, 75)
        if split <= 0:
            split = 75
        else:
            split += 1
        parts.append(value[:split])
        value = value[split:]
    parts.append(value)
    return prefix + (" +\n" + "\t" * (indent + 1)).join(
        json.dumps(part, ensure_ascii=False) for part in parts
    ) + ","


def properties_type(nodes, event_id, selected_types, indent):
    declarations = []
    for node in nodes:
        # JSON keys outside the native property-name grammar remain untouched
        # as undeclared JSON; never rename them to manufacture native fields.
        if not PROPERTY_NAME.fullmatch(node["name"]):
            continue
        native = native_type(node, event_id, selected_types, indent + 1)
        pad = "\t" * (indent + 1)
        lines = [pad + "{", field("Name", node["name"], indent + 2)]
        lines.append("\t" * (indent + 2) + "Type: " + native + ",")
        lines.append("\t" * (indent + 2) + "ReadOptional: true,")
        if "null" in selected_types(node, "documented"):
            lines.append("\t" * (indent + 2) + "Nullable: true,")
        lines.append(field("Description", node["description"], indent + 2))
        lines.append(pad + "},")
        declarations.extend(lines)
    if not declarations:
        return "types.Map(types.JSON())"
    return "types.Object([]types.Property{\n" + "\n".join(declarations) + "\n" + "\t" * indent + "})"


def native_type(node, event_id, selected_types, indent):
    """Choose the native value type, independently of future input conversions."""
    kinds = set(selected_types(node, "documented")) - {"null"}
    choice = TYPE_CHOICES.get((event_id, node["path"]))
    if choice:
        expected, native = choice
        if kinds != set(expected):
            raise ValueError(f"Source kinds changed for {event_id} {node['path']}: {sorted(kinds)}")
        return native
    if len(kinds) > 1:
        raise ValueError(f"Unreviewed mixed kinds for {event_id} {node['path']}: {sorted(kinds)}")
    if not kinds:
        return "types.JSON()"
    if kinds == {"object"}:
        return properties_type(node["properties"], event_id, selected_types, indent)
    if kinds == {"array"}:
        item = node.get("items")
        element = native_type(item, event_id, selected_types, indent) if item else "types.JSON()"
        # Type has no nullable array-element flag. Keep such elements as JSON.
        if item and "null" in selected_types(item, "documented"):
            element = "types.JSON()"
        return "types.Array(" + element + ")"
    return {
        "boolean": "types.Boolean()", "string": "types.String()",
        "integer": "types.Int(32)", "number": "types.Float(64).Real()",
    }[next(iter(kinds))]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check generated Go without modifying it")
    args = parser.parse_args()
    taxonomy_files = archive_files("cdp-event-taxonomy.zip")
    property_files = archive_files("cdp-event-properties.zip")
    for name in ("cdp-event-catalog.json", "cdp-event-bindings.json"):
        if property_files["baseline/" + name] != taxonomy_files[name]:
            raise ValueError(f"Catalog baselines differ: {name}")
    taxonomy = json.loads(taxonomy_files["cdp-event-catalog.json"])
    catalog = json.loads(property_files["cdp-event-properties.json"])
    events = {event["event_id"]: event for event in catalog["events"]}
    if len(events) != 102 or set(events) != {event["id"] for event in taxonomy["definitions"]}:
        raise ValueError("Expected the same 102 unique definitions in both sources")

    texts = json.loads(TEXTS.read_text())
    if not isinstance(texts, dict) or set(texts) != set(events):
        raise ValueError("Expected English texts for exactly the source event IDs")
    for event_id, text in texts.items():
        if (
            not isinstance(text, dict) or set(text) != {"display_name", "description"}
            or any(not isinstance(value, str) or not value.strip() for value in text.values())
        ):
            raise ValueError(f"Expected nonempty English display_name and description for {event_id}")

    with tempfile.TemporaryDirectory(prefix="krenalis-event-catalog-") as directory:
        helper = Path(directory) / "generate_event_schemas.py"
        helper.write_bytes(property_files["generate_event_schemas.py"])
        source = runpy.run_path(str(helper))
    selected_types = source["selected_types"]
    schemas = {}
    schema_names = {}
    definitions = []
    for definition in taxonomy["definitions"]:
        event_id = definition["id"]
        event = events[event_id]
        branches = definition["match"]["any_of"]
        if len(branches) != 1 or any(test["op"] != "eq" for test in branches[0]["all"]):
            raise ValueError(f"Unsupported source matcher: {event_id}")
        match = {test["field"]: test["value"] for test in branches[0]["all"]}
        if set(match) - {"type", "event", "name"}:
            raise ValueError(f"Unsupported source matcher fields: {event_id}")
        nodes = source["expand_event"](catalog, event)
        roots = {node["name"]: node for node in nodes}
        traits = roots.get("traits")
        if match["type"] not in {"identify", "group"}:
            context = roots.get("context", {})
            traits = next((n for n in context.get("properties", []) if n["name"] == "traits"), None)
        roots = {"Traits": traits, "Properties": roots.get("properties")}
        native_schemas = {}
        for kind, root in roots.items():
            if not root or not root.get("properties"):
                continue
            native = properties_type(root["properties"], event_id, selected_types, 0)
            if native not in schema_names:
                words = re.split(r"[._]", event_id)
                name = words[0] + "".join(word.title() for word in words[1:]) + kind
                if (
                    kind == "Traits" and match["type"] not in {"identify", "group"}
                    and "eventContextTraits" not in schemas
                ):
                    name = "eventContextTraits"
                if name in schemas:
                    raise ValueError(f"Conflicting generated schema name: {name}")
                schema_names[native] = name
                schemas[name] = native
            native_schemas[kind] = schema_names[native]
        lines = ["\t{"]
        for key, value in (
            ("ID", event_id), ("Type", match["type"]), ("Event", match.get("event", "")),
            ("Name", match.get("name", "")), ("DisplayName", texts[event_id]["display_name"]),
            ("Description", texts[event_id]["description"]),
        ):
            if value:
                lines.append(field(key, value, 2))
        if native_schemas:
            lines.append("\t\tSchemas: EventSchemas{")
            lines.extend("\t\t\t" + name + ": " + native + "," for name, native in native_schemas.items())
            lines.append("\t\t},")
        lines.append("\t},")
        definitions.extend(lines)

    header = """// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Code generated by filtering/event-catalog/generate_catalog.py; DO NOT EDIT.

package state

import "github.com/krenalis/krenalis/tools/types"

"""
    declarations = [f"var {name} = {native}\n" for name, native in schemas.items()]
    data = header + "\n".join(declarations)
    data += "\n// eventDefinitions is shared by all workspaces and is never modified.\n"
    data += "var eventDefinitions = []EventDefinition{\n" + "\n".join(definitions) + "\n}\n"
    with tempfile.TemporaryDirectory(prefix="krenalis-event-catalog-format-") as directory:
        formatted = Path(directory) / "catalog.go"
        formatted.write_text(data)
        subprocess.run(["gofmt", "-w", str(formatted)], check=True)
        data = formatted.read_bytes()
    if args.check:
        if OUTPUT.read_bytes() != data:
            raise SystemExit("Compiled catalog differs from the generator; regenerate it")
    else:
        OUTPUT.write_bytes(data)
    print(f"Generated {len(events)} event definitions and {len(schemas)} distinct schemas: {OUTPUT}")


if __name__ == "__main__":
    main()
