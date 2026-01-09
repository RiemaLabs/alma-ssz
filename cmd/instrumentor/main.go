package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/token"
	"hash/fnv"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

var (
	targetDir string
	metadata  = make(map[string]VarInfo)
	targetFile string
)

type VarInfo struct {
	CID         uint64
	PackageName string
	FuncName    string
	BlockID     int
	VarName     string
	Location    string
}

type Metadata struct {
	Columns []string           // List of CIDs in string format, defining the vector order
	Details map[string]VarInfo // Details for each CID
}

func main() {
	flag.StringVar(&targetFile, "file", "./schemas/schemas_encoding.go", "Go file to instrument")
	flag.Parse()

	log.Printf("Instrumenting file: %s", targetFile)

	err := instrumentFile(targetFile) // Directly call instrumentFile
	if err != nil {
		log.Fatalf("Error instrumenting file %s: %v", targetFile, err)
	}

	saveMetadata()
	log.Println("Instrumentation complete.")
}

func saveMetadata() {
	// Create corpus dir if not exists
	if err := os.MkdirAll("corpus", 0755); err != nil {
		log.Fatalf("Failed to create corpus dir: %v", err)
	}

	// Convert map to sorted columns
	var columns []string
	for cid := range metadata {
		columns = append(columns, cid)
	}
	sort.Strings(columns)

	meta := Metadata{
		Columns: columns,
		Details: metadata,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal metadata: %v", err)
	}

	if err := ioutil.WriteFile("corpus/metadata.json", data, 0644); err != nil {
		log.Fatalf("Failed to write metadata: %v", err)
	}
	log.Printf("Saved metadata for %d dimensions to corpus/metadata.json", len(columns))
}

func instrumentFile(path string) error {
	code, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	f, err := decorator.Parse(code)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Inject import
	needsImport := true
	for _, imp := range f.Imports {
		if imp.Path != nil && imp.Path.Value == "\"github.com/ferranbt/fastssz/tracer\"" {
			needsImport = false
			break
		}
	}

	if needsImport {
		importDecl := &dst.GenDecl{
			Tok: token.IMPORT,
			Specs: []dst.Spec{
				&dst.ImportSpec{
					Path: &dst.BasicLit{Kind: token.STRING, Value: "\"github.com/ferranbt/fastssz/tracer\""},
				},
			},
		}
		f.Decls = append([]dst.Decl{importDecl}, f.Decls...)
	}

	packageName := f.Name.Name
	var currentFunc string
	blockCounter := 0

	dstutil.Apply(f, func(c *dstutil.Cursor) bool {
		node := c.Node()

		switch n := node.(type) {
		case *dst.FuncDecl:
			currentFunc = n.Name.Name
			blockCounter = 0 // Reset for new function
		case *dst.BlockStmt, *dst.CaseClause, *dst.CommClause:
			blockCounter++
		}

		return true
	}, func(c *dstutil.Cursor) bool {
		node := c.Node()

		switch n := node.(type) {
		case *dst.AssignStmt:
			if c.Index() < 0 {
				return true
			}

			for _, lhs := range n.Lhs {
				if ident, ok := lhs.(*dst.Ident); ok {
					if ident.Name == "_" {
						continue
					}

					// Generate CID
				h := fnv.New64a()
				h.Write([]byte(packageName))
				h.Write([]byte(currentFunc))
				h.Write([]byte(strconv.Itoa(blockCounter)))
				h.Write([]byte(ident.Name))
				cidRaw := h.Sum64()
				cidStr := fmt.Sprintf("%d", cidRaw)

					// Store Metadata
					metadata[cidStr] = VarInfo{
							CID:         cidRaw,
							PackageName: packageName,
							FuncName:    currentFunc,
							BlockID:     blockCounter,
							VarName:     ident.Name,
							Location:    path,
					}

					// Create CallStmt
					call := &dst.ExprStmt{
						X: &dst.CallExpr{
							Fun: &dst.SelectorExpr{
								X:   &dst.Ident{Name: "tracer"},
								Sel: &dst.Ident{Name: "Record"},
							},
							Args: []dst.Expr{
								&dst.BasicLit{Kind: token.INT, Value: cidStr},
								&dst.CallExpr{
									Fun: &dst.SelectorExpr{
										X:   &dst.Ident{Name: "tracer"},
										Sel: &dst.Ident{Name: "ToScalar"},
									},
									Args: []dst.Expr{
										&dst.Ident{Name: ident.Name},
									},
								},
							},
						},
					}

					c.InsertAfter(call)
				}
			}
		}
		return true
	})

	f_out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f_out.Close()

	return decorator.Fprint(f_out, f)
}
