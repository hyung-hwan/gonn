package gonn

import "fmt"
import "regexp"
import "strings"
import "time"
import nethtml "golang.org/x/net/html"

type RoffRenderer struct {
	html string
	name string
	section string
	tagline string
	manual string
	organization string
	date time.Time
	buffer []string
}

var trailing_space_pattern *regexp.Regexp = regexp.MustCompile(`(?m)[ \t]+$`)

func NewRoffRenderer(html string, name string, section string, tagline string, manual string, organization string, date time.Time) *RoffRenderer {
	return &RoffRenderer{
		html: html,
		name: name,
		section: section,
		tagline: tagline,
		manual: manual,
		organization: organization,
		date: date,
		buffer: make([]string, 0, 64),
	}
}

func (renderer *RoffRenderer) Render() (string, error) {
	var root *nethtml.Node
	var err error
	var output string

	renderer.title_heading()
	root, err = parse_html_root(renderer.html)
	if err != nil {
		return "", err
	}
	renderer.remove_extraneous(root)
	renderer.normalize_whitespace(root)
	renderer.block_filter(root)
	renderer.write("\n")
	output = strings.Join(renderer.buffer, "")
	output = trailing_space_pattern.ReplaceAllString(output, "")
	return output, nil
}

func (renderer *RoffRenderer) title_heading() {
	renderer.comment(fmt.Sprintf("generated with Gonn/v%s", Version()))
	renderer.comment(fmt.Sprintf("https://github.com/hyung-hwan/gonn/tree/%s", Revision))
	if renderer.name == "" {
		return
	}
	renderer.macro("TH", fmt.Sprintf("\"%s\" \"%s\" \"%s\" \"%s\" \"%s\"",
		renderer.escape_roff(strings.ToUpper(renderer.name)),
		renderer.section,
		renderer.date.Format("January 2006"),
		renderer.organization,
		renderer.manual))
}

func (renderer *RoffRenderer) remove_extraneous(root *nethtml.Node) {
	var nodes []*nethtml.Node
	var index int

	nodes = make([]*nethtml.Node, 0)

	walk_nodes(root, func(node *nethtml.Node) bool {
		if node.Type == nethtml.CommentNode {
			nodes = append(nodes, node)
		}
		return true
	})

	for index = 0; index < len(nodes); index++ {
		remove_node(nodes[index])
	}
}

func (renderer *RoffRenderer) normalize_whitespace(node *nethtml.Node) {
	var child *nethtml.Node
	var next *nethtml.Node
	var content string
	var previous *nethtml.Node
	var following *nethtml.Node

	if node == nil {
		return
	}

	if node.Type == nethtml.TextNode {
		previous = node.PrevSibling
		following = node.NextSibling
		content = line_space_pattern.ReplaceAllString(node.Data, " ")
		if previous == nil || is_block_or_br(previous) {
			content = strings.TrimLeft(content, " ")
		}
		if following == nil || is_block_or_br(following) {
			content = strings.TrimRight(content, " ")
		}
		if content == "" {
			remove_node(node)
		} else {
			node.Data = content
		}
		return
	}

	if node.Type == nethtml.ElementNode && strings.ToLower(node.Data) == "pre" {
		return
	}

	child = node.FirstChild
	for child != nil {
		next = child.NextSibling
		renderer.normalize_whitespace(child)
		child = next
	}
}

func is_block_or_br(node *nethtml.Node) bool {
	if node == nil {
		return true
	}
	if node.Type != nethtml.ElementNode {
		return false
	}
	if strings.ToLower(node.Data) == "br" {
		return true
	}
	return is_block_element(node.Data)
}

func (renderer *RoffRenderer) block_filter(node *nethtml.Node) {
	var child *nethtml.Node
	var next *nethtml.Node
	var lower string
	var previous *nethtml.Node
	var has_previous bool
	var indent bool
	var parent_name string
	var row *nethtml.Node
	var columns []*nethtml.Node
	var formats []string
	var contents []string
	var index int
	var value string

	if node == nil {
		return
	}

	if node.Type == nethtml.DocumentNode || (node.Type == nethtml.ElementNode && strings.ToLower(node.Data) == "div" && node.Parent == nil) {
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
		return
	}

	if node.Type == nethtml.TextNode {
		renderer.inline_filter(node)
		return
	}

	if node.Type != nethtml.ElementNode {
		return
	}

	lower = strings.ToLower(node.Data)
	if html_should_convert_literal_angle_tag(lower) {
		return
	}
	switch lower {
	case "div":
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
	case "h1":
		return
	case "h2":
		renderer.macro("SH", renderer.quote(renderer.escape_roff(render_children(node))))
	case "h3":
		renderer.macro("SS", renderer.quote(renderer.escape_roff(render_children(node))))
	case "h4", "h5", "h6":
		renderer.macro("SS", renderer.quote(renderer.escape_roff(render_children(node))))
	case "p":
		previous = previous_element_sibling(node)
		has_previous = node.PrevSibling != nil
		parent_name = ""
		if node.Parent != nil && node.Parent.Type == nethtml.ElementNode {
			parent_name = strings.ToLower(node.Parent.Data)
		}
		if has_previous && (parent_name == "dd" || parent_name == "li" || parent_name == "blockquote") {
			renderer.macro("IP", "")
		} else if previous != nil {
			if strings.ToLower(previous.Data) != "h1" && strings.ToLower(previous.Data) != "h2" && strings.ToLower(previous.Data) != "h3" {
				renderer.macro("P", "")
			}
		}
		renderer.inline_filter_children(node)
	case "blockquote":
		previous = previous_element_sibling(node)
		indent = previous == nil || (strings.ToLower(previous.Data) != "h1" && strings.ToLower(previous.Data) != "h2" && strings.ToLower(previous.Data) != "h3")
		if indent {
			renderer.macro("IP", "\"\" 4")
		}
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
		if indent {
			renderer.macro("IP", "\"\" 0")
		}
	case "pre":
		previous = previous_element_sibling(node)
		indent = previous == nil || (strings.ToLower(previous.Data) != "h1" && strings.ToLower(previous.Data) != "h2" && strings.ToLower(previous.Data) != "h3")
		if indent {
			renderer.macro("IP", "\"\" 4")
		}
		renderer.macro("nf", "")
		if node.FirstChild != nil && node.FirstChild.Type == nethtml.TextNode && strings.HasPrefix(node.FirstChild.Data, "\n") {
			node.FirstChild.Data = node.FirstChild.Data[1:]
		}
		renderer.inline_filter_children(node)
		renderer.macro("fi", "")
		if indent {
			renderer.macro("IP", "\"\" 0")
		}
	case "dl":
		renderer.macro("TP", "")
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
	case "dt":
		previous = previous_element_sibling(node)
		if previous != nil {
			renderer.macro("TP", "")
		}
		renderer.inline_filter_children(node)
		renderer.write("\n")
	case "dd":
		if first_descendant_by_tag(node, "p") != nil {
			child = node.FirstChild
			for child != nil {
				next = child.NextSibling
				renderer.block_filter(child)
				child = next
			}
		} else {
			renderer.inline_filter_children(node)
		}
		renderer.write("\n")
	case "ol":
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
		renderer.macro("IP", "\"\" 0")
	case "ul":
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
		renderer.macro("IP", "\"\" 0")
	case "li":
		if node.Parent != nil && node.Parent.Type == nethtml.ElementNode && strings.ToLower(node.Parent.Data) == "ol" {
			renderer.macro("IP", fmt.Sprintf("\"%d.\" 4", list_item_position(node)+1))
		}
		if node.Parent != nil && node.Parent.Type == nethtml.ElementNode && strings.ToLower(node.Parent.Data) == "ul" {
			renderer.macro("IP", "\"\\[ci]\" 4")
		}
		if has_descendant_tag(node, []string{"p", "ol", "ul", "dl", "div"}) {
			child = node.FirstChild
			for child != nil {
				next = child.NextSibling
				renderer.block_filter(child)
				child = next
			}
		} else {
			renderer.inline_filter_children(node)
		}
		renderer.write("\n")
	case "table":
		renderer.macro("TS", "")
		renderer.write("allbox;\n")
		renderer.inline_table_children(node)
		renderer.macro("TE", "")
	case "thead":
		row = first_descendant_by_tag(node, "tr")
		if row == nil {
			return
		}
		columns = direct_children_by_tag(row, "th")
		formats = make([]string, 0, len(columns))
		contents = make([]string, 0, len(columns))
		for index = 0; index < len(columns); index++ {
			formats = append(formats, renderer.table_cell_alignment(columns[index]))
			contents = append(contents, renderer.escape_roff(render_children(columns[index])))
		}
		renderer.write(strings.Join(formats, " ") + ".\n")
		renderer.write(strings.Join(contents, "\t") + "\n")
	case "th":
		return
	case "tbody":
		renderer.inline_table_children(node)
	case "tr":
		columns = make([]*nethtml.Node, 0)
		child = node.FirstChild
		for child != nil {
			if child.Type == nethtml.ElementNode && (strings.ToLower(child.Data) == "td" || strings.ToLower(child.Data) == "th") {
				columns = append(columns, child)
			}
			child = child.NextSibling
		}
		for index = 0; index < len(columns); index++ {
			renderer.block_filter(columns[index])
			if index != len(columns)-1 {
				renderer.write("\t")
			}
		}
		renderer.write("\n")
	case "td":
		value = strings.TrimSpace(collect_text(node))
		if value == "" {
			renderer.inline_filter_children(node)
		} else {
			renderer.inline_filter_children(node)
		}
	case "span", "code", "b", "strong", "kbd", "samp", "var", "em", "i", "u", "br", "a":
		renderer.inline_filter(node)
	default:
		child = node.FirstChild
		for child != nil {
			next = child.NextSibling
			renderer.block_filter(child)
			child = next
		}
	}
}

func (renderer *RoffRenderer) inline_table_children(node *nethtml.Node) {
	var child *nethtml.Node
	var next *nethtml.Node

	child = node.FirstChild
	for child != nil {
		next = child.NextSibling
		renderer.block_filter(child)
		child = next
	}
}

func (renderer *RoffRenderer) table_cell_alignment(node *nethtml.Node) string {
	var style string
	var align string
	var ok bool
	var normalized string

	style, ok = get_attribute(node, "style")
	if ok {
		normalized = strings.ToLower(style)
		normalized = strings.ReplaceAll(normalized, " ", "")
		if strings.Contains(normalized, "text-align:left") {
			return "l"
		}
		if strings.Contains(normalized, "text-align:right") {
			return "r"
		}
		if strings.Contains(normalized, "text-align:center") {
			return "c"
		}
	}

	align, ok = get_attribute(node, "align")
	if ok {
		align = strings.ToLower(strings.TrimSpace(align))
		if align == "left" {
			return "l"
		}
		if align == "right" {
			return "r"
		}
		if align == "center" {
			return "c"
		}
	}

	return "l"
}

func list_item_position(node *nethtml.Node) int {
	var position int
	var sibling *nethtml.Node

	position = 0
	sibling = node.PrevSibling
	for sibling != nil {
		if sibling.Type == nethtml.ElementNode && strings.ToLower(sibling.Data) == "li" {
			position++
		}
		sibling = sibling.PrevSibling
	}

	return position
}

func (renderer *RoffRenderer) inline_filter_children(node *nethtml.Node) {
	var child *nethtml.Node
	var next *nethtml.Node

	child = node.FirstChild
	for child != nil {
		next = child.NextSibling
		renderer.inline_filter(child)
		child = next
	}
}

func (renderer *RoffRenderer) inline_filter(node *nethtml.Node) {
	var lower string
	var href string
	var ok bool

	if node == nil {
		return
	}

	if node.Type == nethtml.TextNode {
		renderer.write(renderer.escape_roff(node.Data))
		return
	}

	if node.Type != nethtml.ElementNode {
		return
	}

	lower = strings.ToLower(node.Data)
	if html_should_convert_literal_angle_tag(lower) {
		return
	}
	switch lower {
	case "span":
		renderer.inline_filter_children(node)
	case "code":
		if child_of(node, "pre") {
			renderer.inline_filter_children(node)
		} else {
			renderer.write(`\fB`)
			renderer.inline_filter_children(node)
			renderer.write(`\fR`)
		}
	case "b", "strong", "kbd", "samp":
		renderer.write(`\fB`)
		renderer.inline_filter_children(node)
		renderer.write(`\fR`)
	case "var", "em", "i", "u":
		renderer.write(`\fI`)
		renderer.inline_filter_children(node)
		renderer.write(`\fR`)
	case "br":
		renderer.macro("br", "")
	case "a":
		if has_class(node, "man-ref") {
			renderer.inline_filter_children(node)
		} else if _, ok = get_attribute(node, "data-bare-link"); ok {
			renderer.write(`\fI`)
			renderer.inline_filter_children(node)
			renderer.write(`\fR`)
		} else {
			renderer.inline_filter_children(node)
			renderer.write(" ")
			href, ok = get_attribute(node, "href")
			if ok {
				renderer.write(`\fI`)
				renderer.write(renderer.escape_roff(href))
				renderer.write(`\fR`)
			}
		}
	case "sup":
		renderer.write("^(")
		renderer.inline_filter_children(node)
		renderer.write(")")
	default:
		renderer.inline_filter_children(node)
	}
}

func (renderer *RoffRenderer) macro(name string, value string) {
	if value == "" {
		renderer.writeln("." + name)
		return
	}
	renderer.writeln("." + name + " " + value)
}

func (renderer *RoffRenderer) quote(text string) string {
	return "\"" + strings.ReplaceAll(text, "\"", "\\\"") + "\""
}

func (renderer *RoffRenderer) write(text string) {
	var current string
	var ends_in_newline bool

	if text == "" {
		return
	}

	text = strings.ReplaceAll(text, "\n\\.", "\n\\&\\.")
	text = strings.ReplaceAll(text, "\n'", "\n\\&'")

	ends_in_newline = false
	if len(renderer.buffer) != 0 {
		current = renderer.buffer[len(renderer.buffer)-1]
		ends_in_newline = strings.HasSuffix(current, "\n")
	}

	if strings.HasPrefix(text, `\.`) && ends_in_newline {
		renderer.buffer = append(renderer.buffer, `\&`)
	}
	if strings.HasPrefix(text, `'`) && ends_in_newline {
		renderer.buffer = append(renderer.buffer, `\&`)
	}

	renderer.buffer = append(renderer.buffer, text)
}

func (renderer *RoffRenderer) writeln(text string) {
	var current string

	if len(renderer.buffer) != 0 {
		current = renderer.buffer[len(renderer.buffer)-1]
		if !strings.HasSuffix(current, "\n") {
			renderer.write("\n")
		}
	}
	renderer.write(text)
	renderer.write("\n")
}

func (renderer *RoffRenderer) comment(text string) {
	renderer.writeln(".\\\" " + text)
}

func (renderer *RoffRenderer) escape_roff(text string) string {
	var output string
	var builder strings.Builder
	var index int
	var ch rune

	if text == "" {
		return ""
	}

	output = text
	output = strings.ReplaceAll(output, "\\", `\e`)
	output = strings.ReplaceAll(output, "&bull;", `\[ci]`)
	output = strings.ReplaceAll(output, "&lt;", "<")
	output = strings.ReplaceAll(output, "&gt;", ">")
	output = strings.ReplaceAll(output, "&nbsp;", `\~`)
	output = strings.ReplaceAll(output, "&copy;", `\(co`)
	output = strings.ReplaceAll(output, "&rdquo;", `\(rs`)
	output = strings.ReplaceAll(output, "&mdash;", `\(em`)
	output = strings.ReplaceAll(output, "&reg;", `\(rg`)
	output = strings.ReplaceAll(output, "&sec;", `\(sc`)
	output = strings.ReplaceAll(output, "&ge;", `\(>=`)
	output = strings.ReplaceAll(output, "&le;", `\(<=`)
	output = strings.ReplaceAll(output, "&ne;", `\(!=`)
	output = strings.ReplaceAll(output, "&equiv;", `\(==`)
	output = strings.ReplaceAll(output, "&amp;", "&")
	output = strings.ReplaceAll(output, "...", `\|.\|.\|.`)

	for index, ch = range output {
		_ = index
		switch ch {
		case '•':
			builder.WriteString(`\[ci]`)
		case '<':
			builder.WriteString("<")
		case '>':
			builder.WriteString(">")
		case '\u00A0':
			builder.WriteString(`\~`)
		case '©':
			builder.WriteString(`\(co`)
		case '”':
			builder.WriteString(`\(rs`)
		case '—':
			builder.WriteString(`\(em`)
		case '®':
			builder.WriteString(`\(rg`)
		case '§':
			builder.WriteString(`\(sc`)
		case '≥':
			builder.WriteString(`\(>=`)
		case '≤':
			builder.WriteString(`\(<=`)
		case '≠':
			builder.WriteString(`\(!=`)
		case '≡':
			builder.WriteString(`\(==`)
		case '.', '-':
			builder.WriteString(`\`)
			builder.WriteRune(ch)
		default:
			builder.WriteRune(ch)
		}
	}

	return builder.String()
}
