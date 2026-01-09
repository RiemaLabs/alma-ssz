package pyssz

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"alma.local/ssz/fuzzer"
)

type request struct {
	Op     string `json:"op"`
	Schema string `json:"schema"`
	Data   string `json:"data,omitempty"`
}

type response struct {
	OK    bool   `json:"ok"`
	Canon string `json:"canon,omitempty"`
	Root  string `json:"root,omitempty"`
	Error string `json:"error,omitempty"`
}

// SchemaCheckResult captures schema validation outcomes.
type SchemaCheckResult struct {
	OK    bool
	Error string
}

// Oracle wraps a persistent py-ssz helper process.
type Oracle struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Reader
	mu     sync.Mutex
}

// NewOracle starts the Python helper with the requested bug toggle.
func NewOracle(schemaName, bugID string) (*Oracle, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("pyssz: getwd: %w", err)
	}
	scriptPath := filepath.Join(repoRoot, "scripts", "py_ssz_oracle.py")
	pyPath := filepath.Join(repoRoot, "workspace", "py-ssz")
	pythonExec := os.Getenv("ALMA_PYSSZ_PYTHON")
	if pythonExec == "" {
		venvPython := filepath.Join(repoRoot, ".venv", "bin", "python3")
		if _, statErr := os.Stat(venvPython); statErr == nil {
			pythonExec = venvPython
		} else {
			pythonExec = "python3"
		}
	}

	cmd := exec.Command(pythonExec, scriptPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PYTHONPATH=%s", pyPath),
		fmt.Sprintf("ALMA_PSSZ_BUG=%s", bugID),
		"PYTHONUNBUFFERED=1",
	)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("pyssz: stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pyssz: stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("pyssz: start: %w", err)
	}

	oracle := &Oracle{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdinPipe),
		stdout: bufio.NewReader(stdoutPipe),
	}

	if err := oracle.ping(schemaName); err != nil {
		_ = oracle.Close()
		return nil, err
	}

	return oracle, nil
}

func (o *Oracle) ping(schema string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	req := request{Op: "ping", Schema: schema}
	if err := o.send(req); err != nil {
		return err
	}
	resp, err := o.recv()
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("pyssz: ping failed: %s", resp.Error)
	}
	return nil
}

func (o *Oracle) send(req request) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("pyssz: marshal request: %w", err)
	}
	if _, err := o.stdin.Write(payload); err != nil {
		return fmt.Errorf("pyssz: write request: %w", err)
	}
	if err := o.stdin.WriteByte('\n'); err != nil {
		return fmt.Errorf("pyssz: write newline: %w", err)
	}
	if err := o.stdin.Flush(); err != nil {
		return fmt.Errorf("pyssz: flush request: %w", err)
	}
	return nil
}

func (o *Oracle) recv() (response, error) {
	line, err := o.stdout.ReadString('\n')
	if err != nil {
		return response{}, fmt.Errorf("pyssz: read response: %w", err)
	}
	line = strings.TrimSpace(line)
	var resp response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return response{}, fmt.Errorf("pyssz: unmarshal response: %w", err)
	}
	return resp, nil
}

// SchemaCheck asks py-ssz to validate a schema parameter (e.g., length bounds).
func (o *Oracle) SchemaCheck(schema string, length uint64) (SchemaCheckResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, length)
	req := request{
		Op:     "schema",
		Schema: schema,
		Data:   hex.EncodeToString(buf),
	}
	if err := o.send(req); err != nil {
		return SchemaCheckResult{}, err
	}
	resp, err := o.recv()
	if err != nil {
		return SchemaCheckResult{}, err
	}
	return SchemaCheckResult{OK: resp.OK, Error: resp.Error}, nil
}

// Decode asks py-ssz to decode and re-encode the input bytes.
func (o *Oracle) Decode(schema string, data []byte) (fuzzer.ExternalDecodeResult, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	req := request{
		Op:     "decode",
		Schema: schema,
		Data:   hex.EncodeToString(data),
	}
	if err := o.send(req); err != nil {
		return fuzzer.ExternalDecodeResult{}, err
	}
	resp, err := o.recv()
	if err != nil {
		return fuzzer.ExternalDecodeResult{}, err
	}
	if !resp.OK {
		return fuzzer.ExternalDecodeResult{}, fmt.Errorf("pyssz: %s", resp.Error)
	}
	canon, err := hex.DecodeString(resp.Canon)
	if err != nil {
		return fuzzer.ExternalDecodeResult{}, fmt.Errorf("pyssz: decode canon hex: %w", err)
	}
	rootBytes, err := hex.DecodeString(resp.Root)
	if err != nil {
		return fuzzer.ExternalDecodeResult{}, fmt.Errorf("pyssz: decode root hex: %w", err)
	}
	if len(rootBytes) != 32 {
		return fuzzer.ExternalDecodeResult{}, fmt.Errorf("pyssz: invalid root length %d", len(rootBytes))
	}
	var root [32]byte
	copy(root[:], rootBytes)
	return fuzzer.ExternalDecodeResult{Canonical: canon, Root: root}, nil
}

// Close shuts down the helper process.
func (o *Oracle) Close() error {
	if o.cmd == nil || o.cmd.Process == nil {
		return nil
	}
	_ = o.cmd.Process.Kill()
	_, _ = o.cmd.Process.Wait()
	return nil
}
