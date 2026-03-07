package context

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"atrakta/internal/util"
)

type conventionSection struct {
	Heading string
	Text    string
	Score   int
	Index   int
}

func buildConventionSnippet(relPath, raw string, tokenBudget int) (string, int) {
	if tokenBudget <= 0 {
		return "", 0
	}
	maxChars := tokenBudget * 4
	text := strings.TrimSpace(util.NormalizeContentLF(raw))
	if text == "" {
		return "", 0
	}
	sections := splitConventionSections(text)
	indexPart := renderConventionIndex(relPath, sections)
	if len(indexPart) > maxChars {
		clipped := indexPart[:maxChars]
		return clipped, estimateTokens(clipped)
	}
	var b strings.Builder
	b.WriteString(indexPart)
	remaining := maxChars - b.Len()
	if remaining <= 0 {
		out := b.String()
		return out, estimateTokens(out)
	}
	if len(text) <= remaining {
		b.WriteString(text)
		out := b.String()
		return out, estimateTokens(out)
	}
	chosen := chooseSections(sections, remaining)
	if len(chosen) == 0 {
		head := text
		if len(head) > remaining {
			head = head[:remaining]
		}
		b.WriteString(head)
		out := b.String()
		return out, estimateTokens(out)
	}
	for _, s := range chosen {
		chunk := strings.TrimSpace(s.Text) + "\n\n"
		if len(chunk) > remaining {
			chunk = chunk[:remaining]
		}
		b.WriteString(chunk)
		remaining = maxChars - b.Len()
		if remaining <= 0 {
			break
		}
	}
	out := b.String()
	return out, estimateTokens(out)
}

func splitConventionSections(text string) []conventionSection {
	lines := strings.Split(text, "\n")
	sections := make([]conventionSection, 0, 16)
	var cur *conventionSection
	flush := func() {
		if cur == nil {
			return
		}
		cur.Text = strings.TrimSpace(cur.Text)
		cur.Score = conventionPriority(cur.Heading, cur.Text)
		sections = append(sections, *cur)
		cur = nil
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "#") {
			flush()
			cur = &conventionSection{Heading: trim, Text: trim + "\n", Index: len(sections)}
			continue
		}
		if cur == nil {
			cur = &conventionSection{Heading: "(intro)", Text: "", Index: len(sections)}
		}
		cur.Text += line + "\n"
	}
	flush()
	return sections
}

func renderConventionIndex(relPath string, sections []conventionSection) string {
	var b strings.Builder
	b.WriteString("<!-- conventions:index ")
	b.WriteString(relPath)
	b.WriteString(" -->\n")
	limit := len(sections)
	if limit > 24 {
		limit = 24
	}
	for i := 0; i < limit; i++ {
		s := strings.TrimSpace(sections[i].Heading)
		if s == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

func chooseSections(sections []conventionSection, maxChars int) []conventionSection {
	if len(sections) == 0 || maxChars <= 0 {
		return nil
	}
	cands := append([]conventionSection(nil), sections...)
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].Score != cands[j].Score {
			return cands[i].Score > cands[j].Score
		}
		return cands[i].Index < cands[j].Index
	})
	selectedIdx := make(map[int]struct{}, len(cands))
	used := 0
	for _, c := range cands {
		chunk := len(strings.TrimSpace(c.Text)) + 2
		if chunk <= 0 {
			continue
		}
		if used+chunk > maxChars {
			continue
		}
		selectedIdx[c.Index] = struct{}{}
		used += chunk
	}
	if len(selectedIdx) == 0 {
		return nil
	}
	out := make([]conventionSection, 0, len(selectedIdx))
	for _, s := range sections {
		if _, ok := selectedIdx[s.Index]; ok {
			out = append(out, s)
		}
	}
	return out
}

func conventionPriority(heading, body string) int {
	h := strings.ToLower(heading + "\n" + body)
	score := 1
	keywords := []string{
		"must", "never", "required", "requirement", "security", "safety", "critical",
		"禁止", "必須", "安全", "必ず", "してはいけない",
	}
	for _, k := range keywords {
		if strings.Contains(h, k) {
			score += 3
		}
	}
	if strings.Contains(h, "example") || strings.Contains(h, "例") {
		score++
	}
	return score
}

func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	n := (len(text) + 3) / 4
	if n < 1 {
		return 1
	}
	return n
}

func conventionsTokenBudget() int {
	raw := strings.TrimSpace(os.Getenv("ATRAKTA_CONVENTIONS_MAX_TOKENS"))
	if raw == "" {
		return 600
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 600
	}
	if n > 4000 {
		return 4000
	}
	return n
}
