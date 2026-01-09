#!/usr/bin/env python3

import csv
import sys
from pathlib import Path


EXCLUDED_BENCHES = {
    "FSSZ-INT-01",
    "FSSZ-222",
    "FSSZ-INT-02",
    "FSSZ-INT-03",
}

CROSS_CLIENT_META = {
    "BV-DirtyPadding": {
        "suite": "benchmark",
        "target": "Canonicalization",
        "ssz_schema": "BeaconState.JustificationBits (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "reference": "docs/bugs/Bitvector Dirty Padding Vulnerability 2b6f5aa90cdd80b68c19c26fb294f9eb.md",
    },
    "Bool-Dirty": {
        "suite": "benchmark",
        "target": "Canonicalization",
        "ssz_schema": "Validator.Slashed (boolean)",
        "bug_class": "dirty boolean encoding",
        "reference": "docs/bugs/Dirty Boolean Vulnerability 2b7f5aa90cdd80f5b3bff7a42a4b1fc6.md",
    },
    "Bitlist-Null": {
        "suite": "benchmark",
        "target": "Decoding",
        "ssz_schema": "PendingAttestation.AggregationBits (Bitlist[2048])",
        "bug_class": "null-bitlist missing sentinel",
        "reference": "docs/bugs/Null-Bitlist Trap Vulnerability 2b8f5aa90cdd80efa1b1f17cbd81cfec.md",
    },
    "Union-DirtyTail": {
        "suite": "benchmark",
        "target": "Decoding",
        "ssz_schema": "SuffixStateDiff.new_value (Optional[Bytes32])",
        "bug_class": "union selector trailing data",
        "reference": "docs/bugs/Union Selector Trailing Data Vulnerability 2b7f5aa90cdd801193a4d61ecc297371.md",
    },
    "Gap-Offsets": {
        "suite": "benchmark",
        "target": "Decoding",
        "ssz_schema": "PendingAttestation variable-field offsets",
        "bug_class": "variable-length container gap",
        "reference": "docs/bugs/Variable-Length Container Gap Vulnerability 2b8f5aa90cdd80debc88d63023dc9f02.md",
    },
}

FUZZ_VARIANT_META = {
    "BV-Base": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BitvectorStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/schemas.go",
    },
    "BV-Hard": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.HardBitvectorStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/hard_schemas.go",
    },
    "BV-Pair": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BitvectorPairStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "BV-Wide": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BitvectorWideStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "BV-Offset": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BitvectorOffsetStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "BV-Scatter": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BitvectorScatterStruct (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "BV-BeaconState": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BeaconState.JustificationBits (Bitvector[4])",
        "bug_class": "dirty bitvector padding",
        "patch": "patches/Bitvector_Dirty_Padding.patch",
        "reference": "schemas/schemas.go",
    },
    "Bool-Base": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BooleanStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/schemas.go",
    },
    "Bool-Hard": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.HardBooleanStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/hard_schemas.go",
    },
    "Bool-Pair": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BooleanPairStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bool-Wide": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BooleanWideStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bool-Offset": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BooleanOffsetStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bool-Scatter": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.BooleanScatterStruct (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bool-Validator": {
        "suite": "fuzz-variant",
        "target": "Canonicalization",
        "ssz_schema": "schemas.Validator.Slashed (boolean)",
        "bug_class": "dirty boolean encoding",
        "patch": "patches/Dirty_Boolean.patch",
        "reference": "schemas/schemas.go",
    },
    "Bitlist-Base": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.AggregationBitsContainer.AggregationBits (Bitlist[2048])",
        "bug_class": "null-bitlist missing sentinel",
        "patch": "patches/Null_Bitlist.patch",
        "reference": "schemas/bitlist.go",
    },
    "Bitlist-Pair": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.BitlistPairStruct (Bitlist[2048])",
        "bug_class": "null-bitlist missing sentinel",
        "patch": "patches/Null_Bitlist.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bitlist-Wide": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.BitlistWideStruct (Bitlist[2048])",
        "bug_class": "null-bitlist missing sentinel",
        "patch": "patches/Null_Bitlist.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bitlist-Tri": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.BitlistTriStruct (Bitlist[2048])",
        "bug_class": "null-bitlist missing sentinel",
        "patch": "patches/Null_Bitlist.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Bitlist-Offset": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.BitlistOffsetStruct (Bitlist[1024])",
        "bug_class": "null-bitlist missing sentinel",
        "patch": "patches/Null_Bitlist.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Union-Base": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.UnionStruct (Optional[Bytes32])",
        "bug_class": "union selector trailing data",
        "reference": "schemas/union.go",
    },
    "Union-Hard": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.HardUnionStruct (Optional[Bytes32])",
        "bug_class": "union selector trailing data",
        "reference": "schemas/union.go",
    },
    "Union-Wide": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.UnionWideStruct (Optional[Bytes32])",
        "bug_class": "union selector trailing data",
        "reference": "schemas/union.go",
    },
    "Union-Scatter": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.UnionScatterStruct (Optional[Bytes32])",
        "bug_class": "union selector trailing data",
        "reference": "schemas/union.go",
    },
    "Gap-Base": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.GapStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/schemas.go",
    },
    "Gap-Hard": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.HardGapStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/hard_schemas.go",
    },
    "Gap-Pair": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.GapPairStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Gap-Wide": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.GapWideStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Gap-Tri": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.GapTriStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/ablation_variants.go",
    },
    "Gap-Scatter": {
        "suite": "fuzz-variant",
        "target": "Decoding",
        "ssz_schema": "schemas.GapScatterStruct (variable-length container)",
        "bug_class": "variable-length container gap",
        "patch": "patches/Container_Gap.patch",
        "reference": "schemas/ablation_variants.go",
    },
}


def load_ablation_results(path: Path) -> tuple[list[str], dict[str, dict]]:
    order: list[str] = []
    results: dict[str, dict] = {}
    with path.open(newline="") as f:
        reader = csv.DictReader(f)
        for row in reader:
            bench = (row.get("benchmark") or "").strip()
            if not bench:
                continue
            if bench not in results:
                results[bench] = {
                    "category": (row.get("category") or "").strip(),
                    "bench_schema": (row.get("schema") or "").strip(),
                    "modes": {},
                }
                order.append(bench)
            if not results[bench]["category"] and row.get("category"):
                results[bench]["category"] = (row.get("category") or "").strip()
            if not results[bench]["bench_schema"] and row.get("schema"):
                results[bench]["bench_schema"] = (row.get("schema") or "").strip()
            mode_label = (row.get("mode_label") or "").strip()
            if mode_label:
                results[bench]["modes"][mode_label] = {
                    "result": (row.get("result") or "").strip(),
                    "tte_ms": (row.get("duration_ms") or "").strip(),
                    "coverage": (row.get("coverage") or "").strip(),
                    "step": (row.get("step") or "").strip(),
                }
    return order, results


def component_from_bug_class(bug_class: str) -> str:
    lower = bug_class.lower()
    if "sszgen" in lower or "codegen" in lower or "generator" in lower:
        return "Codegen"
    if "merkle" in lower:
        return "Merkleization"
    if "htr" in lower:
        return "HTR"
    if "proof" in lower:
        return "Proofs"
    if "decode" in lower or "offset" in lower or "bitlist" in lower:
        return "Decoding"
    if "size" in lower or "encode" in lower:
        return "Encoding"
    return "Runtime"


def component_from_section(section: str, bug_class: str) -> str:
    lower = section.lower()
    bug_lower = bug_class.lower()
    if "sszgen" in lower or "code generation" in lower:
        return "Codegen"
    if "canonicalization" in lower:
        if "dirty" in bug_lower or "canonical" in bug_lower:
            return "Canonicalization"
        return "Decoding"
    if "merkleization" in lower or "proof" in lower or "htr" in lower:
        if "htr" in bug_lower:
            return "HTR"
        if "proof" in bug_lower:
            return "Proofs"
        return "Merkleization"
    return component_from_bug_class(bug_class)


def parse_fastssz_metadata(path: Path) -> dict[str, dict]:
    meta: dict[str, dict] = {}
    if not path.exists():
        return meta
    section = ""
    with path.open() as f:
        for line in f:
            if line.startswith("##"):
                section = line.strip()
                continue
            if not line.lstrip().startswith("|"):
                continue
            parts = [p.strip() for p in line.strip().strip("|").split("|")]
            if len(parts) < 4:
                continue
            if parts[0].lower().startswith("benchmark id"):
                continue
            if set(parts[0]) <= {"-"}:
                continue
            bench = parts[0]
            if not bench.startswith("FSSZ-"):
                continue
            bug_class = parts[1]
            patch = parts[2].strip("`")
            reference = parts[3].strip("`")
            meta[bench] = {
                "suite": "benchmark",
                "target": component_from_section(section, bug_class),
                "ssz_schema": "",
                "bug_class": bug_class,
                "patch": patch,
                "reference": reference,
            }
    return meta


def main() -> int:
    repo_root = Path(__file__).resolve().parents[1]
    ablation_path = repo_root / "ablation_results.csv"
    fastssz_meta_path = repo_root / "docs" / "bugs" / "fastssz-regressions.md"
    out_path = repo_root / "alma_ccs26_new" / "data" / "benchmark_results.csv"

    if not ablation_path.exists():
        print(f"missing ablation results: {ablation_path}", file=sys.stderr)
        return 1

    order, ablation = load_ablation_results(ablation_path)
    fastssz_meta = parse_fastssz_metadata(fastssz_meta_path)

    out_path.parent.mkdir(parents=True, exist_ok=True)

    fieldnames = [
        "benchmark_id",
        "suite",
        "category",
        "bench_schema",
        "ssz_schema",
        "target",
        "bug_class",
        "patch",
        "reference",
        "data_source",
        "regression_stage",
        "full_result",
        "full_tte_ms",
        "full_coverage",
        "full_step",
        "norl_result",
        "norl_tte_ms",
        "norl_coverage",
        "norl_step",
        "nospec_result",
        "nospec_tte_ms",
        "nospec_coverage",
        "nospec_step",
    ]

    missing_meta = []

    with out_path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for bench in order:
            if bench in EXCLUDED_BENCHES:
                continue
            bench_data = ablation.get(bench, {})
            category = bench_data.get("category", "")
            bench_schema = bench_data.get("bench_schema", "")
            modes = bench_data.get("modes", {})
            meta = CROSS_CLIENT_META.get(bench)
            if meta is None:
                meta = FUZZ_VARIANT_META.get(bench)
            if meta is None:
                meta = fastssz_meta.get(bench)
            if meta is None:
                meta = {}
                missing_meta.append(bench)
            row = {
                "benchmark_id": bench,
                "suite": meta.get("suite") or "n/a",
                "category": category or "n/a",
                "bench_schema": bench_schema or "n/a",
                "ssz_schema": meta.get("ssz_schema") or "n/a",
                "target": meta.get("target") or "n/a",
                "bug_class": meta.get("bug_class") or "n/a",
                "patch": meta.get("patch") or "n/a",
                "reference": meta.get("reference") or "n/a",
                "regression_stage": "n/a",
            }

            row["data_source"] = "ablation"
            for label in ["full", "norl", "nospec"]:
                m = modes.get(label, {})
                row[f"{label}_result"] = m.get("result", "n/a")
                row[f"{label}_tte_ms"] = m.get("tte_ms", "n/a")
                row[f"{label}_coverage"] = m.get("coverage", "n/a")
                row[f"{label}_step"] = m.get("step", "n/a")

            writer.writerow(row)

    if missing_meta:
        print(f"warning: missing metadata for {len(missing_meta)} benchmarks:", file=sys.stderr)
        for bench in missing_meta:
            print(f"  - {bench}", file=sys.stderr)

    print(f"Wrote unified benchmark results to {out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
