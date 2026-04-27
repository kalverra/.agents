package agent

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/kalverra/agents/internal/markdown"
	"github.com/kalverra/agents/internal/output"
)

// Installer deploys global agents markdown, hooks, and skills.
type Installer struct {
	RepoRoot   string
	DryRun     bool
	Yes        bool
	UseCopy    bool
	Verbose    bool
	WithHooks  bool
	WithSkills bool
	Targets    []Agent // nil means all detected
}

// InstallReport summarizes what was deployed.
type InstallReport struct {
	Agents []string           `json:"agents"`
	Hooks  bool               `json:"hooks"`
	Skills bool               `json:"skills"`
	DryRun bool               `json:"dry_run"`
	Plan   *InstallPlanReport `json:"plan,omitempty"`
}

// Install runs the full deployment for all targeted agents.
func (inst *Installer) Install() (*InstallReport, error) {
	log.Debug().Msg("Starting installation")
	src := filepath.Join(inst.RepoRoot, "GLOBAL_AGENTS.md")
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("missing source file: %s", src)
	}

	detected := DetectAll(inst.Verbose)
	forcing := inst.Targets != nil
	sel := computeSelection(inst, detected, forcing)

	if !sel.anyAgent() {
		output.Println("Nothing installed. Try: agents discover")
		output.Println("Force paths with: agents install --targets claude,gemini,antigravity,cursor,codex")
		return nil, fmt.Errorf("no agents installed")
	}

	skillDirs := inst.repoSkillDirs()
	plan, err := inst.buildInstallPlan(sel, skillDirs)
	if err != nil {
		return nil, err
	}

	report := installReportFromSelection(inst, sel)
	if inst.DryRun {
		printInstallPlan(plan, inst.WithHooks, inst.WithSkills)
		report.DryRun = true
		report.Plan = planToReport(plan)
		return report, nil
	}

	if err := inst.confirmInstall(plan, inst.WithHooks, inst.WithSkills); err != nil {
		return nil, err
	}

	var hooksDir string
	if inst.WithHooks && sel.hookScriptsNeeded() {
		hooksDir, err = inst.deployHookScripts()
		if err != nil {
			return nil, fmt.Errorf("deploying hook scripts: %w", err)
		}
	}

	didClaude, err := inst.installClaude(src, hooksDir, detected, forcing)
	if err != nil {
		return nil, err
	}

	_, didAntigravity, err := inst.installGemini(src, hooksDir, detected, forcing)
	if err != nil {
		return nil, err
	}

	didCursor, err := inst.installCursor(src, hooksDir, detected, forcing)
	if err != nil {
		return nil, err
	}

	didCodex, err := inst.installCodex(src, detected, forcing)
	if err != nil {
		return nil, err
	}

	if err := inst.installSkills(didClaude, didCursor, didAntigravity, didCodex, skillDirs); err != nil {
		return nil, err
	}

	if err := inst.writeLocalMergedGlobal(src); err != nil {
		return nil, err
	}

	return report, nil
}

func installReportFromSelection(inst *Installer, sel installSelection) *InstallReport {
	report := &InstallReport{
		Hooks:  inst.WithHooks && sel.hookScriptsNeeded(),
		Skills: inst.WithSkills,
		DryRun: false,
	}
	if sel.DidClaude {
		report.Agents = append(report.Agents, "claude")
	}
	if sel.DidGemini {
		report.Agents = append(report.Agents, "gemini")
	}
	if sel.DidAntigravity {
		report.Agents = append(report.Agents, "antigravity")
	}
	if sel.DidCursor {
		report.Agents = append(report.Agents, "cursor")
	}
	if sel.DidCodex {
		report.Agents = append(report.Agents, "codex")
	}
	return report
}

func (inst *Installer) installClaude(src, hooksDir string, detected map[Agent]bool, forcing bool) (bool, error) {
	if !TargetWanted(Claude, inst.Targets) {
		return false, nil
	}
	if !detected[Claude] && !forcing {
		return false, nil
	}
	if !detected[Claude] {
		output.Warnf("claude-code not detected; writing %s anyway (--targets).\n", MarkdownDest(Claude))
	}
	if err := inst.deployMarkdown(src, MarkdownDest(Claude), inst.WithHooks); err != nil {
		return false, err
	}
	if inst.WithHooks {
		if err := inst.installHooksForAgent(Claude, hooksDir); err != nil {
			return false, err
		}
	}
	return true, nil
}

// installGemini handles the Gemini/Antigravity installation logic.
// Returns (didInstall, didAntigravity, err).
func (inst *Installer) installGemini(src, hooksDir string, detected map[Agent]bool, forcing bool) (bool, bool, error) {
	geminiWanted := TargetWanted(Gemini, inst.Targets) || TargetWanted(Antigravity, inst.Targets)
	if !geminiWanted {
		return false, false, nil
	}
	geminiDetected := detected[Gemini] || detected[Antigravity]
	if !geminiDetected && !forcing {
		return false, false, nil
	}
	if !geminiDetected {
		output.Warnf("gemini-cli / antigravity not detected; writing %s anyway (--targets).\n",
			MarkdownDest(Gemini))
	}

	antigravityForced := forcing && Contains(inst.Targets, Antigravity)
	needsMarkdownHooks := detected[Antigravity] || antigravityForced
	stripMarkdown := inst.WithHooks && !needsMarkdownHooks

	if err := inst.deployMarkdown(src, MarkdownDest(Gemini), stripMarkdown); err != nil {
		return false, false, err
	}

	geminiForced := forcing && Contains(inst.Targets, Gemini)
	if inst.WithHooks && (detected[Gemini] || geminiForced) {
		if err := inst.installHooksForAgent(Gemini, hooksDir); err != nil {
			return false, false, err
		}
	}

	didAntigravity := detected[Antigravity] || antigravityForced
	return true, didAntigravity, nil
}

func (inst *Installer) installCursor(src, hooksDir string, detected map[Agent]bool, forcing bool) (bool, error) {
	if !TargetWanted(Cursor, inst.Targets) {
		return false, nil
	}
	if !detected[Cursor] && !forcing {
		return false, nil
	}
	if !detected[Cursor] {
		output.Warnf("cursor dirs not detected; writing ~/.cursor/rules/global-agents.mdc anyway (--targets).\n")
	}
	if err := inst.writeCursorMDC(src); err != nil {
		return false, err
	}
	if inst.WithHooks {
		if err := inst.installHooksForAgent(Cursor, hooksDir); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (inst *Installer) installCodex(src string, detected map[Agent]bool, forcing bool) (bool, error) {
	if !TargetWanted(Codex, inst.Targets) {
		return false, nil
	}
	if !detected[Codex] && !forcing {
		return false, nil
	}
	if !detected[Codex] {
		output.Warnf("codex not detected; writing %s anyway (--targets).\n", MarkdownDest(Codex))
	}
	if err := inst.deployMarkdown(src, MarkdownDest(Codex), false); err != nil {
		return false, err
	}
	return true, nil
}

func (inst *Installer) installSkills(
	didClaude,
	didCursor,
	didAntigravity,
	didCodex bool,
	skillDirs []string,
) error {
	if !inst.WithSkills {
		return nil
	}
	if len(skillDirs) == 0 {
		output.Warnf("no skill directories with SKILL.md in repo\n")
	}
	if didClaude {
		if err := inst.copySkills(skillDirs, SkillsDest(Claude)); err != nil {
			return err
		}
	}
	if didCursor {
		if err := inst.copySkills(skillDirs, SkillsDest(Cursor)); err != nil {
			return err
		}
	}
	if didAntigravity {
		if err := inst.copySkills(skillDirs, SkillsDest(Antigravity)); err != nil {
			return err
		}
	}
	if didCodex {
		if err := inst.copySkillsPreservingExtras(skillDirs, SkillsDest(Codex)); err != nil {
			return err
		}
	}
	return nil
}

func (inst *Installer) deployMarkdown(src, dest string, stripHooks bool) error {
	raw, err := os.ReadFile(src) //nolint:gosec // src is a repo-internal path, not user input
	if err != nil {
		return err
	}

	userSrc := filepath.Join(filepath.Dir(src), "USER_AGENTS.md")
	body, err := markdown.MergeUserAgents(string(raw), userSrc)
	if err != nil {
		return err
	}
	hasUserContent := body != string(raw)

	if stripHooks {
		body = markdown.StripHookableSections(body)
	} else {
		body = markdown.StripHookableDelimiterLines(body)
		if body == string(raw) && !hasUserContent {
			return inst.linkOrCopy(src, dest)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	_ = os.Remove(dest)
	//nolint:gosec
	if err := os.WriteFile(
		dest,
		[]byte(body),
		0o600,
	); err != nil {
		return err
	}
	output.Successf("Wrote: %s\n", dest)
	return nil
}

func (inst *Installer) linkOrCopy(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	_ = os.Remove(dest)

	if inst.UseCopy {
		data, err := os.ReadFile(src) //nolint:gosec // src is a repo-internal path, not user input
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil { //nolint:gosec // world-readable config file is intended
			return err
		}
		output.Successf("Copied: %s\n", dest)
		return nil
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	if err := os.Symlink(absSrc, dest); err != nil {
		return err
	}
	output.Successf("Symlinked: %s -> %s\n", dest, absSrc)
	return nil
}

func (inst *Installer) writeCursorMDC(globalPath string) error {
	dest := MarkdownDest(Cursor)
	raw, err := os.ReadFile(globalPath) //nolint:gosec // globalPath is a repo-internal path, not user input
	if err != nil {
		return err
	}

	userSrc := filepath.Join(filepath.Dir(globalPath), "USER_AGENTS.md")
	body, err := markdown.MergeUserAgents(string(raw), userSrc)
	if err != nil {
		return err
	}

	if inst.WithHooks {
		body = markdown.StripHookableSections(body)
	} else {
		body = markdown.StripHookableDelimiterLines(body)
	}

	header := fmt.Sprintf(
		"---\ndescription: Machine-wide context from %s/GLOBAL_AGENTS.md\nalwaysApply: true\n---\n\n",
		inst.RepoRoot,
	)

	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	//nolint:gosec // world-readable config file is intended
	if err := os.WriteFile(
		dest,
		[]byte(header+body),
		0o600,
	); err != nil {
		return err
	}
	output.Successf("Wrote: %s\n", dest)
	return nil
}

func (inst *Installer) deployHookScripts() (string, error) {
	srcDir := filepath.Join(inst.RepoRoot, "hooks")
	destDir := HooksDeployDir()
	scripts := hookScriptNames()

	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", err
	}
	for _, s := range scripts {
		data, err := os.ReadFile(filepath.Join(srcDir, s)) //nolint:gosec // srcDir is a repo-internal path
		if err != nil {
			return "", fmt.Errorf("reading hook script %s: %w", s, err)
		}
		dst := filepath.Join(destDir, s)
		if err := os.WriteFile(dst, data, 0o755); err != nil { //nolint:gosec // hook scripts must be executable
			return "", err
		}
	}
	output.Successf("Deployed hook scripts to %s\n", destDir)
	return destDir, nil
}

func (inst *Installer) installHooksForAgent(a Agent, hooksDir string) error {
	snippetName := HookSnippetFile(a)
	if snippetName == "" {
		return nil
	}

	snippetPath := filepath.Join(inst.RepoRoot, "hooks", snippetName)
	raw, err := os.ReadFile(snippetPath) //nolint:gosec // snippetPath is a repo-internal path
	if err != nil {
		return fmt.Errorf("reading snippet %s: %w", snippetName, err)
	}

	rawStr := strings.ReplaceAll(string(raw), "<HOOKS_DIR>", hooksDir)
	var snippet map[string]any
	if err := json.Unmarshal([]byte(rawStr), &snippet); err != nil {
		return fmt.Errorf("parsing snippet %s: %w", snippetName, err)
	}

	settingsPath := HookSettingsPath(a)
	if err := mergeHooksFromSnippet(settingsPath, snippet); err != nil {
		return err
	}

	if a == Cursor {
		extraPath := filepath.Join(inst.RepoRoot, "hooks", "cursor-session-start-snippet.json")
		extraRaw, err := os.ReadFile(extraPath) //nolint:gosec // extraPath is a repo-internal path
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("reading %s: %w", extraPath, err)
		}
		extraStr := strings.ReplaceAll(string(extraRaw), "<HOOKS_DIR>", hooksDir)
		var extraSnippet map[string]any
		if err := json.Unmarshal([]byte(extraStr), &extraSnippet); err != nil {
			return fmt.Errorf("parsing session start snippet: %w", err)
		}
		if err := mergeHooksFromSnippet(settingsPath, extraSnippet); err != nil {
			return err
		}
	}
	return nil
}

// mergeHooksFromSnippet merges every key under snippet["hooks"] into the target settings file.
func mergeHooksFromSnippet(settingsPath string, snippet map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		return err
	}

	var existing map[string]any
	data, err := os.ReadFile(settingsPath) //nolint:gosec // settingsPath is a known agent config path
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		existing = make(map[string]any)
	} else {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	hooksSection, _ := existing["hooks"].(map[string]any)
	if hooksSection == nil {
		hooksSection = make(map[string]any)
		existing["hooks"] = hooksSection
	}

	snippetHooks, _ := snippet["hooks"].(map[string]any)
	for hookKey, newVal := range snippetHooks {
		newEntries, ok := newVal.([]any)
		if !ok {
			continue
		}
		existingEntries, _ := hooksSection[hookKey].([]any)

		for _, newEntry := range newEntries {
			newID := extractMatchField(newEntry)
			replaced := false
			for i, old := range existingEntries {
				oldID := extractMatchField(old)
				if oldID != "" && oldID == newID {
					existingEntries[i] = newEntry
					replaced = true
					break
				}
			}
			if !replaced {
				existingEntries = append(existingEntries, newEntry)
			}
		}

		hooksSection[hookKey] = existingEntries
	}

	// Cursor expects ~/.cursor/hooks.json to declare schema version (flat hook entries).
	if filepath.Base(settingsPath) == "hooks.json" {
		if _, ok := existing["version"]; !ok {
			existing["version"] = 1
		}
	}

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(
		settingsPath,
		append(out, '\n'),
		0o600,
	); err != nil { //nolint:gosec // world-readable settings file is intended
		return err
	}
	output.Successf("Merged hooks into %s\n", settingsPath)
	return nil
}

// extractMatchField pulls the "command" field from a hook entry for dedup.
func extractMatchField(entry any) string {
	m, _ := entry.(map[string]any)
	if m == nil {
		return ""
	}
	// Check nested "hooks" array first (Claude format)
	if nested, ok := m["hooks"].([]any); ok && len(nested) > 0 {
		if inner, ok := nested[0].(map[string]any); ok {
			if cmd, ok := inner["command"].(string); ok {
				return cmd
			}
		}
	}
	if cmd, ok := m["command"].(string); ok {
		return cmd
	}
	if name, ok := m["name"].(string); ok {
		return name
	}
	return ""
}

func (inst *Installer) repoSkillDirs() []string {
	skillsRoot := filepath.Join(inst.RepoRoot, "skills")
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		skillMD := filepath.Join(skillsRoot, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); err == nil {
			dirs = append(dirs, filepath.Join(skillsRoot, e.Name()))
		}
	}
	return dirs
}

func (inst *Installer) copySkills(skillDirs []string, destRoot string) error {
	if destRoot == "" {
		return nil
	}

	repoNames := make(map[string]struct{}, len(skillDirs))
	for _, d := range skillDirs {
		repoNames[filepath.Base(d)] = struct{}{}
	}

	if err := os.MkdirAll(destRoot, 0o750); err != nil {
		return err
	}

	entries, err := os.ReadDir(destRoot)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if _, keep := repoNames[e.Name()]; !keep {
			if err := os.RemoveAll(filepath.Join(destRoot, e.Name())); err != nil {
				return fmt.Errorf("removing extra skill %q: %w", e.Name(), err)
			}
		}
	}

	if len(skillDirs) == 0 {
		output.Successf("Removed all skills under %s\n", destRoot)
		return nil
	}

	n := 0
	for _, sd := range skillDirs {
		dst := filepath.Join(destRoot, filepath.Base(sd))
		_ = os.RemoveAll(dst)
		if err := copyDir(sd, dst); err != nil {
			return fmt.Errorf("copying skill %s: %w", filepath.Base(sd), err)
		}
		n++
	}
	output.Successf("Copied %d skill(s) to %s\n", n, destRoot)
	return nil
}

func (inst *Installer) copySkillsPreservingExtras(skillDirs []string, destRoot string) error {
	if destRoot == "" {
		return nil
	}

	if err := os.MkdirAll(destRoot, 0o750); err != nil {
		return err
	}

	if len(skillDirs) == 0 {
		output.Successf("No repo skills to copy to %s\n", destRoot)
		return nil
	}

	n := 0
	for _, sd := range skillDirs {
		dst := filepath.Join(destRoot, filepath.Base(sd))
		_ = os.RemoveAll(dst)
		if err := copyDir(sd, dst); err != nil {
			return fmt.Errorf("copying skill %s: %w", filepath.Base(sd), err)
		}
		n++
	}
	output.Successf("Copied %d skill(s) to %s\n", n, destRoot)
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := os.ReadFile(path) //nolint:gosec // path is derived from WalkDir, not user input
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644) //nolint:gosec // world-readable skill files are intended
	})
}

func (inst *Installer) writeLocalMergedGlobal(src string) error {
	raw, err := os.ReadFile(src) //nolint:gosec // src is a repo-internal path, not user input
	if err != nil {
		return err
	}
	userSrc := filepath.Join(filepath.Dir(src), "USER_AGENTS.md")
	body, err := markdown.MergeUserAgents(string(raw), userSrc)
	if err != nil {
		return err
	}

	out := filepath.Join(inst.RepoRoot, "GLOBAL_AGENTS.local.md")
	banner := "<!-- Generated by `agents install`. Merged USER_AGENTS.md; gitignored. " +
		"Use @GLOBAL_AGENTS.local.md in Cursor for combined context. -->\n\n"
	//nolint:gosec // world-readable local config is intended
	if err := os.WriteFile(
		out,
		[]byte(banner+body),
		0o644,
	); err != nil {
		return err
	}
	output.Successf("Wrote: %s\n", out)
	return nil
}
