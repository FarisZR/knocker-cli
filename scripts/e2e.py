#!/usr/bin/env python3
"""Run the Go service integration test against a disposable local Knocker server."""
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request

server_source = Path(sys.argv[1] if len(sys.argv) > 1 else "../knocker").resolve()
repo = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="knocker-e2e-", dir="/tmp") as directory:
    workspace = Path(directory)
    key = secrets.token_hex(32)
    config = workspace / "knocker.json"
    config.write_text(json.dumps({
        "server": {"trusted_proxies": ["127.0.0.1/32"]},
        "whitelist": {"storage_path": str(workspace / "whitelist.json")},
        "firewalld": {"enabled": False},
        "api_keys": [{"key": key, "allow_remote_whitelist": True, "max_ttl": 60}],
    }))
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
    url = f"http://127.0.0.1:{port}"
    env = {**os.environ, "KNOCKER_CONFIG_PATH": str(config)}
    with (workspace / "server.log").open("w+") as output:
        server = subprocess.Popen([
            "uv", "run", "uvicorn", "src.main:app", "--host", "127.0.0.1",
            "--port", str(port), "--no-proxy-headers",
        ], cwd=server_source, env=env, stdout=output, stderr=output)
        try:
            for _ in range(600):
                if server.poll() is not None:
                    raise RuntimeError("Knocker exited before becoming ready")
                try:
                    with urllib.request.urlopen(url + "/health", timeout=1):
                        break
                except OSError:
                    time.sleep(0.1)
            else:
                raise RuntimeError("Knocker did not become ready within 60 seconds")
            subprocess.run([
                "go", "test", "-race", "./internal/service", "-run", "^TestServerEndToEnd$", "-v", "-count=1",
            ], cwd=repo, env={**os.environ, "KNOCKER_E2E_URL": url, "KNOCKER_E2E_KEY": key}, check=True)
        except BaseException:
            output.seek(0)
            print(output.read(), file=sys.stderr)
            raise
        finally:
            server.terminate()
            try:
                server.wait(timeout=5)
            except subprocess.TimeoutExpired:
                server.kill()
                server.wait()
