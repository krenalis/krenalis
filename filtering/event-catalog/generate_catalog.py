#!/usr/bin/env python3
# Copyright 2026 Open2b. All rights reserved.
# Use of this source code is governed by an Elastic License 2.0
# that can be found in the LICENSE file.

"""Generate the compiled Krenalis catalog from the supplied source archives.

Uses only the standard library and the documented-profile helpers delivered
with the property catalog. Does not modify either archive or its JSON Schemas.
"""

import hashlib
import json
from pathlib import Path
import re
import runpy
import subprocess
import tempfile
import zipfile


HERE = Path(__file__).resolve().parent
OUTPUT = HERE.parents[1] / "core/internal/state/event_catalog_data.go"
PROPERTY_NAME = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
MONEY_FIELDS = {
    "amount", "discount", "price", "revenue", "shipping", "subtotal", "tax", "total",
}
INTEGER_FIELDS = {
    "bitrate", "employees", "logins", "pod_length", "pod_position", "position", "quartile", "step",
}
DURATION_FIELDS = {"conversation_duration", "total_length"}


def archive_files(name):
    """Check the source manifest before reading any executable helpers."""
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
    name = node["name"]
    if not kinds:
        # Both providers document the logical Date despite leaving its wire
        # representation unspecified; Krenalis uses its normal Date encoding.
        if name == "birthday" and "logical type Date" in node["description"]:
            return "types.Date()"
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
    if kinds == {"boolean"}:
        return "types.Boolean()"
    if kinds <= {"number", "integer", "string"}:
        if name in MONEY_FIELDS or name == "value" and (
            "Monetary value" in node["description"] or "Checkout value" in node["description"]
        ):
            return "types.Decimal(18, 4).AsMoney()"
        if name == "quantity":
            return "types.Decimal(18, 4)"
        if name == "employees":
            return "types.Int(32)"
        if kinds == {"string"} or name in {"rating", "value"}:
            if name == "email" or name.endswith("_email"):
                return "types.String().AsEmail()"
            if name in {"avatar", "website", "url"} or name.endswith("_url"):
                return "types.String().AsURL()"
            if name in {"createdAt", "created_at"} and event_id.startswith("identity."):
                return "types.DateTime()"
            return "types.String()"
        if name in DURATION_FIELDS:
            return "types.Float(64).Real().AsDuration(types.Second)"
        if event_id.startswith("video.") and name in {"position", "seek_position"}:
            # Measurement/Second will be added only after its separate change
            # has been integrated; these values already use the chosen kind.
            return "types.Float(64).Real()"
        if kinds == {"integer"} or name in INTEGER_FIELDS:
            return "types.Int(32)"
        return "types.Float(64).Real()"
    # String/Object and String/Array describe genuinely different structures.
    return "types.JSON()"


def main():
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
            ("Name", match.get("name", "")), ("DisplayName", definition["display_name"]),
            ("Description", definition["description"]),
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
    OUTPUT.write_text(data)
    subprocess.run(["gofmt", "-w", str(OUTPUT)], check=True)
    print(f"Generated {len(events)} event definitions and {len(schemas)} distinct schemas: {OUTPUT}")


if __name__ == "__main__":
    main()
