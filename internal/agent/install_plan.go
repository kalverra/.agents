package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/huh/v2"
	"github.com/mattn/go-isatty"

	"github.com/kalverra/agents/internal/output"
)

// Hook script basenames deployed to HooksDeployDir (must match deployHookScripts).
func hookScriptNames() []string {
	return []string{"rtk-prepend.sh", "claude-rtk.sh", "gemini-rtk.sh", "cursor-rtk.sh"}
}

// InstallPlanReport is a JSON-serializable snapshot of the install plan (e.g. dry-run + --ai-output).
type InstallPlanReport struct {
	HookScripts []hookScriptEntry   `json:"hook_scripts,omitempty"`
	Markdown    []markdownPlanEntry `json:"markdown,omitempty"`
	HookMerges  []string            `json:"hook_merges,omitempty"`
	Skills      []skillDestPlan     `json:"skills,omitempty"`
	SkillsNotes []string            `json:"skills_notes,omitempty"`
	LocalMerged string              `json:"local_merged"`
}

// installSelection captures which install branches run (mirrors installClaude / installGemini / installCursor / installCodex).
type installSelection struct {
	DidClaude           bool
	DidGemini           bool
	DidAntigravity      bool
	DidCursor           bool
	DidCodex            bool
	StripGeminiMarkdown bool
	GeminiHooksMerge    bool
}

func (s installSelection) anyAgent() bool {
	return s.DidClaude || s.DidGemini || s.DidCursor || s.DidCodex
}

func (s installSelection) hookScriptsNeeded() bool {
	return s.DidClaude || s.GeminiHooksMerge || s.DidCursor
}

// computeSelection derives install flags from detection and --targets without side effects.
func computeSelection(inst *Installer, detected map[Agent]bool, forcing bool) installSelection {
	var s installSelection

	if TargetWanted(Claude, inst.Targets) && (detected[Claude] || forcing) {
		s.DidClaude = true
	}

	geminiWanted := TargetWanted(Gemini, inst.Targets) || TargetWanted(Antigravity, inst.Targets)
	geminiDetected := detected[Gemini] || detected[Antigravity]
	if geminiWanted && (geminiDetected || forcing) {
		s.DidGemini = true
		antigravityForced := forcing && Contains(inst.Targets, Antigravity)
		needsMarkdownHooks := detected[Antigravity] || antigravityForced
		s.StripGeminiMarkdown = inst.WithHooks && !needsMarkdownHooks
		s.DidAntigravity = detected[Antigravity] || antigravityForced

		geminiForced := forcing && Contains(inst.Targets, Gemini)
		s.GeminiHooksMerge = inst.WithHooks && (detected[Gemini] || geminiForced)
	}

	if TargetWanted(Cursor, inst.Targets) && (detected[Cursor] || forcing) {
		s.DidCursor = true
	}

	if TargetWanted(Codex, inst.Targets) && (detected[Codex] || forcing) {
		s.DidCodex = true
	}

	return s
}

// InstallPlan describes filesystem actions before any writes.
type InstallPlan struct {
	HookScripts []hookScriptEntry
	Markdown    []markdownPlanEntry
	HookMerges  []string
	Skills      []skillDestPlan
	SkillsNotes []string
	LocalMerged string
}

type hookScriptEntry struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

type markdownPlanEntry struct {
	Agent   string `json:"agent"`
	Dest    string `json:"dest"`
	Summary string `json:"summary"`
}

type skillDestPlan struct {
	Label     string   `json:"label"`
	DestRoot  string   `json:"dest_root"`
	Removed   []string `json:"removed"`
	Installed []string `json:"installed"`
}

// buildInstallPlan gathers paths and skill diffs for display and JSON.
func (inst *Installer) buildInstallPlan(sel installSelection, skillDirs []string) (*InstallPlan, error) {
	plan := &InstallPlan{
		LocalMerged: filepath.Join(inst.RepoRoot, "GLOBAL_AGENTS.local.md"),
	}

	if inst.WithHooks && sel.hookScriptsNeeded() {
		hd := HooksDeployDir()
		srcDir := filepath.Join(inst.RepoRoot, "hooks")
		for _, name := range hookScriptNames() {
			plan.HookScripts = append(plan.HookScripts, hookScriptEntry{
				Src:  filepath.Join(srcDir, name),
				Dest: filepath.Join(hd, name),
			})
		}
	}

	if sel.DidClaude {
		note := "full merge + hook sections"
		if !inst.WithHooks {
			note = "merge USER_AGENTS; strip hook delimiter lines only"
		}
		plan.Markdown = append(plan.Markdown, markdownPlanEntry{
			Agent:   "Claude",
			Dest:    MarkdownDest(Claude),
			Summary: note,
		})
		if inst.WithHooks {
			plan.HookMerges = append(plan.HookMerges, HookSettingsPath(Claude))
		}
	}

	if sel.DidGemini {
		note := "merge USER_AGENTS"
		if sel.StripGeminiMarkdown {
			note = "merge USER_AGENTS; strip hookable sections (Gemini CLI)"
		} else if inst.WithHooks {
			note = "merge USER_AGENTS; keep hook sections (Antigravity)"
		}
		plan.Markdown = append(plan.Markdown, markdownPlanEntry{
			Agent:   "Gemini / Antigravity",
			Dest:    MarkdownDest(Gemini),
			Summary: note,
		})
		if sel.GeminiHooksMerge {
			plan.HookMerges = append(plan.HookMerges, HookSettingsPath(Gemini))
		}
	}

	if sel.DidCursor {
		plan.Markdown = append(plan.Markdown, markdownPlanEntry{
			Agent:   "Cursor",
			Dest:    MarkdownDest(Cursor),
			Summary: "frontmatter + merged GLOBAL_AGENTS.md",
		})
		if inst.WithHooks {
			hpath := HookSettingsPath(Cursor)
			extra := filepath.Join(inst.RepoRoot, "hooks", "cursor-session-start-snippet.json")
			if _, err := os.Stat(extra); err == nil {
				plan.HookMerges = append(plan.HookMerges, hpath+" (includes session-start snippet)")
			} else {
				plan.HookMerges = append(plan.HookMerges, hpath)
			}
		}
	}

	if sel.DidCodex {
		plan.Markdown = append(plan.Markdown, markdownPlanEntry{
			Agent:   "Codex",
			Dest:    MarkdownDest(Codex),
			Summary: "merge USER_AGENTS; keep hookable instructions (Codex hooks not installed)",
		})
	}

	if inst.WithSkills {
		if sel.DidClaude {
			sp, err := inst.skillDestPlan("Claude", SkillsDest(Claude), skillDirs)
			if err != nil {
				return nil, err
			}
			plan.Skills = append(plan.Skills, sp)
		}
		if sel.DidCursor {
			sp, err := inst.skillDestPlan("Cursor", SkillsDest(Cursor), skillDirs)
			if err != nil {
				return nil, err
			}
			plan.Skills = append(plan.Skills, sp)
		}
		if sel.DidGemini {
			plan.SkillsNotes = append(plan.SkillsNotes,
				fmt.Sprintf(
					"Gemini CLI: skip %s — gemini-cli loads universal skills from %s by default; copying there would duplicate skills and conflict",
					SkillsDest(Gemini),
					filepath.Join(inst.RepoRoot, "skills"),
				),
			)
			if sel.DidAntigravity {
				spA, err := inst.skillDestPlan("Antigravity", SkillsDest(Antigravity), skillDirs)
				if err != nil {
					return nil, err
				}
				plan.Skills = append(plan.Skills, spA)
			}
		}
		if sel.DidCodex {
			plan.SkillsNotes = append(plan.SkillsNotes,
				fmt.Sprintf(
					"Codex: skip %s — Codex loads user skills from %s by default; copying there would duplicate skills and conflict",
					SkillsDest(Codex),
					filepath.Join(inst.RepoRoot, "skills"),
				),
			)
		}
	}

	return plan, nil
}

func skillBasenames(skillDirs []string) []string {
	var names []string
	for _, d := range skillDirs {
		names = append(names, filepath.Base(d))
	}
	sort.Strings(names)
	return names
}

// skillDestPlan compares on-disk skill dir children to repo skill names.
func (inst *Installer) skillDestPlan(label, destRoot string, skillDirs []string) (skillDestPlan, error) {
	repoNames := skillBasenames(skillDirs)
	p := skillDestPlan{
		Label:     label,
		DestRoot:  destRoot,
		Installed: append([]string(nil), repoNames...),
	}
	repoSet := make(map[string]struct{}, len(repoNames))
	for _, n := range repoNames {
		repoSet[n] = struct{}{}
	}

	entries, err := os.ReadDir(destRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return skillDestPlan{}, err
	}
	for _, e := range entries {
		name := e.Name()
		if _, ok := repoSet[name]; !ok {
			p.Removed = append(p.Removed, name)
		}
	}
	sort.Strings(p.Removed)
	return p, nil
}

// formatInstallPlan returns the same human-readable summary as printInstallPlan (without JSON suppression).
func formatInstallPlan(plan *InstallPlan, withHooks, withSkills bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s", strings.Repeat("—", 60))

	if withHooks && len(plan.HookScripts) > 0 {
		fmt.Fprintf(&b, "\n\nHook scripts → %s\n", HooksDeployDir())
		for _, h := range plan.HookScripts {
			fmt.Fprintf(&b, "  • %s\n", filepath.Base(h.Dest))
		}
	}

	if len(plan.Markdown) > 0 {
		b.WriteString("\n\nMarkdown\n")
		for _, m := range plan.Markdown {
			fmt.Fprintf(&b, "  • [%s] %s\n", m.Agent, m.Dest)
			fmt.Fprintf(&b, "      %s\n", m.Summary)
		}
	}

	if withHooks && len(plan.HookMerges) > 0 {
		b.WriteString("\n\nMerge hooks into settings\n")
		for _, p := range plan.HookMerges {
			fmt.Fprintf(&b, "  • %s\n", p)
		}
	}

	if withSkills && len(plan.SkillsNotes) > 0 {
		b.WriteString("\n\nSkills (notes)\n")
		for _, n := range plan.SkillsNotes {
			fmt.Fprintf(&b, "  • %s\n", n)
		}
	}

	if withSkills && len(plan.Skills) > 0 {
		b.WriteString("\n\nSkills\n")
		for _, s := range plan.Skills {
			fmt.Fprintf(&b, "  • [%s] %s\n", s.Label, s.DestRoot)
			if len(s.Removed) > 0 {
				fmt.Fprintf(&b, "      remove: %s\n", strings.Join(s.Removed, ", "))
			} else {
				b.WriteString("      remove: (none)\n")
			}
			if len(s.Installed) > 0 {
				fmt.Fprintf(&b, "      install: %s\n", strings.Join(s.Installed, ", "))
			} else {
				b.WriteString("      install: (none — destination will be empty)\n")
			}
		}
	}

	fmt.Fprintf(&b, "\nLocal merged file: %s\n", plan.LocalMerged)
	b.WriteString(strings.Repeat("—", 60))
	b.WriteString("\n\n")
	return b.String()
}

// printInstallPlan renders a human-readable summary (respects output.JSON suppression).
func printInstallPlan(plan *InstallPlan, withHooks, withSkills bool) {
	// Trim one trailing newline so strings.Split matches output.Println line boundaries (incl. final blank line).
	text := strings.TrimSuffix(formatInstallPlan(plan, withHooks, withSkills), "\n")
	for line := range strings.SplitSeq(text, "\n") {
		output.Println(line)
	}
}

// confirmInstall blocks until the user approves, unless --yes or non-interactive rules apply.
func (inst *Installer) confirmInstall(plan *InstallPlan, withHooks, withSkills bool) error {
	if inst.Yes {
		printInstallPlan(plan, withHooks, withSkills)
		return nil
	}
	if output.JSON() || !isatty.IsTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("refusing to install without --yes (non-interactive terminal or --ai-output)")
	}

	printInstallPlan(plan, withHooks, withSkills)

	var proceed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with install?").
				Affirmative("Yes").
				Negative("No").
				Value(&proceed),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCharm)).
		WithAccessible(os.Getenv("ACCESSIBLE") != "")

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return fmt.Errorf("install cancelled")
		}
		return fmt.Errorf("install: %w", err)
	}
	if !proceed {
		return fmt.Errorf("install cancelled")
	}
	return nil
}

// planToReport converts InstallPlan to InstallPlanReport for JSON output.
func planToReport(p *InstallPlan) *InstallPlanReport {
	if p == nil {
		return nil
	}
	return &InstallPlanReport{
		HookScripts: append([]hookScriptEntry(nil), p.HookScripts...),
		Markdown:    append([]markdownPlanEntry(nil), p.Markdown...),
		HookMerges:  append([]string(nil), p.HookMerges...),
		Skills:      append([]skillDestPlan(nil), p.Skills...),
		SkillsNotes: append([]string(nil), p.SkillsNotes...),
		LocalMerged: p.LocalMerged,
	}
}
