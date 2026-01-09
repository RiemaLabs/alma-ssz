package main

import (
	"flag"
	"fmt"
	"time"

	"alma.local/ssz/benchschemas"
	"alma.local/ssz/rl"
	"alma.local/ssz/schemas"
	ssz "github.com/ferranbt/fastssz"
	testcases "github.com/ferranbt/fastssz/sszgen/testcases"
)

func main() {
	var (
		schemaName      = flag.String("schema", "BitvectorStruct", "Schema to fuzz")
		mode            = flag.String("mode", "baseline", "baseline | norl | rl")
		budget          = flag.Duration("budget", 30*time.Minute, "Wall-clock budget")
		maxSteps        = flag.Int("max-steps", 50000, "Max steps per episode")
		batchSize       = flag.Int("batch-size", 50, "Batch size")
		requireBV       = flag.Bool("require-bitvector-bug", false, "Only stop when Bitvector dirty padding bug is hit")
		schemaValidate  = flag.Bool("schema-validate", false, "Use schema parameter validation mode (py-ssz only)")
		externalOracle  = flag.String("oracle", "", "External oracle identifier")
		externalBug     = flag.String("oracle-bug", "", "External oracle bug toggle")
		disableTail     = flag.Bool("no-tail", false, "Disable tail mutations")
		disableGap      = flag.Bool("no-gap", false, "Disable offset gap mutations")
		enableBitlistNL = flag.Bool("bitlist-null", false, "Enable bitlist null-sentinel mutation")
	)
	flag.Parse()

	isBaseline := false
	noRL := false
	switch *mode {
	case "baseline":
		isBaseline = true
	case "norl":
		noRL = true
	case "rl":
		// full RL
	default:
		panic("unknown mode")
	}

	if *schemaValidate {
		opts := rl.SchemaValidationOpts{
			CaseName:       *schemaName,
			BatchSize:      *batchSize,
			MaxSteps:       *maxSteps,
			IsBaseline:     isBaseline,
			NoRL:           noRL,
			ExternalOracle: *externalOracle,
			ExternalBug:    *externalBug,
			D_ctx:          7,
		}
		start := time.Now()
		metrics, err := rl.RunSchemaValidationMetrics(opts, *budget)
		dur := time.Since(start)
		if err != nil {
			fmt.Printf("MODE=%s SCHEMA=%s RESULT=timeout DURATION=%s COVERAGE=%.0f\n", *mode, *schemaName, dur, metrics.Coverage)
			return
		}
		fmt.Printf("MODE=%s SCHEMA=%s RESULT=bug STEP=%d DURATION=%s COVERAGE=%.0f\n", *mode, *schemaName, metrics.BugStep, dur, metrics.Coverage)
		return
	}

	var targetSchema ssz.Unmarshaler
	switch *schemaName {
	case "BitvectorStruct":
		targetSchema = &schemas.BitvectorStruct{}
	case "HardBitvectorStruct":
		targetSchema = &schemas.HardBitvectorStruct{}
	case "BitvectorPairStruct":
		targetSchema = &schemas.BitvectorPairStruct{}
	case "BitvectorWideStruct":
		targetSchema = &schemas.BitvectorWideStruct{}
	case "BitvectorOffsetStruct":
		targetSchema = &schemas.BitvectorOffsetStruct{}
	case "BitvectorScatterStruct":
		targetSchema = &schemas.BitvectorScatterStruct{}
	case "BooleanStruct":
		targetSchema = &schemas.BooleanStruct{}
	case "HardBooleanStruct":
		targetSchema = &schemas.HardBooleanStruct{}
	case "BooleanPairStruct":
		targetSchema = &schemas.BooleanPairStruct{}
	case "BooleanWideStruct":
		targetSchema = &schemas.BooleanWideStruct{}
	case "BooleanOffsetStruct":
		targetSchema = &schemas.BooleanOffsetStruct{}
	case "BooleanScatterStruct":
		targetSchema = &schemas.BooleanScatterStruct{}
	case "GapStruct":
		targetSchema = &schemas.GapStruct{}
	case "HardGapStruct":
		targetSchema = &schemas.HardGapStruct{}
	case "GapPairStruct":
		targetSchema = &schemas.GapPairStruct{}
	case "GapWideStruct":
		targetSchema = &schemas.GapWideStruct{}
	case "GapTriStruct":
		targetSchema = &schemas.GapTriStruct{}
	case "GapScatterStruct":
		targetSchema = &schemas.GapScatterStruct{}
	case "UnionStruct":
		targetSchema = &schemas.UnionStruct{}
	case "HardUnionStruct":
		targetSchema = &schemas.HardUnionStruct{}
	case "UnionWideStruct":
		targetSchema = &schemas.UnionWideStruct{}
	case "UnionScatterStruct":
		targetSchema = &schemas.UnionScatterStruct{}
	case "BeaconState":
		targetSchema = &schemas.BeaconState{}
	case "Validator":
		targetSchema = &schemas.Validator{}
	case "PendingAttestation":
		targetSchema = &schemas.PendingAttestation{}
	case "AggregationBitsContainer":
		targetSchema = &schemas.AggregationBitsContainer{}
	case "BitlistPairStruct":
		targetSchema = &schemas.BitlistPairStruct{}
	case "BitlistWideStruct":
		targetSchema = &schemas.BitlistWideStruct{}
	case "BitlistTriStruct":
		targetSchema = &schemas.BitlistTriStruct{}
	case "BitlistOffsetStruct":
		targetSchema = &schemas.BitlistOffsetStruct{}
	case "BeaconStateBench":
		targetSchema = &benchschemas.BeaconStateBench{}
	case "ValidatorEnvelope":
		targetSchema = &benchschemas.ValidatorEnvelope{}
	case "AttestationEnvelope":
		targetSchema = &benchschemas.AttestationEnvelope{}
	case "BlockBodyBench":
		targetSchema = &benchschemas.BlockBodyBench{}
	case "UnionBench":
		targetSchema = &benchschemas.UnionBench{}
	case "PSSZBoolBench":
		targetSchema = &benchschemas.PSSZBoolBench{}
	case "PSSZBitvectorBench":
		targetSchema = &benchschemas.PSSZBitvectorBench{}
	case "PSSZBitlistBench":
		targetSchema = &benchschemas.PSSZBitlistBench{}
	case "PSSZByteListBench":
		targetSchema = &benchschemas.PSSZByteListBench{}
	case "PSSZGapBench":
		targetSchema = &benchschemas.PSSZGapBench{}
	case "PSSZTailBench":
		targetSchema = &benchschemas.PSSZTailBench{}
	case "PSSZHTRListBench":
		targetSchema = &benchschemas.PSSZHTRListBench{}
	case "PSSZHeaderListBench":
		targetSchema = &benchschemas.PSSZHeaderListBench{}
	case "FSSZ-9":
		targetSchema = &benchschemas.AttestationEnvelope{}
	case "FSSZ-23":
		targetSchema = &benchschemas.BlockBodyBench{}
	case "FSSZ-181":
		targetSchema = &testcases.Case4{}
	case "FSSZ-162":
		targetSchema = &testcases.Uints{}
	case "FSSZ-152":
		targetSchema = &testcases.PR1512{}
	case "FSSZ-127":
		targetSchema = &testcases.Obj2{}
	case "FSSZ-54":
		targetSchema = &testcases.ListP{}
	case "FSSZ-76":
		targetSchema = &testcases.Case1A{}
	case "FSSZ-153":
		targetSchema = &testcases.Issue153{}
	case "FSSZ-158":
		targetSchema = &testcases.Issue158{}
	case "FSSZ-166":
		targetSchema = &testcases.Issue165{}
	case "FSSZ-136":
		targetSchema = &testcases.Issue136{}
	case "FSSZ-156":
		targetSchema = &testcases.Issue156{}
	case "FSSZ-159":
		targetSchema = &testcases.Issue159[[48]byte]{}
	case "FSSZ-164":
		targetSchema = &testcases.Issue64{}
	case "FSSZ-188":
		targetSchema = &testcases.Issue188{}
	case "FSSZ-86":
		targetSchema = &testcases.Case2B{}
	case "FSSZ-100":
		targetSchema = &testcases.TimeType{}
	case "FSSZ-149":
		targetSchema = &testcases.IntegrationUint{}
	case "FSSZ-151":
		targetSchema = &testcases.ListC{}
	case "FSSZ-1":
		targetSchema = &testcases.ListP{}
	case "FSSZ-52":
		targetSchema = &testcases.Case4{}
	case "FSSZ-173":
		targetSchema = &benchschemas.BeaconStateBench{}
	case "FSSZ-147":
		targetSchema = &schemas.PendingAttestation{}
	case "FSSZ-119":
		targetSchema = &benchschemas.BlockBodyBench{}
	case "FSSZ-111":
		targetSchema = &schemas.BeaconState{}
	case "FSSZ-110":
		targetSchema = &benchschemas.BeaconBlockHeader{}
	case "FSSZ-98":
		targetSchema = &testcases.ListP{}
	case "FSSZ-96":
		targetSchema = &benchschemas.ValidatorEnvelope{}
	default:
		panic("unknown schema")
	}

	opts := rl.RLOpts{
		Episodes:            1,
		MaxSteps:            *maxSteps,
		AgentType:           "policy",
		SchemaName:          *schemaName,
		BatchSize:           *batchSize,
		D_ctx:               7,
		RequireBitvectorBug: *requireBV,
		ExternalOracle:      *externalOracle,
		ExternalBug:         *externalBug,
		DisableTail:         *disableTail,
		DisableGap:          *disableGap,
		EnableBitlistNull:   *enableBitlistNL,
		IsBaseline:          isBaseline,
		NoRL:                noRL,
	}

	start := time.Now()
	metrics, err := rl.RunUntilBugMetrics(targetSchema, opts, *budget)
	dur := time.Since(start)
	if err != nil {
		fmt.Printf("MODE=%s SCHEMA=%s RESULT=timeout DURATION=%s COVERAGE=%.0f\n", *mode, *schemaName, dur, metrics.Coverage)
		return
	}
	fmt.Printf("MODE=%s SCHEMA=%s RESULT=bug STEP=%d DURATION=%s COVERAGE=%.0f\n", *mode, *schemaName, metrics.BugStep, dur, metrics.Coverage)
}
