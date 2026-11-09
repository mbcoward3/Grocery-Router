#!/usr/bin/env python3
"""Offline Helm rendering and security contract."""
from pathlib import Path
import subprocess
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[1]
CHART = ROOT / "charts/grocery-router"
DIGEST = "sha256:" + "a" * 64


def render(*values, success=True):
    command = ["helm", "template", "grocery-router", str(CHART), "--namespace", "grocery-router-prod"]
    for value in values:
        command += ["--set", value]
    result = subprocess.run(command, capture_output=True, text=True)
    if success and result.returncode:
        raise AssertionError(result.stderr)
    if not success:
        if result.returncode == 0:
            raise AssertionError("invalid configuration unexpectedly rendered")
        return []
    return [doc for doc in yaml.safe_load_all(result.stdout) if doc]


class ChartContract(unittest.TestCase):
    def test_requires_immutable_image(self):
        render(success=False)
        render("image.digest=latest", success=False)
        render("image.digest=sha256:bad", success=False)
        render(f"image.digest={DIGEST}", "replicas=0", success=False)

    def test_runtime_and_security_contract(self):
        docs = render(f"image.digest={DIGEST}")
        self.assertEqual({doc["kind"] for doc in docs}, {"Deployment", "Service"})
        deployment = next(doc for doc in docs if doc["kind"] == "Deployment")
        self.assertEqual(deployment["spec"]["strategy"], {"type": "Recreate"})
        pod = deployment["spec"]["template"]["spec"]
        self.assertFalse(pod["automountServiceAccountToken"])
        self.assertEqual(pod["securityContext"]["runAsUser"], 65532)
        container = pod["containers"][0]
        self.assertEqual(container["image"], f"ghcr.io/mbcoward3/grocery-router@{DIGEST}")
        self.assertTrue(container["securityContext"]["readOnlyRootFilesystem"])
        self.assertEqual(container["envFrom"][0]["secretRef"]["name"], "grocery-router-auth")
        database = next(item for item in container["env"] if item["name"] == "GROCERY_ROUTER_DATABASE_URL")
        self.assertEqual(database["valueFrom"]["secretKeyRef"], {"name": "grocery-router-db-app", "key": "uri"})

    def test_route(self):
        render(f"image.digest={DIGEST}", "httpRoute.enabled=true", success=False)
        docs = render(
            f"image.digest={DIGEST}",
            "httpRoute.enabled=true",
            "httpRoute.hostname=groceries.matthewcoward.com",
        )
        route = next(doc for doc in docs if doc["kind"] == "HTTPRoute")
        self.assertEqual(route["spec"]["hostnames"], ["groceries.matthewcoward.com"])
        self.assertEqual(route["spec"]["parentRefs"][0]["sectionName"], "groceries")

    def test_lint(self):
        subprocess.run(["helm", "lint", str(CHART), "--strict", "--set", f"image.digest={DIGEST}"], check=True)


if __name__ == "__main__":
    unittest.main()
