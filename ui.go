package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// STIG Viewer 3.x uses red Open, green Not a Finding, black Not Applicable,
// and gray Not Reviewed. The black symbol sits on a pale badge for dark terminals.
func statusColor(status string) string {
	switch status {
	case "open":
		return "red"
	case "not_a_finding":
		return "green"
	case "not_applicable":
		return "black"
	case "not_reviewed":
		return "gray"
	default:
		return "gray"
	}
}
func statusBadge(status string) string {
	switch statusColor(status) {
	case "red":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F07070")).Render("! OPEN")
	case "green":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#67C58F")).Render("✓ NOT A FINDING")
	case "black":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(lipgloss.Color("#B8B8B8")).Render("⊘ NOT APPLICABLE")
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0")).Render("○ NOT REVIEWED")
	}
}
func (m model) pageSize() int { return max(1, min(10, m.height-19)) }
func (m model) selectedRefs() []match {
	visible := m.visible()
	if len(visible) == 0 {
		return nil
	}
	out := []match{}
	for _, ref := range m.groups[visible[m.cursor]].matches {
		if ref.selected {
			out = append(out, ref)
		}
	}
	return out
}
func (m model) selectedCount() int { return len(m.selectedRefs()) }
func (m *model) moveHost(k string, count int) {
	if count == 0 {
		m.host = 0
		return
	}
	switch k {
	case "pgdown":
		m.host = min(count-1, m.host+m.pageSize())
	case "pgup":
		m.host = max(0, m.host-m.pageSize())
	case "home":
		m.host = 0
	case "end":
		m.host = count - 1
	}
}
func (m model) frame(lines []string) string {
	w := max(40, m.width)
	inner := w - 4
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, inner, "…")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#718096")).Width(w - 2).Render(strings.Join(lines, "\n"))
}
func (m model) separator() string { return strings.Repeat("─", max(36, m.width-4)) }
func (m model) render() string {
	visible := m.visible()
	schemaMode := "OFF"
	if m.schema != nil {
		schemaMode = "ON"
	}
	lines := []string{fmt.Sprintf("CHECKMATE   %d checklist(s)  ·  %d control groups   Schema: %s", len(m.files), len(m.groups), schemaMode), fmt.Sprintf("Status: %s  Severity: %s  Search: %s  Host: %s  %d matches", filters[m.filter], severities[m.severityFilter], safe(m.query), safe(m.hostQuery), len(visible)), m.separator()}
	if len(visible) == 0 {
		lines = append(lines, "No matching controls.", m.separator(), "/ control search  h host search  f status  F severity  q quit")
		if m.stage == hostSearch {
			lines = append(lines, "Host search: "+safe(m.hostQuery)+"_  Enter accept  Esc cancel")
		}
		if m.stage == search {
			lines = append(lines, "Search: "+safe(m.query)+"_  Enter accept  Esc cancel")
		}
		return m.frame(lines)
	}
	g := m.groups[visible[m.cursor]]
	r := g.sample
	if m.stage == details {
		return m.renderDetails(g)
	}
	if m.stage == confirm {
		return m.renderConfirm(g)
	}
	if m.stage == guard {
		return m.renderGuard(g)
	}
	if m.stage == preview {
		return m.renderPreview(g)
	}
	lines = append(lines, fmt.Sprintf("[%d/%d] %s  %s  severity: %s", m.cursor+1, len(visible), safe(r.Benchmark), safe(r.ID), strings.ToUpper(safe(r.Severity))))
	lines = append(lines, fmt.Sprintf("Rule: %s   Release: %s", safe(r.RuleID), safe(r.Release)))
	indices := m.hostIndices(g)
	lines = append(lines, "Summary: "+safe(r.Title), m.separator(), fmt.Sprintf("Affected checklists   %d selected of %d   (Tab focus, Space select)", m.selectedCount(), len(g.matches)))
	size := m.pageSize()
	start := (m.host / size) * size
	end := min(len(indices), start+size)
	for i := start; i < end; i++ {
		ref := g.matches[indices[i]]
		row := m.files[ref.file].Rules[ref.rule]
		selected := " "
		if ref.selected {
			selected = "x"
		}
		focus := " "
		if i == m.host {
			focus = ">"
		}
		host := safe(filepath.Base(filepath.Dir(m.files[ref.file].Path)))
		lines = append(lines, fmt.Sprintf("%s [%s] %-20s  %-18s  %s", focus, selected, clip(host, 20), statusBadge(row.Status), safe(m.files[ref.file].Path)))
	}
	lines = append(lines, fmt.Sprintf("Hosts %d-%d of %d   PgUp/PgDn pages   Home/End jumps", start+1, end, len(indices)), m.separator())
	focused := g.matches[indices[m.host]]
	current := m.files[focused.file].Rules[focused.rule]
	lines = append(lines, "Focused: "+safe(m.files[focused.file].Path), "Status: "+statusBadge(current.Status)+"   Comment: "+safe(current.Comment), "Finding: "+safe(current.Findings), m.separator())
	switch m.stage {
	case browse:
		lines = append(lines, "j/k controls  Tab host  PgUp/PgDn hosts  Space select  m matching  A visible  c clear  v details  e edit  f/F filters  /,h search  q quit")
	case search:
		lines = append(lines, "Search: "+safe(m.query)+"_   Enter accept  Esc cancel")
	case hostSearch:
		lines = append(lines, "Host search: "+safe(m.hostQuery)+"_  Enter accept  Esc cancel")
	case choose:
		lines = append(lines, "State: 1 Open   2 Not a Finding   3 Not Applicable   4 Not Reviewed   Esc cancel")
	case comment:
		lines = append(lines, "New state: "+statusBadge(m.target)+"   Comment (Ctrl+J newline, Enter review, Esc cancel):")
		wrapped := wrapLines(m.note, max(1, m.width-7))
		start := max(0, len(wrapped)-3)
		lines = append(lines, wrapped[start:]...)
		lines = append(lines, "_")
	}
	if m.message != "" {
		lines = append(lines, "Notice: "+safe(m.message))
	}
	return m.frame(lines)
}
func (m model) renderConfirm(g group) string {
	selected := m.selectedRefs()
	size := m.pageSize()
	start := 0
	if len(selected) > 0 {
		start = (min(m.host, len(selected)-1) / size) * size
	}
	end := min(len(selected), start+size)
	counts := map[string]int{}
	for _, ref := range selected {
		counts[m.files[ref.file].Rules[ref.rule].Status]++
	}
	lines := []string{"CHECKMATE   Review changes", fmt.Sprintf("%s  %s   severity: %s", safe(g.sample.Benchmark), safe(g.sample.ID), strings.ToUpper(safe(g.sample.Severity))), m.separator(), "New state: " + statusBadge(m.target), "New comment: " + safe(strings.ReplaceAll(m.note, "\n", " ↵ ")), fmt.Sprintf("Selected old states: Open %d · Not a Finding %d · Not Applicable %d · Not Reviewed %d", counts["open"], counts["not_a_finding"], counts["not_applicable"], counts["not_reviewed"]), m.separator(), fmt.Sprintf("Affected checklists   %d selected of %d", len(selected), len(g.matches))}
	for i := start; i < end; i++ {
		ref := selected[i]
		path := m.files[ref.file].Path
		old := m.files[ref.file].Rules[ref.rule]
		focus := " "
		if i == m.host {
			focus = ">"
		}
		lines = append(lines, fmt.Sprintf("%s UPDATE %-15s %s -> %s", focus, safe(filepath.Base(filepath.Dir(path))), statusBadge(old.Status), statusBadge(m.target)))
		lines = append(lines, fmt.Sprintf("  Comment: %s -> %s", safe(old.Comment), safe(strings.ReplaceAll(m.note, "\n", " ↵ "))))
	}
	if len(selected) > 0 {
		lines = append(lines, fmt.Sprintf("Selected hosts %d-%d of %d   PgUp/PgDn pages   Home/End jumps", start+1, end, len(selected)))
	} else {
		lines = append(lines, "No hosts selected; no files will change.")
	}
	lines = append(lines, m.separator(), "Existing CKLBs replaced; provenance sidecars remain stale.", "PgUp/PgDn hosts   v full per-host preview   Y/Enter apply   N/Esc cancel")
	return m.frame(lines)
}
func (m model) renderPreview(g group) string {
	selected := m.selectedRefs()
	if len(selected) == 0 {
		return m.renderConfirm(g)
	}
	ref := selected[min(m.host, len(selected)-1)]
	old := m.files[ref.file].Rules[ref.rule]
	content := []string{"Host: " + safe(m.files[ref.file].Path), "Old state: " + statusBadge(old.Status), "Old comment:"}
	content = append(content, wrapLines(old.Comment, max(1, m.width-6))...)
	content = append(content, "", "New state: "+statusBadge(m.target), "New comment:")
	content = append(content, wrapLines(m.note, max(1, m.width-6))...)
	size := max(1, m.height-10)
	start := min(m.detailOffset, max(0, len(content)-1))
	end := min(len(content), start+size)
	lines := []string{fmt.Sprintf("CHECKMATE   Full change preview  host %d/%d", min(m.host, len(selected)-1)+1, len(selected)), m.separator()}
	lines = append(lines, content[start:end]...)
	lines = append(lines, m.separator(), fmt.Sprintf("Lines %d-%d of %d   j/k scroll   PgUp/PgDn pages   Tab next host   Esc return", start+1, end, len(content)))
	return m.frame(lines)
}
func (m model) renderGuard(g group) string {
	lines := []string{"CHECKMATE   Additional authorization", m.separator(), fmt.Sprintf("%s  %s  %d selected hosts", safe(g.sample.Benchmark), safe(g.sample.ID), m.selectedCount()), "Bulk Open -> Not a Finding changes require individual evidence review.", "No files have been written yet.", m.separator(), "Type OVERRIDE and press Enter to apply:", safe(m.ack) + "_", "Esc: return to change preview"}
	if m.message != "" {
		lines = append(lines, "Notice: "+safe(m.message))
	}
	return m.frame(lines)
}
func wrapLines(s string, width int) []string {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		r := []rune(safe(line))
		if len(r) == 0 {
			out = append(out, "")
			continue
		}
		for len(r) > 0 {
			take := min(len(r), max(1, width))
			out = append(out, string(r[:take]))
			r = r[take:]
		}
	}
	return out
}
func (m model) renderDetails(g group) string {
	ref := g.matches[m.hostIndices(g)[m.host]]
	row := m.files[ref.file].Rules[ref.rule]
	content := append([]string{"Full current comment:"}, wrapLines(row.Comment, max(1, m.width-6))...)
	content = append(content, "", "Full finding details:")
	content = append(content, wrapLines(row.Findings, max(1, m.width-6))...)
	size := max(1, m.height-11)
	start := min(m.detailOffset, max(0, len(content)-1))
	end := min(len(content), start+size)
	lines := []string{"CHECKMATE   Full finding details", fmt.Sprintf("%s  %s  %s", safe(row.Benchmark), safe(row.ID), statusBadge(row.Status)), "Host: " + safe(m.files[ref.file].Path), m.separator()}
	lines = append(lines, content[start:end]...)
	lines = append(lines, fmt.Sprintf("Details lines %d-%d of %d", start+1, end, len(content)), m.separator(), "j/k scroll  PgUp/PgDn page  Home first  Esc/v return")
	return m.frame(lines)
}
