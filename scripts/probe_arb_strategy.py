"""Run offline behaviour probes against the current Go strategy using a temporary overlay."""

import json
import subprocess
import tempfile
from pathlib import Path


def main():
    scripts = Path(__file__).resolve().parent
    root = scripts.parent
    package = Path("application/strategy/scripts/strategy_entry/strategy/onextwo")
    virtual_test = root / package / "arb_opportunity_probe_test.go"
    fixture = scripts / "testdata/arb_strategy_probe.go"
    with tempfile.TemporaryDirectory(prefix="arb-strategy-probe-") as work:
        overlay = Path(work) / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(virtual_test): str(fixture)}}))
        return subprocess.run(
            ["go", "test", f"-overlay={overlay}", "-count=1", "-run=^TestOpportunityAuditProbes$",
             "-v", f"./{package}"],
            cwd=root,
        ).returncode


if __name__ == "__main__":
    raise SystemExit(main())
