#!/usr/bin/env python3
"""Generate a JSON Lines log file of a target size for logging tests.

Each line is one JSON object resembling a structured application/access log.
A small fraction of lines are intentionally malformed (truncated JSON, bad
types, or plain garbage) so bad-line handling can be exercised.

Usage:
    python3 tools/genjson.py --out testdata/logs.jsonl --size 50MiB --seed 42
"""
import argparse
import json
import random
import sys
import time


METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"]
PATHS = [
    "/api/users", "/api/users/{id}", "/api/orders", "/api/orders/{id}",
    "/login", "/logout", "/health", "/metrics", "/api/search",
    "/api/products", "/api/products/{id}", "/static/app.js", "/static/main.css",
    "/api/cart", "/api/checkout", "/admin/dashboard", "/api/v2/reports",
]
STATUSES = [200, 200, 200, 200, 201, 204, 301, 302, 304, 400, 401, 403, 404, 429, 500, 502, 503]
LEVELS = ["DEBUG", "INFO", "INFO", "INFO", "WARN", "ERROR"]
PROTOCOLS = ["HTTP/1.1", "HTTP/2.0"]
SERVICES = ["api-gateway", "auth-svc", "order-svc", "user-svc", "search-svc", "billing-svc"]
REGIONS = ["us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"]
USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
    "curl/7.68.0", "python-requests/2.31.0", "PostmanRuntime/7.36.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
]
MESSAGES = [
    "request completed", "request failed", "cache miss", "cache hit",
    "db query slow", "rate limit exceeded", "auth token expired",
    "upstream timeout", "payload validated", "user authenticated",
]


def rand_ip(rng):
    return f"{rng.randint(1,223)}.{rng.randint(0,255)}.{rng.randint(0,255)}.{rng.randint(1,254)}"


def make_record(rng, ts):
    """Emit one Caddy-style JSON access-log line (nested), with time_format=rfc3339.

    The parser only reads ts, request.{remote_ip,proto,method,uri,headers}, status
    and size; the other fields (level, logger, msg, duration, resp_headers, ...) are
    realistic noise that Caddy actually emits and that the parser ignores.
    """
    status = rng.choice(STATUSES)
    level = "error" if status >= 500 else ("warn" if status >= 400 else rng.choice(LEVELS)).lower()
    rec = {
        "level": level,
        "ts": time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime(ts)) + f".{rng.randint(0,999):03d}Z",
        "logger": "http.log.access",
        "msg": rng.choice(MESSAGES),
        "request": {
            "remote_ip": rand_ip(rng),
            "remote_port": str(rng.randint(1024, 65535)),
            "proto": rng.choice(PROTOCOLS),
            "method": rng.choice(METHODS),
            "host": rng.choice(SERVICES) + ".example.com",
            "uri": rng.choice(PATHS),
            "headers": {
                "User-Agent": [rng.choice(USER_AGENTS)],
                "Referer": [rng.choice(["-", "https://example.com", "https://google.com/search"])],
                "Accept-Encoding": ["gzip, deflate, br"],
            },
        },
        "user_id": str(rng.randint(1, 100000)),
        "duration": round(rng.lognormvariate(3.0, 1.0) / 1000, 6),
        "size": rng.randint(0, 50000),
        "status": status,
        "resp_headers": {
            "Content-Type": ["application/json"],
            "Server": ["Caddy"],
        },
    }
    return rec


def make_bad_line(rng):
    kind = rng.randint(0, 3)
    if kind == 0:
        return "this is not json at all, just garbage text"
    if kind == 1:
        # truncated / unterminated JSON
        return '{"level":"info","ts":"2026-06-29T00:00:00.000Z","request":{"method":"GET"'
    if kind == 2:
        # invalid type (status as unquoted word)
        return '{"ts":"2026-06-29T00:00:00.000Z","status":not_a_number}'
    # empty-ish
    return ""


def parse_size(s):
    s = s.strip().lower()
    mult = 1
    for suf, m in (("kib", 1 << 10), ("mib", 1 << 20), ("gib", 1 << 30),
                   ("kb", 1000), ("mb", 1000 * 1000), ("gb", 1000 * 1000 * 1000),
                   ("k", 1 << 10), ("m", 1 << 20), ("g", 1 << 30), ("b", 1)):
        if s.endswith(suf):
            mult = m
            s = s[: -len(suf)]
            break
    return int(float(s) * mult)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--size", default="50MiB", help="target size, e.g. 50MiB, 100MB")
    ap.add_argument("--seed", type=int, default=42)
    ap.add_argument("--bad-rate", type=float, default=0.01, help="fraction of malformed lines")
    args = ap.parse_args()

    target = parse_size(args.size)
    rng = random.Random(args.seed)

    written = 0
    lines = 0
    bad = 0
    ts = int(time.mktime(time.strptime("2026-06-01", "%Y-%m-%d")))

    buf = []
    bufbytes = 0
    with open(args.out, "w", encoding="utf-8") as f:
        while written < target:
            ts += rng.randint(0, 3)
            if rng.random() < args.bad_rate:
                line = make_bad_line(rng)
                bad += 1
            else:
                line = json.dumps(make_record(rng, ts), separators=(",", ":"))
            buf.append(line)
            n = len(line) + 1
            bufbytes += n
            written += n
            lines += 1
            if bufbytes >= (1 << 20):
                f.write("\n".join(buf) + "\n")
                buf.clear()
                bufbytes = 0
        if buf:
            f.write("\n".join(buf) + "\n")

    print(f"wrote {args.out}: {written} bytes, {lines} lines ({bad} malformed)", file=sys.stderr)


if __name__ == "__main__":
    main()
