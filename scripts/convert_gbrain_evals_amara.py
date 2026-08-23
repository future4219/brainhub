#!/usr/bin/env python3
"""Convert gbrain-evals amara-life-v1 into one Markdown file per page."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath
from typing import Any


SOURCE_ID = "gbrain-evals-amara-v1"
EXPECTED_TYPES = {
    "calendar-event": 20,
    "email": 50,
    "slack": 300,
    "meeting": 8,
    "note": 40,
}
EXPECTED_DOCS = 6


def sha256(value: str) -> str:
    return hashlib.sha256(value.encode()).hexdigest()


def canonical_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def safe_target(root: Path, slug: str) -> Path:
    path = PurePosixPath(slug)
    if path.is_absolute() or not path.parts or any(part in {"", ".", ".."} for part in path.parts):
        raise ValueError(f"unsafe slug: {slug!r}")
    if not re.fullmatch(r"[a-z0-9][a-z0-9._/-]*", slug):
        raise ValueError(f"unsupported slug: {slug!r}")
    return root.joinpath(*path.parts[:-1], path.parts[-1] + ".md")


def frontmatter(values: dict[str, Any]) -> str:
    lines = ["---"]
    lines.extend(f"{key}: {json.dumps(value, ensure_ascii=False, separators=(',', ':'))}" for key, value in values.items())
    lines.append("---")
    return "\n".join(lines) + "\n"


def read_jsonl(path: Path) -> dict[str, dict[str, Any]]:
    rows = [json.loads(line) for line in path.read_text().splitlines() if line]
    by_slug = {row["slug"]: row for row in rows}
    if len(by_slug) != len(rows):
        raise ValueError(f"duplicate slug in {path}")
    return by_slug


def ical_to_iso(value: str) -> str:
    parsed = datetime.strptime(value, "%Y%m%dT%H%M%SZ").replace(tzinfo=timezone.utc)
    return parsed.isoformat(timespec="milliseconds").replace("+00:00", "Z")


def read_calendar(path: Path) -> dict[str, tuple[dict[str, Any], str]]:
    lines = path.read_bytes().decode().split("\r\n")
    events: dict[str, tuple[dict[str, Any], str]] = {}
    current: list[str] | None = None
    for line in lines:
        if line == "BEGIN:VEVENT":
            if current is not None:
                raise ValueError("nested VEVENT")
            current = [line]
        elif current is not None:
            current.append(line)
            if line == "END:VEVENT":
                fields: dict[str, Any] = {"attendees": []}
                for event_line in current[1:-1]:
                    key, value = event_line.split(":", 1)
                    if key == "UID":
                        fields["uid"] = value
                    elif key == "DTSTART":
                        fields["dtstart"] = ical_to_iso(value)
                    elif key == "DTEND":
                        fields["dtend"] = ical_to_iso(value)
                    elif key == "SUMMARY":
                        fields["summary"] = value
                    elif key.startswith("ATTENDEE;CN="):
                        fields["attendees"].append({"name": key.removeprefix("ATTENDEE;CN="), "email": value.removeprefix("mailto:")})
                    elif key == "LOCATION":
                        fields["location"] = value
                event_id = fields["uid"].split("@", 1)[0]
                slug = f"cal/{event_id}"
                fields["slug"] = slug
                events[slug] = (fields, "\n".join(current) + "\n")
                current = None
    if current is not None:
        raise ValueError("unterminated VEVENT")
    return events


def markdown_slug(markdown: str, path: Path) -> str:
    match = re.search(r"(?m)^slug:\s*([^\s]+)\s*$", markdown)
    if not match:
        raise ValueError(f"missing slug in {path}")
    return match.group(1)


def build_pages(input_root: Path, output_root: Path) -> tuple[dict[Path, str], Counter[str]]:
    manifest = json.loads((input_root / "corpus-manifest.json").read_text())
    if manifest.get("corpus_id") != "amara-life-v1" or manifest.get("license") != "MIT":
        raise ValueError("unexpected corpus identity")
    items = manifest.get("items", [])
    counts = Counter(item["type"] for item in items)
    if counts != Counter(EXPECTED_TYPES):
        raise ValueError(f"unexpected manifest counts: {dict(counts)}")

    emails = read_jsonl(input_root / "inbox/emails.jsonl")
    slack = read_jsonl(input_root / "slack/messages.jsonl")
    calendar = read_calendar(input_root / "calendar.ics")
    pages: dict[Path, str] = {}

    for item in items:
        slug = item["slug"]
        kind = item["type"]
        source_path = input_root / item["path"]
        if kind == "email":
            record = emails.pop(slug)
            if sha256(canonical_json(record)) != item["content_sha256"]:
                raise ValueError(f"manifest hash mismatch: {slug}")
            body = record.pop("body_text")
            record.pop("slug")
            content = frontmatter({"type": kind, "date": record.pop("ts"), **record}) + body
        elif kind == "slack":
            record = slack.pop(slug)
            if sha256(canonical_json(record)) != item["content_sha256"]:
                raise ValueError(f"manifest hash mismatch: {slug}")
            body = record.pop("text")
            record.pop("slug")
            content = frontmatter({"type": kind, "date": record.pop("ts"), **record}) + body
        elif kind == "calendar-event":
            event, body = calendar.pop(slug)
            if sha256(canonical_json(event)) != item["content_sha256"]:
                raise ValueError(f"manifest hash mismatch: {slug}")
            event.pop("slug")
            content = frontmatter({"type": kind, "date": event.pop("dtstart"), **event}) + body
        elif kind in {"meeting", "note"}:
            content = source_path.read_text()
            if sha256(content) != item["content_sha256"]:
                raise ValueError(f"manifest hash mismatch: {slug}")
        else:
            raise ValueError(f"unsupported type: {kind}")

        target = safe_target(output_root, slug)
        if target in pages:
            raise ValueError(f"duplicate output: {target}")
        pages[target] = content

    if emails or slack or calendar:
        raise ValueError("raw records and manifest do not match one-to-one")

    docs = sorted((input_root / "doc").glob("*.md"))
    if len(docs) != EXPECTED_DOCS:
        raise ValueError(f"expected {EXPECTED_DOCS} docs, found {len(docs)}")
    for source_path in docs:
        content = source_path.read_text()
        target = safe_target(output_root, markdown_slug(content, source_path))
        if target in pages:
            raise ValueError(f"duplicate output: {target}")
        pages[target] = content
        counts["doc"] += 1

    expected_total = sum(EXPECTED_TYPES.values()) + EXPECTED_DOCS
    if len(pages) != expected_total:
        raise ValueError(f"expected {expected_total} pages, built {len(pages)}")
    return pages, counts


def verify_output(output_root: Path, pages: dict[Path, str]) -> None:
    actual = {path for path in output_root.rglob("*.md") if path.name != "README.md"}
    if actual != set(pages):
        missing = sorted(str(path.relative_to(output_root)) for path in set(pages) - actual)
        extra = sorted(str(path.relative_to(output_root)) for path in actual - set(pages))
        raise ValueError(f"output mismatch: missing={missing}, extra={extra}")
    changed = [str(path.relative_to(output_root)) for path, content in pages.items() if path.read_text() != content]
    if changed:
        raise ValueError(f"content mismatch: {changed}")
    forbidden = [path for path in output_root.rglob("*") if path.is_file() and ("gold" in path.parts or path.suffix in {".json", ".jsonl", ".ics"})]
    if forbidden:
        raise ValueError(f"forbidden output: {forbidden}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()

    input_root = args.input.resolve()
    output_root = args.output.resolve()
    if input_root.name != "amara-life-v1":
        raise ValueError(f"input must be amara-life-v1, got {input_root}")
    if output_root.name != SOURCE_ID or not (output_root / ".git").is_dir() or not (output_root / "README.md").is_file():
        raise ValueError(f"output must be the provisioned {SOURCE_ID} git repo")

    pages, counts = build_pages(input_root, output_root)
    if not args.check:
        existing = [path for path in output_root.rglob("*.md") if path.name != "README.md"]
        if existing:
            raise ValueError(f"refusing to overwrite {len(existing)} existing corpus pages")
        for path, content in pages.items():
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)

    verify_output(output_root, pages)
    print(json.dumps({"source_id": SOURCE_ID, "page_count": len(pages), "counts": dict(sorted(counts.items())), "mode": "check" if args.check else "generate"}, ensure_ascii=False))


if __name__ == "__main__":
    main()
