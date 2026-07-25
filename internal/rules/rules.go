// Package rules reads, edits and applies auditd rule files in
// /etc/audit/rules.d. Files are edited as whole-text with a .bak backup;
// parsing exists to render structured chips and to catch syntax errors
// before augenrules --load replaces the live ruleset.
package rules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Manager struct {
	Dir string // /etc/audit/rules.d
}

// Entry is one line of a rules file, parsed for display when possible.
type Entry struct {
	Line int               `json:"line"`
	Raw  string            `json:"raw"`
	Kind string            `json:"kind"` // watch | syscall | control | comment | blank | other
	// Parsed pieces for watch/syscall kinds.
	Action   string          `json:"action,omitempty"`    // always | never (syscall)
	Path     string          `json:"path,omitempty"`      // -w path, or -F path=/-F dir= value
	PathKind string          `json:"path_kind,omitempty"` // watch | path | dir
	Perms  string            `json:"perms,omitempty"`  // -p/-F perm= value
	Key    string            `json:"key,omitempty"`    // -k/-F key= value
	Fields map[string]string `json:"fields,omitempty"` // remaining -F name=value
	Err    string            `json:"err,omitempty"`    // parse problem, if any
}

type File struct {
	Name    string   `json:"name"`
	Content string   `json:"content"`
	Entries []*Entry `json:"entries"`
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.rules$`)

func (m *Manager) safePath(name string) (string, error) {
	if !nameRe.MatchString(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("недопустимое имя файла: %q", name)
	}
	return filepath.Join(m.Dir, name), nil
}

func (m *Manager) List() ([]*File, error) {
	glob, err := filepath.Glob(filepath.Join(m.Dir, "*.rules"))
	if err != nil {
		return nil, err
	}
	sort.Strings(glob) // augenrules merges in lexicographic order
	var files []*File
	for _, p := range glob {
		f, err := m.Read(filepath.Base(p))
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (m *Manager) Read(name string) (*File, error) {
	p, err := m.safePath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return &File{Name: name, Content: string(data), Entries: Parse(string(data))}, nil
}

// Write saves content, keeping the previous version as <name>.bak
// (augenrules only merges *.rules, so backups are inert).
func (m *Manager) Write(name, content string) error {
	p, err := m.safePath(name)
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(p); err == nil {
		if err := os.WriteFile(p+".bak", old, 0o640); err != nil {
			return fmt.Errorf("бэкап не записался, отмена: %w", err)
		}
	}
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}
	return os.WriteFile(p, []byte(content), 0o640)
}

func (m *Manager) Delete(name string) error {
	p, err := m.safePath(name)
	if err != nil {
		return err
	}
	// Keep a backup even on delete.
	if old, err := os.ReadFile(p); err == nil {
		_ = os.WriteFile(p+".bak", old, 0o640)
	}
	return os.Remove(p)
}

// Validate returns per-line problems in the supported syntax subset.
func Validate(content string) []string {
	var problems []string
	for _, e := range Parse(content) {
		if e.Err != "" {
			problems = append(problems, fmt.Sprintf("строка %d: %s (%s)", e.Line, e.Err, e.Raw))
		}
	}
	return problems
}

type ApplyResult struct {
	OK          bool     `json:"ok"`
	Output      string   `json:"output"`       // augenrules --load combined output
	Loaded      []string `json:"loaded"`       // auditctl -l after load
	Status      string   `json:"status"`       // auditctl -s after load
	CheckOutput string   `json:"check_output"` // augenrules --check
}

func run(name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s не найден: %w", name, err)
	}
	out, err := exec.Command(path, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Apply loads the merged rules.d into the kernel and reports the result.
func (m *Manager) Apply() *ApplyResult {
	res := &ApplyResult{}
	out, err := run("augenrules", "--load")
	res.Output = out
	if err != nil {
		res.Output += "\n" + err.Error()
		// fall through: still report what actually got loaded
	}
	res.OK = err == nil
	if l, lerr := run("auditctl", "-l"); lerr == nil {
		res.Loaded = strings.Split(l, "\n")
	}
	if s, serr := run("auditctl", "-s"); serr == nil {
		res.Status = s
	}
	if c, _ := run("augenrules", "--check"); c != "" {
		res.CheckOutput = c
	}
	return res
}

// Status reports live kernel state without changing anything.
func (m *Manager) Status() *ApplyResult {
	res := &ApplyResult{OK: true}
	if l, err := run("auditctl", "-l"); err == nil {
		res.Loaded = strings.Split(l, "\n")
	} else {
		res.OK = false
		res.Output = err.Error()
	}
	if s, err := run("auditctl", "-s"); err == nil {
		res.Status = s
	}
	if c, _ := run("augenrules", "--check"); c != "" {
		res.CheckOutput = c
	}
	return res
}

// Parse splits rules-file content into display entries. It understands
// the -w and -a forms; anything else parseable stays as "other"/"control"
// and is preserved verbatim on save (editing is whole-text).
func Parse(content string) []*Entry {
	var entries []*Entry
	for i, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		e := parseLine(strings.TrimSpace(line))
		e.Line = i + 1
		e.Raw = line
		entries = append(entries, e)
	}
	return entries
}

func parseLine(s string) *Entry {
	switch {
	case s == "":
		return &Entry{Kind: "blank"}
	case strings.HasPrefix(s, "#"):
		return &Entry{Kind: "comment"}
	}
	tok := strings.Fields(s)
	switch tok[0] {
	case "-w":
		return parseWatch(tok)
	case "-a", "-A":
		return parseSyscall(tok)
	case "-D", "-b", "-f", "-e", "--backlog_wait_time", "-r", "--loginuid-immutable":
		return &Entry{Kind: "control"}
	default:
		return &Entry{Kind: "other"}
	}
}

func parseWatch(tok []string) *Entry {
	e := &Entry{Kind: "watch", Action: "always", PathKind: "watch"}
	for i := 1; i < len(tok); i++ {
		switch tok[i] {
		case "-w":
			// already consumed as tok[0]
		case "-p":
			if i+1 < len(tok) {
				i++
				e.Perms = tok[i]
			}
		case "-k":
			if i+1 < len(tok) {
				i++
				e.Key = tok[i]
			}
		default:
			if e.Path == "" && !strings.HasPrefix(tok[i], "-") {
				e.Path = tok[i]
			} else {
				e.Err = "неизвестный параметр " + tok[i]
			}
		}
	}
	if e.Path == "" && len(tok) > 1 {
		e.Path = tok[1]
	}
	if e.Path == "" {
		e.Err = "нет пути"
	}
	return e
}

func parseSyscall(tok []string) *Entry {
	e := &Entry{Kind: "syscall", Fields: map[string]string{}}
	if len(tok) < 2 {
		return &Entry{Kind: "syscall", Err: "нет action,list"}
	}
	parts := strings.SplitN(tok[1], ",", 2)
	e.Action = parts[0]
	if e.Action != "always" && e.Action != "never" {
		// -a list,action is also legal ("exit,always")
		if len(parts) == 2 && (parts[1] == "always" || parts[1] == "never") {
			e.Action = parts[1]
		} else {
			e.Err = "неизвестное действие " + tok[1]
		}
	}
	for i := 2; i < len(tok); i++ {
		switch tok[i] {
		case "-F":
			if i+1 >= len(tok) {
				e.Err = "-F без значения"
				continue
			}
			i++
			kv := strings.SplitN(tok[i], "=", 2)
			if len(kv) != 2 {
				// operators like != are rare in our subset; keep raw
				e.Fields[tok[i]] = ""
				continue
			}
			switch kv[0] {
			case "path", "dir":
				e.Path = kv[1]
				e.PathKind = kv[0]
			case "perm":
				e.Perms = kv[1]
			case "key":
				e.Key = kv[1]
			default:
				e.Fields[kv[0]] = kv[1]
			}
		case "-k":
			if i+1 < len(tok) {
				i++
				e.Key = tok[i]
			}
		case "-S":
			if i+1 < len(tok) {
				i++
				e.Fields["syscall"] = tok[i]
			}
		case "-C":
			if i+1 < len(tok) {
				i++
				e.Fields["compare"] = tok[i]
			}
		default:
			e.Err = "неизвестный параметр " + tok[i]
		}
	}
	if len(e.Fields) == 0 {
		e.Fields = nil
	}
	return e
}
