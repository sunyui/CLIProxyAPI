#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


def parse_args():
    parser = argparse.ArgumentParser(description="Analyze CLIProxyAPI v1/messages cache logs.")
    parser.add_argument(
        "--log-dir",
        default=str(Path.home() / ".cli-proxy-api" / "auths" / "logs"),
        help="Directory containing CLIProxyAPI request logs.",
    )
    parser.add_argument("--limit", type=int, default=20, help="Number of newest message logs to analyze.")
    return parser.parse_args()


def section(text, name):
    marker = f"=== {name} ==="
    start = text.find(marker)
    if start < 0:
        return ""
    start += len(marker)
    end = text.find("\n=== ", start)
    return text[start : end if end >= 0 else len(text)].strip()


def json_from_block(block):
    block = block.strip()
    if not block:
        return None
    starts = [idx for idx in (block.find("{"), block.find("[")) if idx >= 0]
    if starts:
        block = block[min(starts) :]
    try:
        return json.loads(block)
    except json.JSONDecodeError:
        return None


def cache_paths(value, prefix=""):
    paths = []
    if isinstance(value, dict):
        if "cache_control" in value:
            paths.append(prefix or "$")
        for key, child in value.items():
            child_prefix = f"{prefix}.{key}" if prefix else key
            paths.extend(cache_paths(child, child_prefix))
    elif isinstance(value, list):
        for index, child in enumerate(value):
            paths.extend(cache_paths(child, f"{prefix}[{index}]"))
    return paths


def sse_usages(block):
    usages = []
    obj = json_from_block(block)
    if isinstance(obj, dict) and obj.get("usage"):
        usages.append(obj["usage"])

    for line in block.splitlines():
        line = line.strip()
        if not line.startswith("data:"):
            continue
        payload = line[5:].strip()
        if payload == "[DONE]":
            continue
        try:
            event = json.loads(payload)
        except json.JSONDecodeError:
            continue
        if isinstance(event, dict) and event.get("usage"):
            usages.append(event["usage"])
    return usages


def normalized_usage(usage):
    details = usage.get("prompt_tokens_details") or {}
    cached = details.get("cached_tokens")
    if cached is None:
        cached = usage.get("cache_read_input_tokens")
    return {
        "prompt": usage.get("prompt_tokens") or usage.get("input_tokens"),
        "cached": cached,
        "created": usage.get("cache_creation_input_tokens"),
        "completion": usage.get("completion_tokens") or usage.get("output_tokens"),
        "total": usage.get("total_tokens"),
    }


def newest_logs(log_dir, limit):
    files = [p for p in Path(log_dir).glob("v1-messages-*.log") if "count_tokens" not in p.name]
    return sorted(files, key=lambda p: p.stat().st_mtime)[-limit:]


def analyze(path):
    text = path.read_text(errors="replace")
    request = json_from_block(section(text, "API REQUEST 1")) or json_from_block(section(text, "REQUEST BODY"))
    paths = cache_paths(request) if request is not None else []
    tool_cache_messages = []
    if isinstance(request, dict):
        for index, message in enumerate(request.get("messages") or []):
            if isinstance(message, dict) and message.get("role") == "tool" and "cache_control" in message:
                tool_cache_messages.append(index)

    usages = []
    for block_name in ("API RESPONSE 1", "RESPONSE"):
        for usage in sse_usages(section(text, block_name)):
            normalized = normalized_usage(usage)
            if normalized not in usages:
                usages.append(normalized)

    return {
        "file": path.name,
        "cache_paths": len(paths),
        "cache_path_locations": paths,
        "tool_cache_messages": tool_cache_messages,
        "usage": usages[-2:],
    }


def main():
    args = parse_args()
    for path in newest_logs(args.log_dir, args.limit):
        result = analyze(path)
        print(
            f"{result['file']} cache_paths={result['cache_paths']} "
            f"tool_cache_messages={result['tool_cache_messages']} usage={result['usage']}"
        )


if __name__ == "__main__":
    main()
