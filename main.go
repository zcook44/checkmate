package main

import (
	"flag"
	"fmt"
	"os"

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
	height, detailOffset         int
	stage                        stage
	target, note, query, message string
	width                        int
}

var filters = []string{"all", "not_reviewed", "open", "not_a_finding", "not_applicable"}
var choices = []string{"open", "not_a_finding", "not_applicable", "not_reviewed"}

func newModel(files []checklist.File) model {
	m := model{files: files, width: 100, height: 30}
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
		m.height = msg.Height
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
					m.host = 0
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
			m.moveHost(k, m.selectedCount())
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
			switch k {
			case "j", "down":
				m.detailOffset++
			case "k", "up":
				m.detailOffset = max(0, m.detailOffset-1)
			case "pgdown":
				m.detailOffset += m.pageSize()
			case "pgup":
				m.detailOffset = max(0, m.detailOffset-m.pageSize())
			case "home":
				m.detailOffset = 0
			}
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
		case "shift+tab":
			if len(visible) > 0 {
				m.host = (m.host + len(m.groups[visible[m.cursor]].matches) - 1) % len(m.groups[visible[m.cursor]].matches)
			}
		case "pgdown", "pgup", "home", "end":
			if len(visible) > 0 {
				m.moveHost(k, len(m.groups[visible[m.cursor]].matches))
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
				m.detailOffset = 0
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
func (m model) View() string { return m.render() }
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
