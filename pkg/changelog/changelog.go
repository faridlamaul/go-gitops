package changelog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// conventionalCommit captures "type(scope)!: subject" from commit titles.
	conventionalCommit = regexp.MustCompile(`^([a-zA-Z]+)(?:\(([^)]*)\))?(!)?:\s*(.+)$`)
	// githubPRRef captures the "(#123)" PR reference GitHub appends to squash commits.
	githubPRRef = regexp.MustCompile(`\s*\(#(\d+)\)\s*$`)
	breakingBody = regexp.MustCompile(`(?m)^BREAKING(?: |-)CHANGE:\s*(.+)$`)
)

// Section definitions for release notes.
type releaseSection struct {
	Header string
	Types  []string
}

var releaseSections = []releaseSection{
	{Header: "💥 Breaking Changes", Types: []string{"breaking"}},
	{Header: "🚀 Features", Types: []string{"feat"}},
	{Header: "🐛 Bug Fixes", Types: []string{"fix"}},
	{Header: "⚡ Performance Improvements", Types: []string{"perf"}},
	{Header: "♻️ Refactoring", Types: []string{"refactor"}},
	{Header: "📝 Documentation", Types: []string{"docs"}},
	{Header: "🔧 Build System & CI", Types: []string{"build", "ci"}},
	{Header: "🧹 Chores & Maintenance", Types: []string{"chore", "style"}},
}

const otherSection = "📦 Other Changes"

var sectionByType = func() map[string]string {
	m := make(map[string]string)
	for _, sec := range releaseSections {
		for _, t := range sec.Types {
			m[t] = sec.Header
		}
	}
	return m
}()

// CommitInfo represents a git commit with relevant metadata.
type CommitInfo struct {
	SHA         string
	Title       string
	Body        string
	AuthorName  string
	AuthorLogin string
}

// ParsedCommit is a commit decomposed into Conventional Commit fields.
type ParsedCommit struct {
	Type       string
	Scope      string
	Subject    string
	IsBreaking bool
	PRNumber   string
}

// ParseCommit decomposes a commit into Conventional Commit parts.
func ParseCommit(c CommitInfo) ParsedCommit {
	title := strings.TrimSpace(c.Title)
	var prNumber string

	// Extract GitHub PR number e.g. "(#123)"
	if loc := githubPRRef.FindStringSubmatchIndex(title); loc != nil {
		prNumber = title[loc[2]:loc[3]]
		title = strings.TrimSpace(title[:loc[0]])
	}

	isBreaking := false
	if breakingBody.MatchString(c.Body) || breakingBody.MatchString(c.Title) {
		isBreaking = true
	}

	m := conventionalCommit.FindStringSubmatch(title)
	if m == nil {
		return ParsedCommit{
			Type:       "other",
			Subject:    title,
			IsBreaking: isBreaking,
			PRNumber:   prNumber,
		}
	}

	cType := strings.ToLower(m[1])
	scope := m[2]
	if m[3] == "!" {
		isBreaking = true
	}
	subject := strings.TrimSpace(m[4])

	return ParsedCommit{
		Type:       cType,
		Scope:      scope,
		Subject:    subject,
		IsBreaking: isBreaking,
		PRNumber:   prNumber,
	}
}

// ReleaseMeta carries context needed to render GitHub release notes.
type ReleaseMeta struct {
	RepoOwner   string
	RepoName    string
	PreviousTag string
	NewTag      string
	Branch      string
	Actor       string
}

// GenerateReleaseNotes renders GitHub release notes grouped by Conventional Commit categories.
func GenerateReleaseNotes(commits []CommitInfo, meta ReleaseMeta) (string, bool, bool) {
	grouped := make(map[string][]string)
	hasBreaking := false
	hasFeatures := false
	contributors := make(map[string]bool)

	for _, c := range commits {
		parsed := ParseCommit(c)
		if parsed.IsBreaking {
			hasBreaking = true
		}
		if parsed.Type == "feat" {
			hasFeatures = true
		}
		if c.AuthorLogin != "" {
			contributors["@"+c.AuthorLogin] = true
		} else if c.AuthorName != "" {
			contributors[c.AuthorName] = true
		}

		line := formatCommitLine(c, parsed, meta.RepoOwner, meta.RepoName)

		if parsed.IsBreaking {
			grouped["💥 Breaking Changes"] = append(grouped["💥 Breaking Changes"], line)
		}

		secHeader, ok := sectionByType[parsed.Type]
		if !ok {
			secHeader = otherSection
		}
		grouped[secHeader] = append(grouped[secHeader], line)
	}

	var b strings.Builder

	// 1. One-glance Summary Header
	b.WriteString(fmt.Sprintf("## %s\n\n", meta.NewTag))
	if meta.PreviousTag != "" {
		compareURL := fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", meta.RepoOwner, meta.RepoName, meta.PreviousTag, meta.NewTag)
		b.WriteString(fmt.Sprintf("🔍 **Full Changelog**: [%s...%s](%s)\n\n", meta.PreviousTag, meta.NewTag, compareURL))
	}

	b.WriteString(fmt.Sprintf("> 📊 **Metrics:** %d commit(s)", len(commits)))
	if hasBreaking {
		b.WriteString(" • 💥 Breaking Changes Included")
	}
	b.WriteString(fmt.Sprintf(" • 👥 %d contributor(s)\n\n", len(contributors)))

	// 2. Ordered Sections
	for _, sec := range releaseSections {
		items := grouped[sec.Header]
		if len(items) > 0 {
			b.WriteString(fmt.Sprintf("### %s\n\n", sec.Header))
			for _, item := range items {
				b.WriteString(item + "\n")
			}
			b.WriteString("\n")
		}
	}

	// 3. Other Changes
	if items := grouped[otherSection]; len(items) > 0 {
		b.WriteString(fmt.Sprintf("### %s\n\n", otherSection))
		for _, item := range items {
			b.WriteString(item + "\n")
		}
		b.WriteString("\n")
	}

	// 4. Contributors acknowledgment
	if len(contributors) > 0 {
		var authors []string
		for author := range contributors {
			authors = append(authors, author)
		}
		sort.Strings(authors)
		b.WriteString(fmt.Sprintf("❤️ **Thanks to all contributors:** %s\n", strings.Join(authors, ", ")))
	}

	return b.String(), hasBreaking, hasFeatures
}

func formatCommitLine(c CommitInfo, p ParsedCommit, owner, repo string) string {
	shortSHA := c.SHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	var b strings.Builder
	b.WriteString("- ")

	if p.Scope != "" {
		b.WriteString(fmt.Sprintf("**%s:** ", p.Scope))
	}

	b.WriteString(p.Subject)

	if p.PRNumber != "" {
		prURL := fmt.Sprintf("https://github.com/%s/%s/pull/%s", owner, repo, p.PRNumber)
		b.WriteString(fmt.Sprintf(" ([#%s](%s))", p.PRNumber, prURL))
	}

	if c.SHA != "" && owner != "" && repo != "" {
		commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, repo, c.SHA)
		b.WriteString(fmt.Sprintf(" ([`%s`](%s))", shortSHA, commitURL))
	}

	if c.AuthorLogin != "" {
		b.WriteString(fmt.Sprintf(" by @%s", c.AuthorLogin))
	} else if c.AuthorName != "" {
		b.WriteString(fmt.Sprintf(" by %s", c.AuthorName))
	}

	return b.String()
}
