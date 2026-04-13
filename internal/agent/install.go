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
	"github.com/kalverra/agents/internal/ui"
)

// Installer deploys global agents markdown, hooks, and skills.
type Installer struct {
	RepoRoot   string
	DryRun     bool
	UseCopy    bool
	Verbose    bool
	WithHooks  bool
	WithSkills bool
	Targets    []Agent // nil means all detected
	AIOutput   bool
}

// Install runs the full deployment for all targeted agents.
func (inst *Installer) Install() error {
	log.Debug().Msg("Starting installation")
	src := filepath.Join(inst.RepoRoot, "GLOBAL_AGENTS.md")
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("missing source file: %s", src)
	}

	var hooksDir string
	if inst.WithHooks {
		var err error
		hooksDir, err = inst.deployHookScripts()
		if err != nil {
			return fmt.Errorf("deploying hook scripts: %w", err)
		}
	}

	detected := DetectAll(inst.Verbose)
	forcing := inst.Targets != nil

	didClaude, err := inst.installClaude(src, hooksDir, detected, forcing)
	if err != nil {
		return err
	}

	didGemini, didAntigravity, err := inst.installGemini(src, hooksDir, detected, forcing)
	if err != nil {
		return err
	}

	didCursor, err := inst.installCursor(src, hooksDir, detected, forcing)
	if err != nil {
		return err
	}

	if !didClaude && !didGemini && !didCursor {
		ui.Println("Nothing installed. Try: agents discover")
		ui.Println("Force paths with: agents install --targets claude,gemini,antigravity,cursor")
		return fmt.Errorf("no agents installed")
	}

	if err := inst.installSkills(didClaude, didCursor, didAntigravity); err != nil {
		return err
	}

	return inst.writeLocalMergedGlobal(src)
}

func (inst *Installer) installClaude(src, hooksDir string, detected map[Agent]bool, forcing bool) (bool, error) {
	if !TargetWanted(Claude, inst.Targets) {
		return false, nil
	}
	if !detected[Claude] && !forcing {
		return false, nil
	}
	if !detected[Claude] {
		ui.WarnPrintf("claude-code not detected; writing %s anyway (--targets).\n", MarkdownDest(Claude))
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
		ui.WarnPrintf("gemini-cli / antigravity not detected; writing %s anyway (--targets).\n",
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
		ui.WarnPrintf("cursor dirs not detected; writing ~/.cursor/rules/global-agents.mdc anyway (--targets).\n")
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

func (inst *Installer) installSkills(didClaude, didCursor, didAntigravity bool) error {
	if !inst.WithSkills {
		return nil
	}
	skillDirs := inst.repoSkillDirs()
	if len(skillDirs) == 0 {
		ui.WarnPrintf("skills deploy requested but no skill dirs with SKILL.md; skipping.\n")
		return nil
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
	return nil
}

func (inst *Installer) deployMarkdown(src, dest string, stripHooks bool) error {
	if inst.DryRun {
		ui.Printf("[dry-run] write %s (strip_hooks=%v, copy=%v)\n", dest, stripHooks, inst.UseCopy)
		return nil
	}

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
	ui.SuccessPrintf("Wrote: %s\n", dest)
	return nil
}

func (inst *Installer) linkOrCopy(src, dest string) error {
	if inst.DryRun {
		verb := "symlink"
		if inst.UseCopy {
			verb = "cp"
		}
		ui.Printf("[dry-run] %s %s -> %s\n", verb, src, dest)
		return nil
	}
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
		ui.SuccessPrintf("Copied: %s\n", dest)
		return nil
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	if err := os.Symlink(absSrc, dest); err != nil {
		return err
	}
	ui.SuccessPrintf("Symlinked: %s -> %s\n", dest, absSrc)
	return nil
}

func (inst *Installer) writeCursorMDC(globalPath string) error {
	dest := MarkdownDest(Cursor)
	if inst.DryRun {
		ui.Printf("[dry-run] write %s (frontmatter + GLOBAL_AGENTS.md)\n", dest)
		return nil
	}

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
	ui.SuccessPrintf("Wrote: %s\n", dest)
	return nil
}

func (inst *Installer) deployHookScripts() (string, error) {
	srcDir := filepath.Join(inst.RepoRoot, "hooks")
	destDir := HooksDeployDir()
	scripts := []string{"rtk-prepend.sh", "claude-rtk.sh", "gemini-rtk.sh", "cursor-rtk.sh"}

	if inst.DryRun {
		for _, s := range scripts {
			ui.Printf("[dry-run] cp %s -> %s\n", filepath.Join(srcDir, s), filepath.Join(destDir, s))
		}
		return destDir, nil
	}

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
	ui.SuccessPrintf("Deployed hook scripts to %s\n", destDir)
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
	hookKey := HookKey(a)

	return mergeHooksIntoSettings(settingsPath, snippet, hookKey, inst.DryRun)
}

func mergeHooksIntoSettings(settingsPath string, snippet map[string]any, hookKey string, dryRun bool) error {
	if dryRun {
		ui.Printf("[dry-run] merge hooks into %s (key: %s)\n", settingsPath, hookKey)
		return nil
	}

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

	existingEntries, _ := hooksSection[hookKey].([]any)

	snippetHooks, _ := snippet["hooks"].(map[string]any)
	newEntries, _ := snippetHooks[hookKey].([]any)

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
	ui.SuccessPrintf("Merged hooks into %s\n", settingsPath)
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
	if inst.DryRun {
		for _, sd := range skillDirs {
			ui.Printf("[dry-run] copytree %s -> %s\n", sd, filepath.Join(destRoot, filepath.Base(sd)))
		}
		return nil
	}

	if err := os.MkdirAll(destRoot, 0o750); err != nil {
		return err
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
	ui.SuccessPrintf("Copied %d skill(s) to %s\n", n, destRoot)
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
	if inst.DryRun {
		return nil
	}
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
	ui.SuccessPrintf("Wrote: %s\n", out)
	return nil
}
