package gonn

import "bytes"
import "encoding/json"
import "fmt"
import "os"
import "path/filepath"
import "regexp"
import "strings"
import "time"
import "github.com/yuin/goldmark"
import goldhtml "github.com/yuin/goldmark/renderer/html"
import nethtml "golang.org/x/net/html"
import "golang.org/x/net/html/atom"
import yaml "gopkg.in/yaml.v3"

type SectionHead struct {
	ID string
	Text string
}

type DocumentOptions struct {
	Name string
	Section string
	Manual string
	Organization string
	Date *time.Time
	Styles []string
	Reader func(string) (string, error)
	Index *Index
}

type Document struct {
	Path string
	Data string
	Index *Index
	Name string
	Section string
	Tagline string
	Manual string
	Organization string
	Date *time.Time
	Styles []string
	basename string
	reader func(string) (string, error)
	markdown string
	html_fragment string
	toc []SectionHead
	name_set bool
	section_set bool
}

var title_pattern *regexp.Regexp = regexp.MustCompile(`^([\w_.\[\]~+=@:-]+)\s*\((\d\w*)\)\s*-+\s*(.*)$`)
var short_title_pattern *regexp.Regexp = regexp.MustCompile(`^([\w_.\[\]~+=@:-]+)\s+-+\s+(.*)$`)
var markdown_heading_anchor_pattern *regexp.Regexp = regexp.MustCompile(`^[#]{2,5} +[\w '-]+[# ]*$`)
var markdown_heading_cleanup_pattern *regexp.Regexp = regexp.MustCompile(`[^\w -]`)
var anchor_cleanup_pattern *regexp.Regexp = regexp.MustCompile(`\W+`)
var anchor_trim_pattern *regexp.Regexp = regexp.MustCompile(`(^-+|-+$)`)
var angle_quote_pattern *regexp.Regexp = regexp.MustCompile(`<([^:.\/]+?)>`)
var literal_angle_pattern *regexp.Regexp = regexp.MustCompile(`<([^>]+)>`)
var manual_reference_pattern *regexp.Regexp = regexp.MustCompile(`([0-9A-Za-z_:.+=@~-]+)(\(\d+\w*\))`)
var manual_reference_name_pattern *regexp.Regexp = regexp.MustCompile(`^[0-9A-Za-z_:.+=@~-]+$`)
var manual_reference_section_pattern *regexp.Regexp = regexp.MustCompile(`^\((\d+\w*)\)`)

func NewDocument(path string, options DocumentOptions) (*Document, error) {
	var document *Document
	var data string
	var err error
	var sniff_name string
	var sniff_section string
	var sniff_tagline string

	document = &Document{
		Path: path,
		Styles: []string{"man"},
	}

	if path != "" && path != "-" {
		document.basename = filepath.Base(path)
	}

	if options.Reader != nil {
		document.reader = options.Reader
	} else {
		document.reader = default_reader
	}

	data, err = document.reader(path)
	if err != nil {
		return nil, err
	}

	document.Data = data
	sniff_name, sniff_section, sniff_tagline = sniff_document(data)
	document.Name = sniff_name
	document.Section = sniff_section
	document.Tagline = sniff_tagline
	document.name_set = sniff_name != ""
	document.section_set = sniff_section != ""
	document.Manual = options.Manual
	document.Organization = options.Organization
	document.Date = options.Date

	if options.Index != nil {
		document.Index = options.Index
	} else {
		if path == "" {
			document.Index = IndexForPath(".")
		} else {
			document.Index = IndexForPath(path)
		}
	}

	document.SetStyles(options.Styles)

	if options.Name != "" {
		document.Name = options.Name
		document.name_set = true
	}
	if options.Section != "" {
		document.Section = options.Section
		document.section_set = true
	}

	if document.Index != nil && path != "" && path != "-" && document.NameValue() != "" {
		document.Index.AddManual(document)
	}

	return document, nil
}

func default_reader(path string) (string, error) {
	var data []byte
	var err error

	if path == "" || path == "-" {
		data, err = os.ReadFile("/dev/stdin")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	data, err = os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func sniff_document(data string) (string, string, string) {
	var preview string
	var heading string
	var matches []string

	preview = data
	if len(preview) > 512 {
		preview = preview[:512]
	}

	heading = extract_first_heading(preview)
	if heading == "" {
		return "", "", ""
	}

	matches = title_pattern.FindStringSubmatch(heading)
	if len(matches) == 4 {
		return matches[1], matches[2], matches[3]
	}

	matches = short_title_pattern.FindStringSubmatch(heading)
	if len(matches) == 3 {
		return matches[1], "", matches[2]
	}

	return "", "", heading
}

func extract_first_heading(data string) string {
	var lines []string
	var first string
	var second string
	var position int
	var hashes string
	var candidate string

	lines = strings.Split(data, "\n")
	if len(lines) == 0 {
		return ""
	}

	first = strings.TrimRight(lines[0], "\r")
	if strings.HasPrefix(first, "#") {
		position = 0
		for position < len(first) && first[position] == '#' {
			position++
		}
		hashes = first[:position]
		if len(hashes) == 1 && position < len(first) && first[position] == ' ' {
			candidate = strings.TrimSpace(first[position+1:])
			candidate = strings.TrimRight(candidate, "# ")
			return strings.TrimSpace(candidate)
		}
	}

	if len(lines) < 2 {
		return ""
	}

	second = strings.TrimRight(lines[1], "\r")
	if second != "" && strings.Trim(second, "=") == "" {
		return strings.TrimSpace(first)
	}

	return ""
}

func (document *Document) PathName() string {
	var basename string
	var offset int

	basename = document.basename
	if basename == "" {
		return ""
	}

	offset = strings.Index(basename, ".")
	if offset < 0 {
		return basename
	}

	return basename[:offset]
}

func (document *Document) PathSection() string {
	var matches []string
	var pattern *regexp.Regexp

	if document.basename == "" {
		return ""
	}

	pattern = regexp.MustCompile(`\.(\d\w*)\.`)
	matches = pattern.FindStringSubmatch(document.basename)
	if len(matches) != 2 {
		return ""
	}

	return matches[1]
}

func (document *Document) NameValue() string {
	if document.Name != "" {
		return document.Name
	}

	return document.PathName()
}

func (document *Document) SectionValue() string {
	if document.Section != "" {
		return document.Section
	}

	return document.PathSection()
}

func (document *Document) ReferenceName() string {
	var name string
	var section string

	name = document.NameValue()
	section = document.SectionValue()
	if section == "" {
		return name
	}

	return fmt.Sprintf("%s(%s)", name, section)
}

func (document *Document) TitleMode() bool {
	return !document.name_set && document.Tagline != ""
}

func (document *Document) Title() string {
	if document.TitleMode() {
		return document.Tagline
	}
	return ""
}

func (document *Document) DateValue() time.Time {
	var info os.FileInfo
	var err error

	if document.Date != nil {
		return *document.Date
	}

	if document.Path != "" && document.Path != "-" {
		info, err = os.Stat(document.Path)
		if err == nil {
			return info.ModTime()
		}
	}

	return time.Now()
}

func (document *Document) SetStyles(styles []string) {
	var values []string

	values = append([]string{"man"}, styles...)
	document.Styles = unique_styles(values)
}

func (document *Document) BaseName(format string) string {
	var parts []string
	var name string
	var section string

	if format == "" || format == "roff" {
		format = ""
	}

	parts = make([]string, 0, 3)
	name = document.PathName()
	if name == "" {
		name = document.NameValue()
	}
	section = document.PathSection()
	if section == "" {
		section = document.SectionValue()
	}

	if name != "" {
		parts = append(parts, name)
	}
	if section != "" {
		parts = append(parts, section)
	}
	if format != "" {
		parts = append(parts, format)
	}

	return strings.Join(parts, ".")
}

func (document *Document) PathFor(format string) string {
	var basename string
	var dir string

	basename = document.BaseName(format)
	if document.basename == "" {
		return basename
	}

	dir = filepath.Dir(document.Path)
	if dir == "." {
		return "./" + basename
	}

	return filepath.Join(dir, basename)
}

func (document *Document) Markdown() string {
	if document.markdown != "" {
		return document.markdown
	}

	document.markdown = document.process_markdown()
	return document.markdown
}

func (document *Document) TOC() []SectionHead {
	var err error

	if document.html_fragment == "" {
		err = document.process_html()
		if err != nil {
			return nil
		}
	}

	return document.toc
}

func (document *Document) Convert(format string) (string, error) {
	if format == "roff" {
		return document.ToRoff()
	}
	if format == "html" {
		return document.ToHTML()
	}
	if format == "html_fragment" {
		return document.ToHTMLFragment()
	}
	if format == "markdown" {
		return document.Markdown(), nil
	}

	return "", fmt.Errorf("unknown format: %s", format)
}

func (document *Document) ToHTMLFragment() (string, error) {
	var body string
	var err error

	body, err = document.html_fragment_body()
	if err != nil {
		return "", err
	}

	return "<div class='mp'>\n" + body + "\n</div>", nil
}

func (document *Document) ToHTML() (string, error) {
	var body string
	var err error

	body, err = document.html_fragment_body()
	if err != nil {
		return "", err
	}

	return RenderHTMLPage(document, body)
}

func (document *Document) ToRoff() (string, error) {
	var body string
	var renderer *RoffRenderer
	var output string
	var err error

	body, err = document.html_fragment_body()
	if err != nil {
		return "", err
	}

	renderer = NewRoffRenderer(body, document.NameValue(), document.SectionValue(), document.Tagline, document.Manual, document.Organization, document.DateValue())
	output, err = renderer.Render()
	if err != nil {
		return "", err
	}

	return output, nil
}

func (document *Document) ToMap() map[string]interface{} {
	var toc [][]string
	var heads []SectionHead
	var index int
	var result map[string]interface{}

	heads = document.TOC()
	toc = make([][]string, 0, len(heads))

	for index = 0; index < len(heads); index++ {
		toc = append(toc, []string{heads[index].ID, heads[index].Text})
	}

	result = make(map[string]interface{})
	result["name"] = document.NameValue()
	result["section"] = document.SectionValue()
	result["tagline"] = document.Tagline
	result["manual"] = document.Manual
	result["organization"] = document.Organization
	result["date"] = document.DateValue()
	result["styles"] = document.Styles
	result["toc"] = toc
	return result
}

func (document *Document) ToJSON() (string, error) {
	var data map[string]interface{}
	var buffer []byte
	var err error

	data = document.ToMap()
	data["date"] = document.DateValue().Format(time.RFC3339)
	buffer, err = json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func (document *Document) ToYAML() (string, error) {
	var data map[string]interface{}
	var buffer []byte
	var err error

	data = document.ToMap()
	buffer, err = yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func (document *Document) html_fragment_body() (string, error) {
	var err error

	if document.html_fragment != "" {
		return document.html_fragment, nil
	}

	err = document.process_html()
	if err != nil {
		return "", err
	}

	return document.html_fragment, nil
}

func (document *Document) process_markdown() string {
	var markdown string

	markdown = document.Data
	markdown = document.markdown_filter_heading_anchors(markdown)
	markdown = document.markdown_filter_link_index(markdown)
	markdown = document.markdown_filter_angle_quotes(markdown)
	return markdown
}

func (document *Document) input_html() (string, error) {
	var parser goldmark.Markdown
	var buffer bytes.Buffer
	var err error
	var root *nethtml.Node

	parser = goldmark.New(
		goldmark.WithRendererOptions(goldhtml.WithUnsafe()),
	)

	err = parser.Convert([]byte(document.Markdown()), &buffer)
	if err != nil {
		return "", err
	}

	root, err = parse_html_root(buffer.String())
	if err != nil {
		return "", err
	}

	remove_first_heading(root)
	return render_children(root), nil
}

func parse_html_root(source string) (*nethtml.Node, error) {
	var root *nethtml.Node
	var nodes []*nethtml.Node
	var err error
	var index int

	root = &nethtml.Node{Type: nethtml.ElementNode, Data: "div"}
	root.DataAtom = atom.Div
	nodes, err = parse_fragment(root, source)
	if err != nil {
		return nil, err
	}

	for index = 0; index < len(nodes); index++ {
		root.AppendChild(nodes[index])
	}

	return root, nil
}

func remove_first_heading(root *nethtml.Node) {
	var heading *nethtml.Node

	heading = first_descendant_by_tag(root, "h1")
	if heading != nil {
		remove_node(heading)
	}
}

func (document *Document) process_html() error {
	var source string
	var root *nethtml.Node
	var err error

	source, err = document.input_html()
	if err != nil {
		return err
	}

	root, err = parse_html_root(source)
	if err != nil {
		return err
	}

	document.html_filter_angle_quotes(root)
	err = document.html_filter_definition_lists(root)
	if err != nil {
		return err
	}
	err = document.html_filter_inject_name_section(root)
	if err != nil {
		return err
	}
	document.html_filter_heading_anchors(root)
	document.html_filter_annotate_bare_links(root)
	err = document.html_filter_manual_reference_links(root)
	if err != nil {
		return err
	}
	document.toc = document.collect_toc(root)
	document.html_fragment = render_children(root)
	return nil
}

func (document *Document) collect_toc(root *nethtml.Node) []SectionHead {
	var toc []SectionHead

	toc = make([]SectionHead, 0)

	walk_nodes(root, func(node *nethtml.Node) bool {
		var id string
		var ok bool
		var text string

		if node.Type != nethtml.ElementNode || strings.ToLower(node.Data) != "h2" {
			return true
		}
		id, ok = get_attribute(node, "id")
		if !ok {
			return true
		}
		text = strings.TrimSpace(collect_text(node))
		toc = append(toc, SectionHead{ID: id, Text: text})
		return true
	})

	return toc
}

func (document *Document) markdown_filter_link_index(markdown string) string {
	var builder strings.Builder
	var position int
	var reference *Reference

	if document.Index == nil || document.Index.Empty() {
		return markdown
	}

	builder.WriteString(markdown)
	builder.WriteString("\n\n")

	for position = 0; position < len(document.Index.References); position++ {
		reference = document.Index.References[position]
		builder.WriteString("[")
		builder.WriteString(reference.Name)
		builder.WriteString("]: ")
		builder.WriteString(reference.URL())
		builder.WriteString("\n")
	}

	return builder.String()
}

func (document *Document) markdown_filter_heading_anchors(markdown string) string {
	var lines []string
	var builder strings.Builder
	var first bool
	var line string
	var index int
	var title string
	var anchor string

	lines = strings.Split(markdown, "\n")
	builder.WriteString(markdown)
	first = true

	for index = 0; index < len(lines); index++ {
		line = lines[index]
		if !markdown_heading_anchor_pattern.MatchString(line) {
			continue
		}
		if first {
			builder.WriteString("\n\n")
			first = false
		}
		title = markdown_heading_cleanup_pattern.ReplaceAllString(line, "")
		title = strings.TrimSpace(title)
		anchor = anchor_cleanup_pattern.ReplaceAllString(title, "-")
		anchor = anchor_trim_pattern.ReplaceAllString(anchor, "")
		builder.WriteString("[")
		builder.WriteString(title)
		builder.WriteString("]: #")
		builder.WriteString(anchor)
		builder.WriteString(" \"")
		builder.WriteString(title)
		builder.WriteString("\"\n")
	}

	return builder.String()
}

func (document *Document) markdown_filter_angle_quotes(markdown string) string {
	var output string

	output = angle_quote_pattern.ReplaceAllStringFunc(markdown, func(match string) string {
		var parts []string
		var contents string
		var tag string
		var attrs string

		contents = angle_quote_pattern.FindStringSubmatch(match)[1]
		parts = strings.SplitN(contents, " ", 2)
		tag = parts[0]
		if len(parts) == 2 {
			attrs = parts[1]
		}
		if strings.Contains(attrs, "/=") || has_html_element(strings.TrimPrefix(tag, "/")) || strings.Contains(document.Data, "</"+tag+">") {
			return match
		}
		return "<var>" + contents + "</var>"
	})

	output = literal_angle_pattern.ReplaceAllStringFunc(output, func(match string) string {
		var parts []string
		var contents string
		var tag string
		var attrs string

		contents = literal_angle_pattern.FindStringSubmatch(match)[1]
		parts = strings.SplitN(contents, " ", 2)
		tag = parts[0]
		if len(parts) == 2 {
			attrs = parts[1]
		}
		if strings.Contains(contents, "://") || strings.Contains(contents, "@") || strings.Contains(attrs, "/=") || has_html_element(strings.TrimPrefix(tag, "/")) || strings.Contains(document.Data, "</"+tag+">") {
			return match
		}
		if strings.ContainsAny(contents, ":.") {
			return "&lt;" + contents + ">"
		}
		return match
	})

	return output
}

func (document *Document) html_filter_angle_quotes(root *nethtml.Node) {
	var code_nodes []*nethtml.Node
	var index int
	var code *nethtml.Node

	code_nodes = make([]*nethtml.Node, 0)

	walk_nodes(root, func(node *nethtml.Node) bool {
		if node.Type == nethtml.ElementNode && strings.ToLower(node.Data) == "code" {
			code_nodes = append(code_nodes, node)
		}
		return true
	})

	for index = 0; index < len(code_nodes); index++ {
		code = code_nodes[index]
		walk_nodes(code, func(node *nethtml.Node) bool {
			if node.Type == nethtml.TextNode {
				node.Data = strings.ReplaceAll(node.Data, "<var>", "<")
				node.Data = strings.ReplaceAll(node.Data, "</var>", ">")
				node.Data = strings.ReplaceAll(node.Data, "&lt;", "<")
				node.Data = strings.ReplaceAll(node.Data, "&gt;", ">")
			}
			return true
		})
	}
}

func (document *Document) html_filter_definition_lists(root *nethtml.Node) error {
	var ul_nodes []*nethtml.Node
	var index int
	var ul *nethtml.Node
	var items []*nethtml.Node
	var item *nethtml.Node
	var item_index int
	var container *nethtml.Node
	var html_source string
	var parts []string
	var dt *nethtml.Node
	var err error

	ul_nodes = make([]*nethtml.Node, 0)

	walk_nodes(root, func(node *nethtml.Node) bool {
		if node.Type == nethtml.ElementNode && strings.ToLower(node.Data) == "ul" {
			ul_nodes = append(ul_nodes, node)
		}
		return true
	})

	for index = len(ul_nodes) - 1; index >= 0; index-- {
		ul = ul_nodes[index]
		items = direct_children_by_tag(ul, "li")
		if len(items) == 0 {
			continue
		}
		for item_index = 0; item_index < len(items); item_index++ {
			item = items[item_index]
			container = first_descendant_by_tag(item, "p")
			if container == nil {
				container = item
			}
			html_source = render_children(container)
			if !strings.Contains(html_source, ":\n") {
				items = nil
				break
			}
		}
		if items == nil {
			continue
		}

		rename_element(ul, "dl")

		for item_index = 0; item_index < len(items); item_index++ {
			item = items[item_index]
			container = first_descendant_by_tag(item, "p")
			if container == nil {
				container = item
			}
			html_source = render_children(container)
			parts = strings.SplitN(html_source, ":\n", 2)
			if len(parts) != 2 {
				continue
			}
			dt, err = insert_before_html(item, "<dt>"+parts[0]+"</dt>", ul)
			if err != nil {
				return err
			}
			if dt != nil && len(strings.TrimSpace(collect_text(dt))) <= 7 {
				append_class(dt, "flush")
			}
			rename_element(item, "dd")
			if container == item {
				err = set_inner_html(item, parts[1])
			} else {
				err = replace_node_with_html(container, "<p>"+parts[1]+"</p>", item)
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (document *Document) html_filter_inject_name_section(root *nethtml.Node) error {
	var markup string
	var first *nethtml.Node
	var nodes []*nethtml.Node
	var err error
	var index int

	if document.TitleMode() {
		markup = "<h1>" + escape_html_text(document.Title()) + "</h1>"
	} else if document.NameValue() != "" {
		markup = "<h2>NAME</h2>\n" +
			"<p class='man-name'>\n  <code>" + escape_html_text(document.NameValue()) + "</code>"
		if document.Tagline != "" {
			markup = markup + " - <span class='man-whatis'>" + escape_html_text(document.Tagline) + "</span>\n"
		} else {
			markup = markup + "\n"
		}
		markup = markup + "</p>\n"
	}

	if markup == "" {
		return nil
	}

	first = first_element_child(root)
	if first == nil {
		nodes, err = parse_fragment(root, markup)
		if err != nil {
			return err
		}
		for index = 0; index < len(nodes); index++ {
			root.AppendChild(nodes[index])
		}
		return nil
	}

	_, err = insert_before_html(first, markup, root)
	return err
}

func (document *Document) html_filter_heading_anchors(root *nethtml.Node) {
	walk_nodes(root, func(node *nethtml.Node) bool {
		var lower string
		var ok bool
		var text string

		_, ok = get_attribute(node, "id")

		if node.Type != nethtml.ElementNode {
			return true
		}
		lower = strings.ToLower(node.Data)
		if lower != "h2" && lower != "h3" && lower != "h4" && lower != "h5" && lower != "h6" {
			return true
		}
		if ok {
			return true
		}
		text = strings.TrimSpace(collect_text(node))
		set_attribute(node, "id", anchor_cleanup_pattern.ReplaceAllString(text, "-"))
		return true
	})
}

func (document *Document) html_filter_annotate_bare_links(root *nethtml.Node) {
	walk_nodes(root, func(node *nethtml.Node) bool {
		var href string
		var ok bool
		var text string

		if node.Type != nethtml.ElementNode || strings.ToLower(node.Data) != "a" {
			return true
		}
		href, ok = get_attribute(node, "href")
		if !ok {
			return true
		}
		text = collect_text(node)
		if href == text || strings.HasPrefix(href, "#") || unescape_html_text(href) == "mailto:"+unescape_html_text(text) {
			set_attribute(node, "data-bare-link", "true")
		}
		return true
	})
}

func (document *Document) html_filter_manual_reference_links(root *nethtml.Node) error {
	var text_nodes []*nethtml.Node
	var code_nodes []*nethtml.Node
	var node *nethtml.Node
	var index int
	var result string
	var err error
	var name string
	var section string
	var sibling *nethtml.Node
	var matches []string

	text_nodes = make([]*nethtml.Node, 0)
	code_nodes = make([]*nethtml.Node, 0)

	walk_nodes(root, func(current *nethtml.Node) bool {
		if current.Type == nethtml.TextNode {
			text_nodes = append(text_nodes, current)
		}
		if current.Type == nethtml.ElementNode && strings.ToLower(current.Data) == "code" {
			code_nodes = append(code_nodes, current)
		}
		return true
	})

	for index = 0; index < len(text_nodes); index++ {
		node = text_nodes[index]
		if !strings.Contains(node.Data, ")") {
			continue
		}
		if node.Parent == nil || skip_manual_reference_parent(node.Parent) || child_of(node, "a") {
			continue
		}
		result = document.replace_manual_references_in_text(node.Data)
		if result == "" || result == escape_html_text(node.Data) {
			continue
		}
		err = replace_node_with_html(node, result, node.Parent)
		if err != nil {
			return err
		}
	}

	for index = 0; index < len(code_nodes); index++ {
		node = code_nodes[index]
		if node.Parent == nil || skip_manual_reference_parent(node.Parent) || child_of(node, "a") {
			continue
		}
		name = collect_text(node)
		if !manual_reference_name_pattern.MatchString(name) {
			continue
		}
		sibling = node.NextSibling
		if sibling == nil || sibling.Type != nethtml.TextNode {
			continue
		}
		matches = manual_reference_section_pattern.FindStringSubmatch(sibling.Data)
		if len(matches) != 2 {
			continue
		}
		section = "(" + matches[1] + ")"
		err = replace_node_with_html(node, document.html_build_manual_reference_link(render_node(node), name, section), node.Parent)
		if err != nil {
			return err
		}
		sibling.Data = manual_reference_section_pattern.ReplaceAllString(sibling.Data, "")
	}

	return nil
}

func skip_manual_reference_parent(parent *nethtml.Node) bool {
	var name string

	if parent == nil || parent.Type != nethtml.ElementNode {
		return false
	}

	name = strings.ToLower(parent.Data)
	return name == "pre" || name == "code" || name == "h1" || name == "h2" || name == "h3"
}

func (document *Document) replace_manual_references_in_text(text string) string {
	var indices [][]int
	var builder strings.Builder
	var last int
	var index int
	var match []int
	var name string
	var section string

	indices = manual_reference_pattern.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return ""
	}

	last = 0
	for index = 0; index < len(indices); index++ {
		match = indices[index]
		builder.WriteString(escape_html_text(text[last:match[0]]))
		name = text[match[2]:match[3]]
		section = text[match[4]:match[5]]
		builder.WriteString(document.html_build_manual_reference_link(escape_html_text(name), name, section))
		last = match[1]
	}
	builder.WriteString(escape_html_text(text[last:]))
	return builder.String()
}

func (document *Document) html_build_manual_reference_link(label string, name string, section string) string {
	var reference *Reference

	if document.Index != nil {
		reference = document.Index.Get(name + section)
	}
	if reference != nil {
		return "<a class='man-ref' href='" + escape_html_text(reference.URL()) + "'>" + label + "<span class='s'>" + escape_html_text(section) + "</span></a>"
	}

	return "<span class='man-ref'>" + label + "<span class='s'>" + escape_html_text(section) + "</span></span>"
}
