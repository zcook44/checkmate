package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zcook44/checkmate/internal/checklist"
)

type stage int

const (
	browse stage = iota
	choose
	comment
	confirm
	search
	details
)

type match struct {
	file     int
	rule     int
	selected bool
}
type group struct {
	key     string
	sample  checklist.Rule
	matches []match
}
type model struct {
	files                        []checklist.File
	groups                       []group
	cursor, host, filter         int
	stage                        stage
	target, note, query, message string
	width                        int
}

var filters = []string{"all", "not_reviewed", "open", "not_a_finding", "not_applicable"}
var choices = []string{"open", "not_a_finding", "not_applicable", "not_reviewed"}

func newModel(files []checklist.File) model {
	m := model{files: files, width: 100}
	indexes := map[string]int{}
	for fi, f := range files {
		for ri, r := range f.Rules {
			k := r.Key()
			idx, ok := indexes[k]
			if !ok {
				idx = len(m.groups)
				indexes[k] = idx
				m.groups = append(m.groups, group{key: k, sample: r})
			}
			m.groups[idx].matches = append(m.groups[idx].matches, match{file: fi, rule: ri, selected: true})
		}
	}
	sort.Slice(m.groups, func(i, j int) bool {
		severity := func(s string) int {
			switch s {
			case "high":
				return 0
			case "medium":
				return 1
			default:
				return 2
			}
		}
		a, b := m.groups[i].sample, m.groups[j].sample
		if severity(a.Severity) != severity(b.Severity) {
			return severity(a.Severity) < severity(b.Severity)
		}
		if a.Benchmark != b.Benchmark {
			return a.Benchmark < b.Benchmark
		}
		return a.ID < b.ID
	})
	return m
}
func (m model) Init() tea.Cmd { return nil }
func (m model) visible() []int {
	out := []int{}
	for i, g := range m.groups {
		matchFilter := m.filter == 0
		if !matchFilter {
			for _, ref := range g.matches {
				if m.files[ref.file].Rules[ref.rule].Status == filters[m.filter] {
					matchFilter = true
					break
				}
			}
		}
		if !matchFilter {
			continue
		}
		if m.query != "" && !strings.Contains(strings.ToLower(g.sample.Benchmark+" "+g.sample.ID+" "+g.sample.RuleID+" "+g.sample.Title), strings.ToLower(m.query)) {
			continue
		}
		out = append(out, i)
	}
	return out
}
func (m *model) clamp() {
	n := len(m.visible())
	if n == 0 {
		m.cursor = 0
		m.host = 0
		return
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	matches := len(m.groups[m.visible()[m.cursor]].matches)
	if m.host >= matches {
		m.host = 0
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if m.stage == search {
			if k == "esc" {
				m.stage = browse
				return m, nil
			}
			if k == "enter" {
				m.stage = browse
				m.cursor = 0
				m.clamp()
				return m, nil
			}
			if k == "backspace" {
				m.query = trimLast(m.query)
			} else if msg.Type == tea.KeySpace {
				m.query += " "
			} else if msg.Type == tea.KeyRunes {
				m.query += safe(string(msg.Runes))
			}
			return m, nil
		}
		if m.stage == comment {
			switch k {
			case "esc":
				m.stage = browse
				return m, nil
			case "enter":
				if strings.TrimSpace(m.note) != "" {
					m.stage = confirm
				} else {
					m.message = "A comment is required"
				}
				return m, nil
			case "backspace":
				m.note = trimLast(m.note)
				return m, nil
			}
			if msg.Type == tea.KeySpace {
				m.note += " "
			}
			if msg.Type == tea.KeyRunes {
				m.note += safe(string(msg.Runes))
			}
			return m, nil
		}
		if m.stage == choose {
			if k == "esc" {
				m.stage = browse
				return m, nil
			}
			states := map[string]string{"1": "open", "2": "not_a_finding", "3": "not_applicable", "4": "not_reviewed", "o": "open", "n": "not_a_finding", "a": "not_applicable", "r": "not_reviewed"}
			if v, ok := states[k]; ok {
				m.target = v
				m.stage = comment
				m.note = ""
			}
			return m, nil
		}
		if m.stage == confirm {
			if k == "esc" || k == "n" {
				m.stage = browse
				return m, nil
			}
			if k == "y" || k == "enter" {
				m.apply()
			}
			return m, nil
		}
		if m.stage == details {
			if k == "esc" || k == "v" {
				m.stage = browse
			}
			return m, nil
		}
		visible := m.visible()
		switch k {
		case "q", "esc":
			return m, tea.Quit
		case "j", "down":
			m.cursor++
			m.host = 0
		case "k", "up":
			m.cursor--
			m.host = 0
		case "tab":
			if len(visible) > 0 {
				m.host = (m.host + 1) % len(m.groups[visible[m.cursor]].matches)
			}
		case "space", " ":
			if len(visible) > 0 {
				g := &m.groups[visible[m.cursor]]
				g.matches[m.host].selected = !g.matches[m.host].selected
			}
		case "f":
			m.filter = (m.filter + 1) % len(filters)
			m.cursor = 0
			m.host = 0
		case "/":
			m.stage = search
		case "v":
			if len(visible) > 0 {
				m.stage = details
			}
		case "e", "enter":
			if len(visible) > 0 {
				m.stage = choose
				m.message = ""
			}
		}
		m.clamp()
	}
	return m, nil
}
func (m *model) apply() {
	visible := m.visible()
	if len(visible) == 0 {
		m.stage = browse
		return
	}
	g := &m.groups[visible[m.cursor]]
	changed := 0
	for i := range g.matches {
		ref := &g.matches[i]
		if !ref.selected {
			continue
		}
		f := m.files[ref.file]
		if err := checklist.Edit(f, ref.rule, m.target, m.note); err != nil {
			m.message = fmt.Sprintf("Stopped after %d updates: %v", changed, err)
			m.stage = browse
			return
		}
		loaded, err := checklist.Load(f.Path)
		if err != nil {
			m.message = fmt.Sprintf("Updated %s but reload failed: %v", f.Path, err)
			m.stage = browse
			return
		}
		m.files[ref.file] = loaded
		changed++
	}
	m.message = fmt.Sprintf("Updated %d checklist(s) in place", changed)
	m.stage = browse
}
func trimLast(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[:len(r)-1])
}

// Do not permit checklist text or input to inject terminal control sequences.
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
func clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}
func (m model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "CHECKMATE  |  CKLB review  |  %d files  |  %d control groups\n", len(m.files), len(m.groups))
	fmt.Fprintf(&b, "Filter: %s  Search: %s  (%d matches)\n", filters[m.filter], m.query, len(m.visible()))
	if m.message != "" {
		fmt.Fprintf(&b, "\n%s\n", safe(m.message))
	}
	visible := m.visible()
	if len(visible) == 0 {
		b.WriteString("\nNo controls match.  / search  f filter  q quit\n")
		return b.String()
	}
	g := m.groups[visible[m.cursor]]
	r := g.sample
	fmt.Fprintf(&b, "\n[%d/%d] %s  %s  severity: %s\n", m.cursor+1, len(visible), safe(r.Benchmark), safe(r.ID), strings.ToUpper(safe(r.Severity)))
	fmt.Fprintf(&b, "Rule: %s  |  Release: %s\n", safe(r.RuleID), safe(r.Release))
	fmt.Fprintf(&b, "Summary: %s\n", clip(safe(r.Title), max(40, m.width-10)))
	b.WriteString("\nAffected checklists (Tab: focus host; Space: include/exclude):\n")
	for i, ref := range g.matches {
		row := m.files[ref.file].Rules[ref.rule]
		selected := "x"
		if !ref.selected {
			selected = " "
		}
		focus := " "
		if i == m.host {
			focus = ">"
		}
		fmt.Fprintf(&b, "%s [%s] %-22s  %-18s  %s\n", focus, selected, clip(safe(filepath.Base(filepath.Dir(m.files[ref.file].Path))), 22), row.Status, clip(safe(m.files[ref.file].Path), max(30, m.width-60)))
	}
	focused := g.matches[m.host]
	current := m.files[focused.file].Rules[focused.rule]
	if m.stage == details {
		return fmt.Sprintf("Full finding details | %s %s | %s\n\n%s\n\nFull current comment:\n%s\n\nEsc or v: return\n", safe(current.Benchmark), safe(current.ID), safe(m.files[focused.file].Path), safe(current.Findings), safe(current.Comment))
	}
	fmt.Fprintf(&b, "\nFocused host: %s\n", safe(m.files[focused.file].Path))
	fmt.Fprintf(&b, "Current status: %s\nCurrent comment: %s\n", current.Status, clip(safe(current.Comment), max(50, m.width-8)))
	fmt.Fprintf(&b, "Finding details: %s\n", clip(safe(current.Findings), max(80, m.width-8)))
	switch m.stage {
	case browse:
		b.WriteString("\nj/k navigate  Tab host  Space select  v full details  e edit  f filter  / search  q quit\n")
	case search:
		fmt.Fprintf(&b, "\nSearch: %s_  Enter apply  Esc cancel\n", m.query)
	case choose:
		b.WriteString("\nChoose state: [1] Open  [2] Not a Finding  [3] Not Applicable  [4] Not Reviewed  Esc cancel\n")
	case comment:
		fmt.Fprintf(&b, "\nNew state: %s\nComment (required): %s_\nEnter review change  Esc cancel\n", m.target, m.note)
	case confirm:
		fmt.Fprintf(&b, "\nCONFIRM: %s -> %s\nComment: %s\n", r.ID, m.target, m.note)
		for _, ref := range g.matches {
			if ref.selected {
				fmt.Fprintf(&b, "UPDATE %s\n", safe(m.files[ref.file].Path))
			}
		}
		b.WriteString("Existing CKLB files replaced; provenance sidecars remain stale.\nY/Enter apply  N/Esc cancel\n")
	}
	return b.String()
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func main() {
	root := flag.String("root", "", "root directory of CKLB checklists")
	list := flag.Bool("list", false, "read-only inventory, no TUI")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "usage: checkmate --root /private/checklists [--list]")
		os.Exit(2)
	}
	files, err := checklist.Discover(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *list {
		m := newModel(files)
		for _, g := range m.groups {
			fmt.Printf("%s  %s  %s  %s  (%d checklists)\n", g.sample.Benchmark, g.sample.ID, g.sample.Severity, clip(safe(g.sample.Title), 90), len(g.matches))
		}
		return
	}
	if _, err = tea.NewProgram(newModel(files), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
