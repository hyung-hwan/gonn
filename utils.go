package gonn

import "bytes"
import "html"
import "path/filepath"
import "regexp"
import "strconv"
import "strings"
import nethtml "golang.org/x/net/html"
import "golang.org/x/net/html/atom"

var html_elements map[string]struct{} = map[string]struct{}{
	"a": {},
	"abbr": {},
	"acronym": {},
	"address": {},
	"applet": {},
	"area": {},
	"b": {},
	"base": {},
	"basefont": {},
	"bdo": {},
	"big": {},
	"blockquote": {},
	"body": {},
	"br": {},
	"button": {},
	"caption": {},
	"center": {},
	"cite": {},
	"code": {},
	"col": {},
	"colgroup": {},
	"dd": {},
	"del": {},
	"dfn": {},
	"dir": {},
	"div": {},
	"dl": {},
	"dt": {},
	"em": {},
	"fieldset": {},
	"font": {},
	"form": {},
	"frame": {},
	"frameset": {},
	"h1": {},
	"h2": {},
	"h3": {},
	"h4": {},
	"h5": {},
	"h6": {},
	"head": {},
	"hr": {},
	"html": {},
	"i": {},
	"iframe": {},
	"img": {},
	"input": {},
	"ins": {},
	"isindex": {},
	"kbd": {},
	"label": {},
	"legend": {},
	"li": {},
	"link": {},
	"map": {},
	"menu": {},
	"meta": {},
	"noframes": {},
	"noscript": {},
	"object": {},
	"ol": {},
	"optgroup": {},
	"option": {},
	"p": {},
	"param": {},
	"pre": {},
	"q": {},
	"s": {},
	"samp": {},
	"script": {},
	"select": {},
	"small": {},
	"span": {},
	"strike": {},
	"strong": {},
	"style": {},
	"sub": {},
	"sup": {},
	"table": {},
	"tbody": {},
	"td": {},
	"textarea": {},
	"tfoot": {},
	"th": {},
	"thead": {},
	"title": {},
	"tr": {},
	"tt": {},
	"u": {},
	"ul": {},
	"var": {},
}

var html_block_elements map[string]struct{} = map[string]struct{}{
	"blockquote": {},
	"body": {},
	"colgroup": {},
	"dd": {},
	"div": {},
	"dl": {},
	"dt": {},
	"fieldset": {},
	"form": {},
	"frame": {},
	"frameset": {},
	"h1": {},
	"h2": {},
	"h3": {},
	"h4": {},
	"h5": {},
	"h6": {},
	"hr": {},
	"head": {},
	"html": {},
	"iframe": {},
	"li": {},
	"noframes": {},
	"noscript": {},
	"object": {},
	"ol": {},
	"optgroup": {},
	"option": {},
	"p": {},
	"param": {},
	"pre": {},
	"script": {},
	"select": {},
	"style": {},
	"table": {},
	"tbody": {},
	"td": {},
	"textarea": {},
	"tfoot": {},
	"th": {},
	"thead": {},
	"title": {},
	"tr": {},
	"tt": {},
	"ul": {},
}

var html_empty_elements map[string]struct{} = map[string]struct{}{
	"area": {},
	"base": {},
	"basefont": {},
	"br": {},
	"col": {},
	"hr": {},
	"input": {},
	"link": {},
	"meta": {},
}

var space_pattern *regexp.Regexp = regexp.MustCompile(`[ \t]+`)
var line_space_pattern *regexp.Regexp = regexp.MustCompile(`[\n ]+`)
var comment_pattern *regexp.Regexp = regexp.MustCompile(`(?s)/\*.+?\*/`)
var newline_compact_pattern *regexp.Regexp = regexp.MustCompile(`([;{,]) *\n`)
var blank_line_pattern *regexp.Regexp = regexp.MustCompile(`\n{2,}`)
var trailing_semicolon_pattern *regexp.Regexp = regexp.MustCompile(`[; ]+\}`)
var operator_space_pattern *regexp.Regexp = regexp.MustCompile(`([{;,+])[ ]+`)

func strings_contains(input string, token string) bool {
	return strings.Contains(input, token)
}

func split_fields(input string, separators string) []string {
	var normalized string
	var token string
	var index int

	normalized = input
	for index = 0; index < len(separators); index++ {
		token = string(separators[index])
		normalized = strings.ReplaceAll(normalized, token, " ")
	}

	return strings.Fields(normalized)
}

func atoi_or_zero(value string) int {
	var number int
	var err error

	number, err = strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return number
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func join_strings(values []string, separator string) string {
	return strings.Join(values, separator)
}

func has_html_element(name string) bool {
	var lower string
	var ok bool

	lower = strings.ToLower(name)
	_, ok = html_elements[lower]
	return ok
}

func is_block_element(name string) bool {
	var lower string
	var ok bool

	lower = strings.ToLower(name)
	_, ok = html_block_elements[lower]
	return ok
}

func is_empty_element(name string) bool {
	var lower string
	var ok bool

	lower = strings.ToLower(name)
	_, ok = html_empty_elements[lower]
	return ok
}

func child_of(node *nethtml.Node, tag string) bool {
	var current *nethtml.Node
	var lower string

	lower = strings.ToLower(tag)
	current = node

	for current != nil {
		if current.Type == nethtml.ElementNode && strings.ToLower(current.Data) == lower {
			return true
		}
		current = current.Parent
	}

	return false
}

func get_attribute(node *nethtml.Node, key string) (string, bool) {
	var index int

	for index = 0; index < len(node.Attr); index++ {
		if node.Attr[index].Key == key {
			return node.Attr[index].Val, true
		}
	}

	return "", false
}

func set_attribute(node *nethtml.Node, key string, value string) {
	var index int

	for index = 0; index < len(node.Attr); index++ {
		if node.Attr[index].Key == key {
			node.Attr[index].Val = value
			return
		}
	}

	node.Attr = append(node.Attr, nethtml.Attribute{Key: key, Val: value})
}

func has_class(node *nethtml.Node, class_name string) bool {
	var classes string
	var ok bool
	var items []string
	var index int

	classes, ok = get_attribute(node, "class")
	if !ok {
		return false
	}

	items = strings.Fields(classes)
	for index = 0; index < len(items); index++ {
		if items[index] == class_name {
			return true
		}
	}

	return false
}

func append_class(node *nethtml.Node, class_name string) {
	var classes string
	var ok bool

	if has_class(node, class_name) {
		return
	}

	classes, ok = get_attribute(node, "class")
	if !ok || classes == "" {
		set_attribute(node, "class", class_name)
		return
	}

	set_attribute(node, "class", classes+" "+class_name)
}

func first_element_child(node *nethtml.Node) *nethtml.Node {
	var child *nethtml.Node

	child = node.FirstChild
	for child != nil {
		if child.Type == nethtml.ElementNode {
			return child
		}
		child = child.NextSibling
	}

	return nil
}

func next_element_sibling(node *nethtml.Node) *nethtml.Node {
	var sibling *nethtml.Node

	sibling = node.NextSibling
	for sibling != nil {
		if sibling.Type == nethtml.ElementNode {
			return sibling
		}
		sibling = sibling.NextSibling
	}

	return nil
}

func previous_element_sibling(node *nethtml.Node) *nethtml.Node {
	var sibling *nethtml.Node

	sibling = node.PrevSibling
	for sibling != nil {
		if sibling.Type == nethtml.ElementNode {
			return sibling
		}
		sibling = sibling.PrevSibling
	}

	return nil
}

func direct_children_by_tag(node *nethtml.Node, tag string) []*nethtml.Node {
	var children []*nethtml.Node
	var child *nethtml.Node
	var lower string

	children = make([]*nethtml.Node, 0)
	lower = strings.ToLower(tag)
	child = node.FirstChild

	for child != nil {
		if child.Type == nethtml.ElementNode && strings.ToLower(child.Data) == lower {
			children = append(children, child)
		}
		child = child.NextSibling
	}

	return children
}

func first_descendant_by_tag(node *nethtml.Node, tag string) *nethtml.Node {
	var child *nethtml.Node
	var found *nethtml.Node
	var lower string

	lower = strings.ToLower(tag)
	child = node.FirstChild

	for child != nil {
		if child.Type == nethtml.ElementNode && strings.ToLower(child.Data) == lower {
			return child
		}
		found = first_descendant_by_tag(child, lower)
		if found != nil {
			return found
		}
		child = child.NextSibling
	}

	return found
}

func has_descendant_tag(node *nethtml.Node, tags []string) bool {
	var found bool
	var lookup map[string]struct{}
	var index int

	lookup = make(map[string]struct{}, len(tags))
	for index = 0; index < len(tags); index++ {
		lookup[strings.ToLower(tags[index])] = struct{}{}
	}

	walk_nodes(node, func(current *nethtml.Node) bool {
		if current != node && current.Type == nethtml.ElementNode {
			if _, found = lookup[strings.ToLower(current.Data)]; found {
				return false
			}
		}
		return true
	})

	return found
}

func walk_nodes(node *nethtml.Node, visit func(*nethtml.Node) bool) {
	var child *nethtml.Node
	var next *nethtml.Node

	if node == nil {
		return
	}

	if !visit(node) {
		return
	}

	child = node.FirstChild
	for child != nil {
		next = child.NextSibling
		walk_nodes(child, visit)
		child = next
	}
}

func collect_text(node *nethtml.Node) string {
	var builder strings.Builder

	walk_nodes(node, func(current *nethtml.Node) bool {
		if current.Type == nethtml.TextNode {
			builder.WriteString(current.Data)
		}
		if current.Type == nethtml.ElementNode && strings.ToLower(current.Data) == "br" {
			builder.WriteString("\n")
		}
		return true
	})

	return builder.String()
}

func render_node(node *nethtml.Node) string {
	var buffer bytes.Buffer

	if node == nil {
		return ""
	}

	nethtml.Render(&buffer, node)
	return buffer.String()
}

func render_children(node *nethtml.Node) string {
	var buffer bytes.Buffer
	var child *nethtml.Node

	child = node.FirstChild
	for child != nil {
		nethtml.Render(&buffer, child)
		child = child.NextSibling
	}

	return buffer.String()
}

func parse_fragment(context *nethtml.Node, source string) ([]*nethtml.Node, error) {
	var nodes []*nethtml.Node
	var err error

	nodes, err = nethtml.ParseFragment(strings.NewReader(source), context)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func replace_node_with_html(node *nethtml.Node, source string, context *nethtml.Node) error {
	var nodes []*nethtml.Node
	var err error
	var parent *nethtml.Node
	var index int

	parent = node.Parent
	if parent == nil {
		return nil
	}

	nodes, err = parse_fragment(context, source)
	if err != nil {
		return err
	}

	for index = 0; index < len(nodes); index++ {
		parent.InsertBefore(nodes[index], node)
	}

	parent.RemoveChild(node)
	return nil
}

func remove_node(node *nethtml.Node) {
	var parent *nethtml.Node

	parent = node.Parent
	if parent != nil {
		parent.RemoveChild(node)
	}
}

func insert_before_html(node *nethtml.Node, source string, context *nethtml.Node) (*nethtml.Node, error) {
	var nodes []*nethtml.Node
	var err error
	var parent *nethtml.Node
	var index int

	parent = node.Parent
	if parent == nil {
		return nil, nil
	}

	nodes, err = parse_fragment(context, source)
	if err != nil {
		return nil, err
	}

	for index = 0; index < len(nodes); index++ {
		parent.InsertBefore(nodes[index], node)
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	return nodes[0], nil
}

func set_inner_html(node *nethtml.Node, source string) error {
	var nodes []*nethtml.Node
	var err error
	var child *nethtml.Node
	var next *nethtml.Node
	var index int

	nodes, err = parse_fragment(node, source)
	if err != nil {
		return err
	}

	child = node.FirstChild
	for child != nil {
		next = child.NextSibling
		node.RemoveChild(child)
		child = next
	}

	for index = 0; index < len(nodes); index++ {
		node.AppendChild(nodes[index])
	}

	return nil
}

func rename_element(node *nethtml.Node, name string) {
	node.Data = name
	node.DataAtom = atom.Lookup([]byte(strings.ToLower(name)))
}

func escape_html_text(value string) string {
	return html.EscapeString(value)
}

func unescape_html_text(value string) string {
	return html.UnescapeString(value)
}

func unique_styles(styles []string) []string {
	var seen map[string]struct{}
	var result []string
	var index int
	var style string
	var ok bool

	seen = make(map[string]struct{})
	result = make([]string, 0, len(styles))

	for index = 0; index < len(styles); index++ {
		style = styles[index]
		if style == "" {
			continue
		}
		_, ok = seen[style]
		if ok {
			continue
		}
		seen[style] = struct{}{}
		result = append(result, style)
	}

	return result
}

func canonical_path(path string) string {
	var absolute string
	var err error

	absolute, err = filepath.Abs(path)
	if err != nil {
		return path
	}

	return absolute
}

func minify_css(data string) string {
	var output string

	output = data
	output = comment_pattern.ReplaceAllString(output, "")
	output = newline_compact_pattern.ReplaceAllString(output, "${1}")
	output = blank_line_pattern.ReplaceAllString(output, "\n")
	output = trailing_semicolon_pattern.ReplaceAllString(output, "}")
	output = operator_space_pattern.ReplaceAllString(output, "${1}")
	output = space_pattern.ReplaceAllString(output, " ")
	output = strings.TrimSpace(output)
	output = "  " + strings.ReplaceAll(output, "\n", "\n  ")
	return output
}
