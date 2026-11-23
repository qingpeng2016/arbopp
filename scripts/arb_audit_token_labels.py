"""Cross-check native team labels of generic two-team moneylines against the two public venue archives."""
import argparse
import concurrent.futures
import json
import re
from pathlib import Path
from urllib.parse import quote
from urllib.request import Request, urlopen

# League-specific aliases for native tickers in this snapshot. Unknown names stay unverified.
TEAMS = {
    "nhl": {"CAR":"Hurricanes", "NYR":"Rangers", "PIT":"Penguins", "DAL":"Stars", "EDM":"Oilers", "STL":"Blues"},
    "nfl": {"BAL":"Ravens", "CAR":"Panthers", "NYJ":"Jets", "LAC":"Chargers", "JAX":"Jaguars", "NYG":"Giants"},
    "nba": {"DET":"Pistons", "ATL":"Hawks", "CHI":"Bulls", "DEN":"Nuggets", "DAL":"Mavericks", "UTA":"Jazz"},
}


def native(slug, platform):
    fragment = quote(slug, safe="")
    url = ("https://predict.fun/market/" if platform == "PREDICT"
           else "https://gamma-api.polymarket.com/markets/slug/") + fragment
    with urlopen(Request(url, headers={"User-Agent":"Mozilla/5.0"}), timeout=22) as response:
        text = response.read().decode()
    if platform == "POLYMARKET":
        data = json.loads(text)
        return dict(zip(json.loads(data["clobTokenIds"]), json.loads(data["outcomes"]), strict=True))
    # The public page embeds GraphQL outcome nodes in React's serialized page data.
    decoded = text.replace(r'\"','"')
    nodes = re.findall(r'\{[^{}]{0,800}"onChainId"\s*:\s*"\d+"[^{}]{0,800}\}', decoded)
    out = {}
    for node in nodes:
        try:
            data = json.loads(node)
        except ValueError:
            continue
        if data.get("name"):
            out[data["onChainId"]] = data["name"]
    return out


def run(folder):
    audit = json.loads((folder/"contracts.json").read_text())
    claims = [c for c in audit["claims"] if c["kind"] == "BINARY_YES_NO" and
              all(re.fullmatch(r".+ vs\. .+", l["outcome"] or "") and not l["outcome"].startswith("Will ")
                  for l in c["legs"]) and {l["platform"] for l in c["legs"]} == {"POLYMARKET","PREDICT"}]
    keys = {(audit["outcomes"][str(l["oid"])]["slug"], l["platform"]) for c in claims for l in c["legs"]}
    results, errors = {}, {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        jobs = {pool.submit(native, *key):key for key in keys}
        for job in concurrent.futures.as_completed(jobs):
            key = jobs[job]
            try:
                results[key] = job.result()
            except (OSError, KeyError, ValueError) as exc:
                errors[str(key)] = str(exc)
    verdicts = []
    for c in claims:
        labels = []
        for l in c["legs"]:
            slug = audit["outcomes"][str(l["oid"])]["slug"]
            raw = results.get((slug, l["platform"]), {}).get(l["token"])
            label = TEAMS.get(slug.split("-")[0], {}).get(raw, raw)
            labels.append(label)
        expected = {w.strip().casefold() for w in c["legs"][0]["outcome"].split(" vs. ")}
        observed = {s.casefold() for s in labels if s}
        status = "unverified"
        if all(labels) and observed <= expected:
            status = "complementary" if len(observed) == 2 else "same_team"
        verdicts.append(dict(sample=c["sample"],rows=c["count"],tokens=[l["token"] for l in c["legs"]],
                             labels=labels,status=status))
    output = {"claims":verdicts,"errors":errors}
    (folder/"native_labels.json").write_text(json.dumps(output,ensure_ascii=False,indent=2))
    print(json.dumps(output,ensure_ascii=False))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("folder", type=Path)
    run(parser.parse_args().folder)
