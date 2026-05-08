package ticket

import (
	"fmt"
	"slices"
	"strings"
)

// adfToMarkdown converts Atlassian Document Format (ADF) JSON into GitHub-flavored markdown.
func adfToMarkdown(v any) string {
	return strings.TrimSpace(adfBlockToMarkdown(v))
}

func adfBlockToMarkdown(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var parts []string
		for _, el := range t {
			if s := adfBlockToMarkdown(el); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n\n")
	case map[string]any:
		typ, _ := t["type"].(string)
		switch typ {
		case "doc":
			return joinDocBlocks(t["content"])
		case "paragraph":
			return renderParagraph(t)
		case "heading":
			level := headingLevel(t)
			prefix := strings.Repeat("#", level) + " "
			return prefix + renderInlineContent(t["content"])
		case "bulletList":
			return renderBulletList(t)
		case "orderedList":
			start := orderedListStart(t)
			return renderOrderedList(t, start)
		case "blockquote":
			inner := joinDocBlocks(t["content"])
			if inner == "" {
				return ""
			}
			var lines []string
			for ln := range strings.SplitSeq(inner, "\n") {
				lines = append(lines, "> "+ln)
			}
			return strings.Join(lines, "\n")
		case "codeBlock":
			return renderCodeBlock(t)
		case "rule":
			return "---"
		case "listItem":
			return renderListItemBlocks(t["content"], "- ", 0)
		case "inlineCard", "blockCard", "embedCard":
			return inlineCardToMarkdown(t)
		case "text":
			return applyTextMarks(t)
		default:
			if c, ok := t["content"]; ok {
				return adfBlockToMarkdown(c)
			}
			return ""
		}
	default:
		return ""
	}
}

func joinDocBlocks(content any) string {
	arr, ok := content.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, el := range arr {
		if s := adfBlockToMarkdown(el); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderParagraph(m map[string]any) string {
	return renderInlineContent(m["content"])
}

func renderInlineContent(content any) string {
	arr, ok := content.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, el := range arr {
		b.WriteString(adfInlineNode(el))
	}
	return b.String()
}

func adfInlineNode(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	typ, _ := m["type"].(string)
	switch typ {
	case "text":
		return applyTextMarks(m)
	case "hardBreak":
		return "  \n"
	case "inlineCard", "blockCard", "embedCard":
		return inlineCardToMarkdown(m)
	default:
		if c, ok := m["content"]; ok {
			return adfBlockToMarkdown(map[string]any{"type": typ, "content": c})
		}
		return ""
	}
}

func applyTextMarks(m map[string]any) string {
	text, _ := m["text"].(string)
	marks, _ := m["marks"].([]any)
	s := text
	for _, v := range slices.Backward(marks) {
		mm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		mt, _ := mm["type"].(string)
		switch mt {
		case "strong":
			s = "**" + s + "**"
		case "em":
			s = "*" + s + "*"
		case "code":
			s = "`" + strings.ReplaceAll(s, "`", "`"+"`") + "`"
		case "link":
			attrs, _ := mm["attrs"].(map[string]any)
			href := ""
			if attrs != nil {
				href, _ = attrs["href"].(string)
			}
			if strings.TrimSpace(s) == "" && strings.TrimSpace(href) != "" {
				s = "<" + href + ">"
				break
			}
			s = "[" + s + "](" + href + ")"
		case "strike":
			s = "~~" + s + "~~"
		default:
			// unknown mark: ignore
		}
	}
	return s
}

func headingLevel(m map[string]any) int {
	attrs, ok := m["attrs"].(map[string]any)
	if !ok {
		return 1
	}
	switch lv := attrs["level"].(type) {
	case float64:
		n := int(lv)
		if n < 1 {
			return 1
		}
		if n > 6 {
			return 6
		}
		return n
	case int:
		if lv < 1 {
			return 1
		}
		if lv > 6 {
			return 6
		}
		return lv
	default:
		return 1
	}
}

func renderBulletList(m map[string]any) string {
	items, _ := m["content"].([]any)
	var lines []string
	for _, item := range items {
		im, ok := item.(map[string]any)
		if !ok || im["type"] != "listItem" {
			continue
		}
		lines = append(lines, renderListItemBlocks(im["content"], "- ", 0))
	}
	return strings.Join(lines, "\n")
}

func orderedListStart(m map[string]any) int {
	attrs, ok := m["attrs"].(map[string]any)
	if !ok {
		return 1
	}
	switch o := attrs["order"].(type) {
	case float64:
		n := int(o)
		if n < 1 {
			return 1
		}
		return n
	case int:
		if o < 1 {
			return 1
		}
		return o
	default:
		return 1
	}
}

func renderOrderedList(m map[string]any, start int) string {
	items, _ := m["content"].([]any)
	var lines []string
	n := start
	for _, item := range items {
		im, ok := item.(map[string]any)
		if !ok || im["type"] != "listItem" {
			continue
		}
		prefix := fmt.Sprintf("%d. ", n)
		lines = append(lines, renderListItemBlocks(im["content"], prefix, 0))
		n++
	}
	return strings.Join(lines, "\n")
}

func renderListItemBlocks(content any, firstPrefix string, depth int) string {
	arr, ok := content.([]any)
	if !ok {
		return ""
	}
	var lines []string
	first := true
	indent := strings.Repeat("  ", depth)
	for _, el := range arr {
		em, ok := el.(map[string]any)
		if !ok {
			continue
		}
		switch em["type"] {
		case "paragraph":
			text := renderInlineContent(em["content"])
			if first {
				lines = append(lines, indent+firstPrefix+text)
				first = false
			} else {
				cont := strings.Repeat(" ", len(firstPrefix))
				lines = append(lines, indent+cont+text)
			}
		case "bulletList":
			nested := renderBulletList(em)
			if nested != "" {
				ind := indent + strings.Repeat(" ", len(firstPrefix))
				for nl := range strings.SplitSeq(nested, "\n") {
					lines = append(lines, ind+nl)
				}
			}
		case "orderedList":
			nested := renderOrderedList(em, orderedListStart(em))
			if nested != "" {
				ind := indent + strings.Repeat(" ", len(firstPrefix))
				for nl := range strings.SplitSeq(nested, "\n") {
					lines = append(lines, ind+nl)
				}
			}
		default:
			if s := adfBlockToMarkdown(el); s != "" {
				ind := indent + strings.Repeat(" ", len(firstPrefix))
				for nl := range strings.SplitSeq(s, "\n") {
					lines = append(lines, ind+nl)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderCodeBlock(m map[string]any) string {
	lang := ""
	if attrs, ok := m["attrs"].(map[string]any); ok {
		if l, ok := attrs["language"].(string); ok {
			lang = l
		}
	}
	body := codeBlockText(m["content"])
	body = strings.TrimRight(body, "\n")
	return "```" + lang + "\n" + body + "\n```"
}

func codeBlockText(content any) string {
	arr, ok := content.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, el := range arr {
		if tm, ok := el.(map[string]any); ok && tm["type"] == "text" {
			if s, ok := tm["text"].(string); ok {
				b.WriteString(s)
			}
		}
	}
	return b.String()
}

func cardURLFromAttrs(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	for _, key := range []string{"url", "href", "originalUrl"} {
		s, ok := attrs[key].(string)
		if ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func inlineCardTitleFromAttrs(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	if t, ok := attrs["title"].(string); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

// inlineCardToMarkdown renders Jira smart links (inlineCard / blockCard / embedCard) as GFM.
func inlineCardToMarkdown(m map[string]any) string {
	attrs, _ := m["attrs"].(map[string]any)
	url := cardURLFromAttrs(attrs)
	if url == "" {
		return ""
	}
	if title := inlineCardTitleFromAttrs(attrs); title != "" {
		return fmt.Sprintf("[%s](%s)", title, url)
	}
	return "<" + url + ">"
}
