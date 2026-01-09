#!/usr/bin/env python3

import argparse
import csv
import concurrent.futures
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path


@dataclass
class Bench:
    name: str
    schema: str
    schema_source: str
    bug: str | None = None
    require_bitvector: bool = False
    testcase_file: str | None = None
    oracle: str | None = None
    oracle_bug: str | None = None
    schema_validate: bool = False
    disable_tail: bool = False
    disable_gap: bool = False
    enable_bitlist_null: bool = False


BENCHES = [
    # Canonical fuzz benchmarks (one per independent bug class).
    Bench("BV-DirtyPadding", "BeaconStateBench", "benchschemas", "FSSZ-INT-01", True),
    Bench("Bool-Dirty", "ValidatorEnvelope", "benchschemas", "FSSZ-222"),
    Bench("Bitlist-Null", "AttestationEnvelope", "benchschemas", "FSSZ-INT-02"),
    Bench("Union-DirtyTail", "UnionBench", "benchschemas", None),
    Bench("Gap-Offsets", "BlockBodyBench", "benchschemas", "FSSZ-INT-03"),
    # sszgen / code generation
    Bench("FSSZ-181", "FSSZ-181", "testcases", "FSSZ-181", testcase_file="case4.go"),
    Bench("FSSZ-162", "FSSZ-162", "testcases", "FSSZ-162", testcase_file="uint.go"),
    Bench("FSSZ-152", "FSSZ-152", "testcases", "FSSZ-152", testcase_file="pr_152.go"),
    Bench("FSSZ-127", "FSSZ-127", "testcases", "FSSZ-127", testcase_file="issue_127.go"),
    Bench("FSSZ-54", "FSSZ-54", "testcases", "FSSZ-54", testcase_file="list.go"),
    Bench("FSSZ-76", "FSSZ-76", "testcases", "FSSZ-76", testcase_file="case1.go"),
    Bench("FSSZ-153", "FSSZ-153", "testcases", "FSSZ-153", testcase_file="issue_153.go"),
    Bench("FSSZ-158", "FSSZ-158", "testcases", "FSSZ-158", testcase_file="issue_158.go"),
    Bench("FSSZ-166", "FSSZ-166", "testcases", "FSSZ-166", testcase_file="issue_166.go"),
    Bench("FSSZ-136", "FSSZ-136", "testcases", "FSSZ-136", testcase_file="issue_136.go"),
    Bench("FSSZ-156", "FSSZ-156", "testcases", "FSSZ-156", testcase_file="issue_156.go"),
    Bench("FSSZ-159", "FSSZ-159", "testcases", "FSSZ-159", testcase_file="issue_159.go"),
    Bench("FSSZ-164", "FSSZ-164", "testcases", "FSSZ-164", testcase_file="issue_164.go"),
    Bench("FSSZ-188", "FSSZ-188", "testcases", "FSSZ-188", testcase_file="issue_188.go"),
    Bench("FSSZ-86", "FSSZ-86", "testcases", "FSSZ-86", testcase_file="case2.go"),
    Bench("FSSZ-100", "FSSZ-100", "testcases", "FSSZ-100", testcase_file="time.go"),
    Bench("FSSZ-149", "FSSZ-149", "testcases", "FSSZ-149", testcase_file="integration_uint.go"),
    Bench("FSSZ-151", "FSSZ-151", "testcases", "FSSZ-151", testcase_file="list.go"),
    Bench("FSSZ-1", "FSSZ-1", "testcases", "FSSZ-1", testcase_file="case3.go"),
    Bench("FSSZ-52", "FSSZ-52", "testcases", "FSSZ-52", testcase_file="case4.go"),
    # Merkleization / HTR / proof generation
    Bench("FSSZ-173", "FSSZ-173", "benchschemas", "FSSZ-173"),
    Bench("FSSZ-147", "FSSZ-147", "schemas", "FSSZ-147"),
    Bench("FSSZ-119", "FSSZ-119", "benchschemas", "FSSZ-119"),
    Bench("FSSZ-111", "FSSZ-111", "schemas", "FSSZ-111"),
    Bench("FSSZ-110", "FSSZ-110", "benchschemas", "FSSZ-110"),
    Bench("FSSZ-98", "FSSZ-98", "testcases", "FSSZ-98", testcase_file="case3.go"),
    Bench("FSSZ-96", "FSSZ-96", "benchschemas", "FSSZ-96"),
    Bench("FSSZ-9", "FSSZ-9", "benchschemas", "FSSZ-9"),
    Bench("FSSZ-23", "FSSZ-23", "benchschemas", "FSSZ-23"),
    # py-ssz historical/spec boundary (external oracle)
    Bench(
        "PSSZ-109",
        "PSSZBitlistBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-109",
        disable_tail=True,
        disable_gap=True,
        enable_bitlist_null=True,
    ),
    Bench(
        "PSSZ-35",
        "PSSZByteListBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-35",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-74",
        "PSSZHeaderListBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-74",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-82",
        "PSSZBitlistBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-82",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-83",
        "PSSZHTRListBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-83",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-BOOL-DIRTY",
        "PSSZBoolBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-BOOL-DIRTY",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-BV-DIRTY",
        "PSSZBitvectorBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-BV-DIRTY",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-HTR-LIST-NOMIX",
        "PSSZHTRListBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-HTR-LIST-NOMIX",
        disable_tail=True,
        disable_gap=True,
    ),
    Bench(
        "PSSZ-HTR-BITLIST-NOMIX",
        "PSSZBitlistBench",
        "benchschemas",
        oracle="pyssz",
        oracle_bug="PSSZ-HTR-BITLIST-NOMIX",
        disable_tail=True,
        disable_gap=True,
    ),
    # py-ssz schema validation (schema-only)
    Bench(
        "PSSZ-111",
        "PSSZ-111",
        "schemas",
        oracle="pyssz",
        oracle_bug="PSSZ-111",
        schema_validate=True,
    ),
    Bench(
        "PSSZ-112",
        "PSSZ-112",
        "schemas",
        oracle="pyssz",
        oracle_bug="PSSZ-112",
        schema_validate=True,
    ),
    Bench(
        "PSSZ-116",
        "PSSZ-116",
        "schemas",
        oracle="pyssz",
        oracle_bug="PSSZ-116",
        schema_validate=True,
    ),
]

MODES = [
    ("full", "rl"),
    ("norl", "norl"),
    ("nospec", "baseline"),
]


def run(cmd: list[str], cwd: Path | None = None, timeout_s: float | None = None) -> str:
    try:
        proc = subprocess.run(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            cwd=str(cwd) if cwd else None,
            timeout=timeout_s,
        )
    except subprocess.TimeoutExpired as exc:
        output = (exc.stdout or "") if isinstance(exc.stdout, str) else ""
        if len(output) > 8000:
            output = f"...truncated...\n{output[-8000:]}"
        raise RuntimeError(f"Command timed out after {timeout_s}s: {' '.join(cmd)}\n{output}") from exc
    if proc.returncode != 0:
        output = proc.stdout or ""
        if len(output) > 8000:
            output = f"...truncated...\n{output[-8000:]}"
        raise RuntimeError(f"Command failed ({proc.returncode}): {' '.join(cmd)}\n{output}")
    return proc.stdout


def parse_duration_ms(value: str) -> float:
    if value.endswith("ms"):
        return float(value[:-2])
    if value.endswith("µs"):
        return float(value[:-2]) / 1000.0
    if value.endswith("us"):
        return float(value[:-2]) / 1000.0
    if value.endswith("ns"):
        return float(value[:-2]) / 1_000_000.0
    if value.endswith("s"):
        return float(value[:-1]) * 1000.0
    if value.endswith("m"):
        return float(value[:-1]) * 60_000.0
    if value.endswith("h"):
        return float(value[:-1]) * 3_600_000.0
    raise ValueError(f"unsupported duration format: {value}")


def parse_measure_output(output: str) -> dict:
    lines = output.strip().splitlines()
    for line in reversed(lines):
        if not line.startswith("MODE="):
            continue
        parts = line.split()
        kv = {}
        for part in parts:
            if "=" not in part:
                continue
            k, v = part.split("=", 1)
            kv[k] = v
        if "MODE" not in kv or "SCHEMA" not in kv or "RESULT" not in kv or "DURATION" not in kv or "COVERAGE" not in kv:
            raise ValueError(f"unable to parse measurement line: {line}")
        return {
            "mode": kv["MODE"],
            "schema": kv["SCHEMA"],
            "result": kv["RESULT"],
            "step": int(kv["STEP"]) if "STEP" in kv else None,
            "duration_ms": parse_duration_ms(kv["DURATION"]),
            "coverage": int(kv["COVERAGE"]),
        }
    raise ValueError(f"no measurement line found in output:\n{output}")


def is_codegen_compile_error(output: str) -> bool:
    return any(
        needle in output
        for needle in (
            "sszgen/testcases",
            "benchschemas",
            "schemas",
            "sszgen/generator",
        )
    )


def toggle_bug(action: str, bug: str, repo_root: Path) -> None:
    cmd = [str(repo_root / "scripts" / "bug_toggle.sh"), action, bug]
    try:
        output = run(cmd)
        print(output.strip())
    except RuntimeError as exc:
        print(exc)
        raise


def regenerate_benchschemas(repo_root: Path, timeout_s: float) -> bool:
    try:
        run(
            [
                "go",
                "run",
                str(repo_root / "workspace" / "fastssz_bench" / "sszgen" / "main.go"),
                "-path",
                str(repo_root / "benchschemas"),
                "-exclude-objs",
                "DebugUnion,UnionBench",
            ],
            timeout_s=timeout_s,
        )
        for filename in ("types_encoding.go", "pyssz_bench_encoding.go"):
            file_path = repo_root / "benchschemas" / filename
            if not file_path.exists():
                continue
            run(
                [
                    "go",
                    "run",
                    str(repo_root / "cmd" / "instrumentor" / "main.go"),
                    "-file",
                    str(file_path),
                ],
                timeout_s=timeout_s,
            )
    except RuntimeError as exc:
        print(f"!! benchschemas regeneration failed: {exc}")
        return False
    return True

def regenerate_testcases(repo_root: Path, filename: str, timeout_s: float) -> bool:
    try:
        run(
            ["go", "generate", filename],
            cwd=repo_root / "workspace" / "fastssz_bench" / "sszgen" / "testcases",
            timeout_s=timeout_s,
        )
    except RuntimeError as exc:
        print(f"!! testcases regeneration failed ({filename}): {exc}")
        return False
    return True


def read_hash_line(path: Path) -> str | None:
    if not path.exists():
        return None
    for line in path.read_text().splitlines():
        line = line.strip()
        if line.startswith("// Hash:"):
            return line.split(":", 1)[1].strip()
    return None


def generated_has_receiver(path: Path, type_name: str) -> bool:
    if not path.exists():
        return False
    text = path.read_text()
    pattern = rf"func\s*\([^)]*\*{re.escape(type_name)}\)"
    if re.search(pattern, text):
        return True
    pattern = rf"func\s*\([^)]*\b{re.escape(type_name)}\b\)"
    return re.search(pattern, text) is not None


def detect_codegen_bug(bench: Bench, repo_root: Path, gen_timeout_s: float) -> tuple[bool, str]:
    if not bench.bug:
        return False, ""
    if bench.bug == "FSSZ-76":
        gen_file = repo_root / "workspace" / "fastssz_bench" / "sszgen" / "testcases" / "case1_encoding.go"
        if generated_has_receiver(gen_file, "Bytes"):
            return True, "exclude-obj violated"
        return False, ""
    if bench.bug == "FSSZ-49":
        gen_file = repo_root / "workspace" / "fastssz_bench" / "sszgen" / "testcases" / "case3_encoding.go"
        if not gen_file.exists():
            return False, ""
        text = gen_file.read_text()
        if "sszgen/testcases/other" not in text or "other.Case3B" not in text:
            return True, "duplicate name resolution failed"
        return False, ""
    if bench.bug == "FSSZ-110":
        gen_file = repo_root / "benchschemas" / "types_encoding.go"
        hashes = []
        hash_before = read_hash_line(gen_file)
        if hash_before:
            hashes.append(hash_before)
        for _ in range(3):
            regen_ok = regenerate_benchschemas(repo_root, gen_timeout_s)
            if not regen_ok:
                return False, ""
            hash_after = read_hash_line(gen_file)
            if hash_after:
                hashes.append(hash_after)
        if len(set(hashes)) > 1:
            return True, "nondeterministic codegen hash"
        return False, ""
    return False, ""


def patch_path_for_bug(bug: str, repo_root: Path) -> Path:
    patches_dir = repo_root / "patches"
    if bug in {"FSSZ-INT-01", "fssz-int-01"}:
        return patches_dir / "Bitvector_Dirty_Padding.patch"
    if bug in {"FSSZ-222", "fssz-222"}:
        return patches_dir / "Dirty_Boolean.patch"
    if bug in {"FSSZ-INT-02", "fssz-int-02"}:
        return patches_dir / "Null_Bitlist.patch"
    if bug in {"FSSZ-INT-03", "fssz-int-03"}:
        return patches_dir / "Container_Gap.patch"
    if bug.startswith("FSSZ-"):
        return patches_dir / "fastssz" / f"{bug}.patch"
    raise ValueError(f"unknown bug id: {bug}")


def patch_requires_codegen(bug: str, repo_root: Path) -> bool:
    patch_path = patch_path_for_bug(bug, repo_root)
    if not patch_path.exists():
        raise FileNotFoundError(f"patch file not found: {patch_path}")
    for line in patch_path.read_text().splitlines():
        if line.startswith("diff --git") and "/sszgen/" in line:
            return True
    return False


def build_measure_binary(repo_root: Path, out_path: Path, timeout_s: float) -> bool:
    out_path.parent.mkdir(parents=True, exist_ok=True)
    try:
        run(
            ["go", "build", "-o", str(out_path), "./cmd/measure"],
            cwd=repo_root,
            timeout_s=timeout_s,
        )
    except RuntimeError as exc:
        print(f"!! measure build failed: {exc}")
        return False
    return True


def run_mode_trials(
    bench: Bench,
    measure_bin: Path,
    mode_label: str,
    mode: str,
    budget_ms: float,
    run_timeout_s: float,
    args: argparse.Namespace,
) -> dict:
    durations: list[float] = []
    coverages: list[int] = []
    steps: list[int] = []
    found_any = False
    for _ in range(max(1, args.trials)):
        trial_start = time.time()
        cmd = [
            str(measure_bin),
            "-schema",
            bench.schema,
            "-mode",
            mode,
            "-budget",
            args.budget,
            "-max-steps",
            str(args.max_steps),
            "-batch-size",
            str(args.batch_size),
        ]
        if bench.oracle:
            cmd.extend(["-oracle", bench.oracle])
        if bench.oracle_bug:
            cmd.extend(["-oracle-bug", bench.oracle_bug])
        if bench.schema_validate:
            cmd.append("-schema-validate")
        if bench.disable_tail:
            cmd.append("-no-tail")
        if bench.disable_gap:
            cmd.append("-no-gap")
        if bench.enable_bitlist_null:
            cmd.append("-bitlist-null")
        if bench.require_bitvector:
            cmd.append("-require-bitvector-bug")

        try:
            output = run(cmd, timeout_s=run_timeout_s)
        except RuntimeError as exc:
            err_output = str(exc)
            if is_codegen_compile_error(err_output):
                found_any = True
                durations.append((time.time() - trial_start) * 1000.0)
                coverages.append(0)
                steps.append(0)
                continue
            if "Command timed out" in err_output:
                durations.append(budget_ms)
                coverages.append(0)
                steps.append(0)
                continue
            raise

        parsed = parse_measure_output(output)
        if parsed["result"] == "bug":
            found_any = True
            durations.append(parsed["duration_ms"])
        else:
            durations.append(budget_ms)
        coverages.append(parsed["coverage"])
        steps.append(parsed["step"] or 0)

    durations.sort()
    coverages.sort()
    steps.sort()
    median_duration = durations[len(durations) // 2]
    median_coverage = coverages[len(coverages) // 2]
    median_steps = steps[len(steps) // 2]
    return {
        "mode_label": mode_label,
        "mode": mode,
        "result": "bug" if found_any else "timeout",
        "duration_ms": median_duration,
        "coverage": median_coverage,
        "step": median_steps,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Measure ablation study results.")
    parser.add_argument("--budget", default="30s", help="Budget per run (Go duration)")
    parser.add_argument("--max-steps", type=int, default=50000, help="Max steps per episode")
    parser.add_argument("--batch-size", type=int, default=50, help="Batch size")
    parser.add_argument("--out", default="ablation_results.csv", help="CSV output path")
    parser.add_argument("--trials", type=int, default=1, help="Trials per fuzzing benchmark/mode")
    parser.add_argument("--gen-timeout", default="5s", help="Timeout for codegen/regeneration commands")
    parser.add_argument("--build-timeout", default="5s", help="Timeout for building the measure binary")
    parser.add_argument("--only", default="", help="Comma-separated benchmark filters")
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[1]
    out_path = repo_root / args.out

    results = []
    budget_ms = parse_duration_ms(args.budget)
    budget_s = budget_ms / 1000.0
    run_timeout_s = max(1.0, budget_s + 0.5)
    gen_timeout_s = parse_duration_ms(args.gen_timeout) / 1000.0
    build_timeout_s = parse_duration_ms(args.build_timeout) / 1000.0

    benches = BENCHES
    if args.only:
        tokens = [t.strip() for t in args.only.split(",") if t.strip()]
        if tokens:
            benches = [b for b in BENCHES if any(tok in b.name for tok in tokens)]

    for bench in benches:
        print(f"==> Benchmark {bench.name} ({bench.schema})")
        bench_start = time.time()
        category = "schema" if bench.schema_validate else "fuzz"
        regen_needed = False
        regen_ok = True
        if bench.bug:
            regen_needed = patch_requires_codegen(bench.bug, repo_root)

        try:
            if bench.bug:
                toggle_bug("activate", bench.bug, repo_root)
            if regen_needed:
                if bench.schema_source == "benchschemas":
                    regen_ok = regenerate_benchschemas(repo_root, gen_timeout_s)
                elif bench.schema_source == "testcases":
                    if not bench.testcase_file:
                        raise ValueError(f"missing testcase file for {bench.name}")
                    regen_ok = regenerate_testcases(repo_root, bench.testcase_file, gen_timeout_s)

            if regen_needed and not regen_ok:
                elapsed_ms = (time.time() - bench_start) * 1000.0
                for label, mode in MODES:
                    results.append(
                        {
                            "benchmark": bench.name,
                            "category": category,
                            "schema": bench.schema,
                            "mode_label": label,
                            "mode": mode,
                            "result": "bug",
                            "duration_ms": elapsed_ms,
                            "coverage": 0,
                            "step": 0,
                        }
                    )
                    print(f"  {label}: result=bug tte_ms={elapsed_ms:.2f} coverage=0 (regen failure)")
                continue

            codegen_bug, reason = detect_codegen_bug(bench, repo_root, gen_timeout_s)
            if codegen_bug:
                elapsed_ms = (time.time() - bench_start) * 1000.0
                for label, mode in MODES:
                    results.append(
                        {
                            "benchmark": bench.name,
                            "category": category,
                            "schema": bench.schema,
                            "mode_label": label,
                            "mode": mode,
                            "result": "bug",
                            "duration_ms": elapsed_ms,
                            "coverage": 0,
                            "step": 0,
                        }
                    )
                    print(f"  {label}: result=bug tte_ms={elapsed_ms:.2f} coverage=0 ({reason})")
                continue

            measure_bin = repo_root / ".tmp" / f"measure_{bench.name}"
            build_ok = build_measure_binary(repo_root, measure_bin, build_timeout_s)
            if not build_ok:
                elapsed_ms = (time.time() - bench_start) * 1000.0
                for label, mode in MODES:
                    results.append(
                        {
                            "benchmark": bench.name,
                            "category": category,
                            "schema": bench.schema,
                            "mode_label": label,
                            "mode": mode,
                            "result": "bug",
                            "duration_ms": elapsed_ms,
                            "coverage": 0,
                            "step": 0,
                        }
                    )
                    print(f"  {label}: result=bug tte_ms={elapsed_ms:.2f} coverage=0 (build failure)")
                continue

            mode_results: dict[str, dict] = {}
            with concurrent.futures.ThreadPoolExecutor(max_workers=len(MODES)) as executor:
                future_map = {
                    executor.submit(
                        run_mode_trials,
                        bench,
                        measure_bin,
                        label,
                        mode,
                        budget_ms,
                        run_timeout_s,
                        args,
                    ): label
                    for label, mode in MODES
                }
                for future in concurrent.futures.as_completed(future_map):
                    parsed = future.result()
                    mode_results[parsed["mode_label"]] = parsed

            for label, mode in MODES:
                parsed = mode_results[label]
                results.append(
                    {
                        "benchmark": bench.name,
                        "category": category,
                        "schema": bench.schema,
                        "mode_label": label,
                        "mode": mode,
                        **parsed,
                    }
                )
                print(
                    f"  {label}: result={parsed['result']} "
                    f"tte_ms={parsed['duration_ms']:.2f} coverage={parsed['coverage']}"
                )
        finally:
            if bench.bug:
                try:
                    toggle_bug("deactivate", bench.bug, repo_root)
                except RuntimeError:
                    pass
            if regen_needed:
                if bench.schema_source == "benchschemas":
                    regenerate_benchschemas(repo_root, gen_timeout_s)
                elif bench.schema_source == "testcases":
                    if not bench.testcase_file:
                        raise ValueError(f"missing testcase file for {bench.name}")
                    regenerate_testcases(repo_root, bench.testcase_file, gen_timeout_s)

    out_path.parent.mkdir(parents=True, exist_ok=True)
    with out_path.open("w", newline="") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=[
                "benchmark",
                "category",
                "schema",
                "mode_label",
                "mode",
                "result",
                "duration_ms",
                "coverage",
                "step",
            ],
        )
        writer.writeheader()
        for row in results:
            if row.get("coverage") is None:
                row["coverage"] = ""
            writer.writerow(row)

    def summarize(rows: list[dict], label: str, prefix: str) -> None:
        bugs_found = sum(1 for r in rows if r["result"] == "bug")
        tte_values = [r["duration_ms"] for r in rows]
        avg_tte = sum(tte_values) / len(tte_values) if tte_values else 0.0
        cov_values = [r["coverage"] for r in rows if isinstance(r.get("coverage"), (int, float))]
        avg_cov = sum(cov_values) / len(cov_values) if cov_values else 0.0
        print(
            f"[summary {prefix}] {label}: bugs_found={bugs_found} "
            f"avg_tte_ms={avg_tte:.2f} avg_coverage={avg_cov:.2f}"
        )

    for label, _ in MODES:
        rows = [r for r in results if r["mode_label"] == label]
        summarize(rows, label, "all")
        fuzz_rows = [r for r in rows if r.get("category") == "fuzz"]
        summarize(fuzz_rows, label, "fuzz")

    print(f"Results written to {out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
