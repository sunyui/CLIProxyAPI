#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


MISSING = object()


def parse_args():
    parser = argparse.ArgumentParser(description="Diff consecutive CLIProxyAPI upstream request prefixes.")
    parser.add_argument(
        "--log-dir",
        default=str(Path.home() / ".cli-proxy-api" / "auths" / "logs"),
        help="Directory containing CLIProxyAPI request logs.",
    )
    parser.add_argument("--limit", type=int, default=6, help="Newest message logs to inspect.")
    parser.add_argument(
        "--normalize-billing-header",
        action="store_true",
        help="Replace volatile x-anthropic-billing-header cch values before diffing.",
    )
    return parser.parse_args()


def section(text, name):
    marker = f"=== {name} ==="
    lines = text.splitlines(keepends=True)
    start = None
    for index, line in enumerate(lines):
        if line.rstrip("\r\n") == marker:
            start = index + 1
            break
    if start is None:
        return ""
    end = len(lines)
    for index in range(start, len(lines)):
        if lines[index].startswith("=== "):
            end = index
            break
    return "".join(lines[start:end]).strip()


def json_from_block(block):
    block = block.strip()
    if not block:
        return None
    start = block.find("{")
    if start < 0:
        start = block.find("[")
    if start >= 0:
        block = block[start:]
    try:
        return json.loads(block)
    except json.JSONDecodeError:
        return None


def newest_logs(log_dir, limit):
    files = [p for p in Path(log_dir).glob("v1-messages-*.log") if "count_tokens" not in p.name]
    return sorted(files, key=lambda p: p.stat().st_mtime)[-limit:]


def normalize_billing_header(value):
    if isinstance(value, dict):
        return {key: normalize_billing_header(child) for key, child in value.items()}
    if isinstance(value, list):
        return [normalize_billing_header(child) for child in value]
    if isinstance(value, str) and value.startswith("x-anthropic-billing-header:"):
        parts = []
        for part in value.split(";"):
            stripped = part.strip()
            if stripped.startswith("cch="):
                parts.append(" cch=<normalized>")
            else:
                parts.append(part)
        return ";".join(parts)
    return value


def request_from_log(path, normalize_header=False):
    text = path.read_text(errors="replace")
    request = json_from_block(section(text, "API REQUEST 1"))
    if request is not None and normalize_header:
        request = normalize_billing_header(request)
    return request


def first_diff(left, right, path="$"):
    if type(left) is not type(right):
        return path, left, right
    if isinstance(left, dict):
        keys = sorted(set(left) | set(right))
        for key in keys:
            lval = left.get(key, MISSING)
            rval = right.get(key, MISSING)
            if lval is MISSING or rval is MISSING:
                return f"{path}.{key}", lval, rval
            found = first_diff(lval, rval, f"{path}.{key}")
            if found:
                return found
        return None
    if isinstance(left, list):
        for index, (lval, rval) in enumerate(zip(left, right)):
            found = first_diff(lval, rval, f"{path}[{index}]")
            if found:
                return found
        if len(left) != len(right):
            index = min(len(left), len(right))
            lval = left[index] if index < len(left) else MISSING
            rval = right[index] if index < len(right) else MISSING
            return f"{path}[{index}]", lval, rval
        return None
    if left != right:
        return path, left, right
    return None


def canonical(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def raw_prefix_chars(left, right):
    limit = min(len(left), len(right))
    for index in range(limit):
        if left[index] != right[index]:
            return index
    return limit


def short(value):
    if value is MISSING:
        return "<missing>"
    text = json.dumps(value, ensure_ascii=False, sort_keys=True)
    text = text.replace("\n", "\\n")
    return text if len(text) <= 180 else text[:177] + "..."


def category(path):
    if path.startswith("$.messages"):
        return "messages"
    if path.startswith("$.tools"):
        return "tools"
    if path.startswith("$.model"):
        return "model"
    if path.startswith("$.stream") or path.startswith("$.stream_options"):
        return "stream"
    if path.startswith("$.reasoning"):
        return "reasoning"
    return "root"


def message_hint(request, path):
    if not path.startswith("$.messages["):
        return ""
    index_text = path.split("[", 1)[1].split("]", 1)[0]
    try:
        message = request["messages"][int(index_text)]
    except (KeyError, IndexError, ValueError, TypeError):
        return ""
    role = message.get("role") if isinstance(message, dict) else None
    content = message.get("content") if isinstance(message, dict) else None
    if isinstance(content, str):
        preview = content[:120].replace("\n", "\\n")
    elif isinstance(content, list):
        preview = json.dumps(content[:1], ensure_ascii=False)[:120].replace("\n", "\\n")
    else:
        preview = ""
    return f" role={role} preview={preview!r}"


def main():
    args = parse_args()
    loaded = []
    for path in newest_logs(args.log_dir, args.limit):
        request = request_from_log(path, args.normalize_billing_header)
        if request is not None:
            loaded.append((path, request))
    if len(loaded) < 2:
        raise SystemExit("Need at least two logs with API REQUEST 1")
    for (left_path, left), (right_path, right) in zip(loaded, loaded[1:]):
        diff = first_diff(left, right)
        left_json = canonical(left)
        right_json = canonical(right)
        prefix = raw_prefix_chars(left_json, right_json)
        print(f"{left_path.name} -> {right_path.name}")
        print(f"  request_chars={len(left_json)}->{len(right_json)} common_prefix_chars={prefix}")
        if diff is None:
            print("  first_diff=<none>")
            continue
        path, lval, rval = diff
        print(f"  first_diff={path} category={category(path)}{message_hint(right, path)}")
        print(f"  left={short(lval)}")
        print(f"  right={short(rval)}")


if __name__ == "__main__":
    main()
