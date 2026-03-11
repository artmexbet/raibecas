package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// editorJSOutput represents the top-level Editor.js output structure.
type editorJSOutput struct {
	Blocks []editorJSBlock `json:"blocks"`
}

// editorJSBlock represents a single Editor.js block.
type editorJSBlock struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// parseEditorJSContent detects whether content is Editor.js JSON and converts it
// to Markdown. If content is not valid Editor.js JSON it is returned unchanged.
func parseEditorJSContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return content
	}

	var output editorJSOutput
	if err := json.Unmarshal([]byte(trimmed), &output); err != nil || len(output.Blocks) == 0 {
		return content
	}

	parts := make([]string, 0, len(output.Blocks))
	for _, block := range output.Blocks {
		md := blockToMarkdown(block)
		if md != "" {
			parts = append(parts, md)
		}
	}
	return strings.Join(parts, "\n\n")
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// stripHTML converts simple HTML inline tags used by Editor.js to Markdown equivalents.
func stripHTML(s string) string {
	// Bold
	s = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(s, "**$1**")
	s = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(s, "**$1**")
	// Italic
	s = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(s, "_$1_")
	s = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(s, "_$1_")
	// Underline → markdown has no standard underline, keep as-is text
	s = regexp.MustCompile(`<u>(.*?)</u>`).ReplaceAllString(s, "$1")
	// Strikethrough
	s = regexp.MustCompile(`<s>(.*?)</s>`).ReplaceAllString(s, "~~$1~~")
	s = regexp.MustCompile(`<del>(.*?)</del>`).ReplaceAllString(s, "~~$1~~")
	// Highlight/mark
	s = regexp.MustCompile(`<mark[^>]*>(.*?)</mark>`).ReplaceAllString(s, "==$1==")
	// Inline code
	s = regexp.MustCompile(`<code[^>]*>(.*?)</code>`).ReplaceAllString(s, "`$1`")
	// Links
	s = regexp.MustCompile(`<a href="([^"]*)"[^>]*>(.*?)</a>`).ReplaceAllString(s, "[$2]($1)")
	// Line breaks
	s = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(s, "\n")
	// Remove remaining tags
	s = htmlTagRe.ReplaceAllString(s, "")
	return s
}

func blockToMarkdown(block editorJSBlock) string {
	switch block.Type {
	case "header":
		return headerBlock(block.Data)
	case "paragraph":
		return paragraphBlock(block.Data)
	case "list":
		return listBlock(block.Data)
	case "checklist":
		return checklistBlock(block.Data)
	case "code":
		return codeBlock(block.Data)
	case "quote":
		return quoteBlock(block.Data)
	case "delimiter":
		return "---"
	case "table":
		return tableBlock(block.Data)
	case "image":
		return imageBlock(block.Data)
	case "raw":
		return rawBlock(block.Data)
	default:
		// Fallback: try to extract "text" field
		var generic map[string]interface{}
		if err := json.Unmarshal(block.Data, &generic); err == nil {
			if text, ok := generic["text"].(string); ok {
				return stripHTML(text)
			}
		}
		return ""
	}
}

func headerBlock(raw json.RawMessage) string {
	var data struct {
		Text  string `json:"text"`
		Level int    `json:"level"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	level := data.Level
	if level < 1 {
		level = 2
	}
	return strings.Repeat("#", level) + " " + stripHTML(data.Text)
}

func paragraphBlock(raw json.RawMessage) string {
	var data struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	return stripHTML(data.Text)
}

func listBlock(raw json.RawMessage) string {
	var data struct {
		Style string            `json:"style"`
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	ordered := data.Style == "ordered"
	lines := make([]string, 0, len(data.Items))
	for i, itemRaw := range data.Items {
		// Items can be strings or objects {content: string}
		var itemStr string
		if err := json.Unmarshal(itemRaw, &itemStr); err != nil {
			var itemObj struct {
				Content string `json:"content"`
			}
			if err2 := json.Unmarshal(itemRaw, &itemObj); err2 != nil {
				continue
			}
			itemStr = itemObj.Content
		}
		prefix := "-"
		if ordered {
			prefix = fmt.Sprintf("%d.", i+1)
		}
		lines = append(lines, prefix+" "+stripHTML(itemStr))
	}
	return strings.Join(lines, "\n")
}

func checklistBlock(raw json.RawMessage) string {
	var data struct {
		Items []struct {
			Text    string `json:"text"`
			Checked bool   `json:"checked"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	lines := make([]string, 0, len(data.Items))
	for _, item := range data.Items {
		check := " "
		if item.Checked {
			check = "x"
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s", check, stripHTML(item.Text)))
	}
	return strings.Join(lines, "\n")
}

func codeBlock(raw json.RawMessage) string {
	var data struct {
		Code     string `json:"code"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	return fmt.Sprintf("```%s\n%s\n```", data.Language, data.Code)
}

func quoteBlock(raw json.RawMessage) string {
	var data struct {
		Text    string `json:"text"`
		Caption string `json:"caption"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	text := stripHTML(data.Text)
	lines := strings.Split(text, "\n")
	quoted := make([]string, 0, len(lines)+2)
	for _, l := range lines {
		quoted = append(quoted, "> "+l)
	}
	if data.Caption != "" {
		quoted = append(quoted, ">")
		quoted = append(quoted, "> — "+stripHTML(data.Caption))
	}
	return strings.Join(quoted, "\n")
}

func tableBlock(raw json.RawMessage) string {
	var data struct {
		WithHeadings bool       `json:"withHeadings"`
		Content      [][]string `json:"content"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || len(data.Content) == 0 {
		return ""
	}

	renderRow := func(cells []string) string {
		escaped := make([]string, len(cells))
		for i, c := range cells {
			escaped[i] = stripHTML(c)
		}
		return "| " + strings.Join(escaped, " | ") + " |"
	}
	separatorRow := func(n int) string {
		cols := make([]string, n)
		for i := range cols {
			cols[i] = "---"
		}
		return "| " + strings.Join(cols, " | ") + " |"
	}

	var sb strings.Builder
	startRow := 0
	if data.WithHeadings && len(data.Content) > 0 {
		sb.WriteString(renderRow(data.Content[0]) + "\n")
		sb.WriteString(separatorRow(len(data.Content[0])) + "\n")
		startRow = 1
	} else if len(data.Content) > 0 {
		sb.WriteString(renderRow(data.Content[0]) + "\n")
		sb.WriteString(separatorRow(len(data.Content[0])) + "\n")
		startRow = 1
	}
	for _, row := range data.Content[startRow:] {
		sb.WriteString(renderRow(row) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func imageBlock(raw json.RawMessage) string {
	var data struct {
		File struct {
			URL string `json:"url"`
		} `json:"file"`
		URL     string `json:"url"`
		Caption string `json:"caption"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	url := data.File.URL
	if url == "" {
		url = data.URL
	}
	return fmt.Sprintf("![%s](%s)", stripHTML(data.Caption), url)
}

func rawBlock(raw json.RawMessage) string {
	var data struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	return data.HTML
}
