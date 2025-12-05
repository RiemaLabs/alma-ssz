# CSVV Experiment Runner

This directory contains the runner for the Context-Sensitive Variable Vectorization (CSVV) experiment.

## Prerequisites

1.  **Instrumentation**: The target library (`workspace/fastssz`) must be instrumented.
    Run the instrumentor from the project root:
    ```bash
    go run cmd/instrumentor/main.go -dir ./workspace/fastssz
    ```
    This injects `tracer.Record` calls into the source code.

2.  **Metadata**: The instrumentor generates `corpus/metadata.json` which maps Context IDs (CIDs) to variable names and locations. This file is required by the runner.

## Running the Experiment

Run the main runner from the project root:

```bash
go run cmd/csvv/main.go
```

## What it does

1.  **Verification**: It runs `ssz.DemonstrateBranching(true)` and `ssz.DemonstrateBranching(false)` to verify that the same variable `x` gets different CIDs in different branches (Context Sensitivity).
2.  **Fuzz Loop**: It runs a loop generating random inputs (currently placeholders) and executing the target function.
3.  **Analysis**: It uses the `internal/analyzer` package to compute the KL Divergence score of each execution trace against the global history.
    -   **Structural Divergence**: New code paths (new CIDs).
    -   **Numerical Divergence**: Rare values for known CIDs.
4.  **Output**: It prints the score for each interesting trace and the total number of dimensions (unique context-variable pairs) discovered.

## Troubleshooting

-   **Import Cycles**: If you see import cycle errors, ensure `workspace/fastssz/tracer` is not instrumented. The instrumentor should skip the `tracer` directory. If needed, run `git restore .` in `workspace/fastssz` and re-run the instrumentor.
