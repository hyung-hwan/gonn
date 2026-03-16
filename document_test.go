package gonn

import "html"
import "os"
import "path/filepath"
import "regexp"
import "strings"
import "testing"
import "time"

var leading_space_pattern *regexp.Regexp = regexp.MustCompile(`(?m)^ +`)
var literal_closing_tag_pattern *regexp.Regexp = regexp.MustCompile(`</[^>]*[.:][^>]*>`)
var pre_open_space_pattern *regexp.Regexp = regexp.MustCompile(`<pre> +`)
var whitespace_pattern *regexp.Regexp = regexp.MustCompile(`\s+`)
var html_fixture_allowlist map[string]struct{} = map[string]struct{}{
	"angle_bracket_syntax.gonn": {},
	"basic_document.gonn": {},
	"definition_list_syntax.gonn": {},
	"section_reference_links.gonn": {},
	"titleless_document.gonn": {},
}
var roff_fixture_allowlist map[string]struct{} = map[string]struct{}{
	"angle_bracket_syntax.gonn": {},
	"definition_list_syntax.gonn": {},
	"section_reference_links.gonn": {},
}

func fixture_root(t *testing.T) string {
	var root string
	var err error

	root, err = filepath.Abs("./test")
	if err != nil {
		t.Fatalf("fixture root: %v", err)
	}

	return root
}

func gonn_ng_fixture_root(t *testing.T) string {
	var root string
	var err error

	root, err = filepath.Abs("./test")
	if err != nil {
		t.Fatalf("gonn-ng fixture root: %v", err)
	}

	return root
}

func canonicalize_html_text(text string) string {
	var output string

	output = html.UnescapeString(text)
	output = leading_space_pattern.ReplaceAllString(output, "")
	output = strings.ReplaceAll(output, "\n", "")
	output = literal_closing_tag_pattern.ReplaceAllString(output, "")
	output = pre_open_space_pattern.ReplaceAllString(output, "<pre>")
	output = whitespace_pattern.ReplaceAllString(output, " ")
	output = strings.ReplaceAll(output, "\"", "'")
	output = strings.TrimSpace(output)
	return output
}

func strip_roff_prelude(text string) string {
	var parts []string
	var index int
	var output []string
	var start int
	var line string

	parts = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(parts) <= 3 {
		start = 0
	} else if strings.HasPrefix(parts[0], ".\\\" generated with Gonn/") {
		start = 3
	} else {
		start = 0
	}

	output = make([]string, 0, len(parts)-start)
	for index = start; index < len(parts); index++ {
		line = strings.TrimRight(parts[index], " \t")
		if index == start && strings.HasPrefix(line, ".TH ") {
			continue
		}
		if line == "." {
			continue
		}
		output = append(output, line)
	}

	return strings.TrimRight(strings.Join(output, "\n"), "\n")
}

func strip_roff_title(text string) string {
	var output string
	var lines []string

	output = strip_roff_prelude(text)
	lines = strings.Split(output, "\n")
	if len(lines) == 0 {
		return output
	}
	if !strings.HasPrefix(lines[0], ".TH ") {
		return output
	}

	return strings.TrimRight(strings.Join(lines[1:], "\n"), "\n")
}

func compare_gonn_ng_html_fixture(t *testing.T, name string) {
	var root string
	var source string
	var expected_path string
	var expected []byte
	var document *Document
	var output string
	var err error

	root = gonn_ng_fixture_root(t)
	source = filepath.Join(root, name+".gonn")
	expected_path = filepath.Join(root, name+".html")
	expected, err = os.ReadFile(expected_path)
	if err != nil {
		t.Fatalf("%s: read expected html: %v", name, err)
	}

	document, err = NewDocument(source, DocumentOptions{})
	if err != nil {
		t.Fatalf("%s: new document: %v", name, err)
	}
	output, err = document.ToHTMLFragment()
	if err != nil {
		t.Fatalf("%s: render html: %v", name, err)
	}

	if canonicalize_html_text(output) != canonicalize_html_text(string(expected)) {
		t.Fatalf("%s: html mismatch\nexpected: %s\nactual:   %s", name, canonicalize_html_text(string(expected)), canonicalize_html_text(output))
	}
}

func compare_gonn_ng_roff_fixture(t *testing.T, name string) {
	var root string
	var source string
	var expected_path string
	var expected []byte
	var document *Document
	var output string
	var err error
	var date time.Time

	root = gonn_ng_fixture_root(t)
	source = filepath.Join(root, name+".gonn")
	expected_path = filepath.Join(root, name+".roff")
	expected, err = os.ReadFile(expected_path)
	if err != nil {
		t.Fatalf("%s: read expected roff: %v", name, err)
	}

	date = time.Date(1979, time.January, 1, 0, 0, 0, 0, time.UTC)
	document, err = NewDocument(source, DocumentOptions{Date: &date})
	if err != nil {
		t.Fatalf("%s: new document: %v", name, err)
	}
	output, err = document.ToRoff()
	if err != nil {
		t.Fatalf("%s: render roff: %v", name, err)
	}

	if strip_roff_title(output) != strip_roff_title(string(expected)) {
		t.Fatalf("%s: roff mismatch\nexpected:\n%s\nactual:\n%s", name, strip_roff_title(string(expected)), strip_roff_title(output))
	}
}

func TestDocumentPaths(t *testing.T) {
	var text string
	var reader func(string) (string, error)
	var document *Document
	var err error

	text = "# hello(1) -- hello world"
	reader = func(path string) (string, error) {
		return text, nil
	}

	document, err = NewDocument("hello.1.gonn", DocumentOptions{Reader: reader})
	if err != nil {
		t.Fatalf("new document: %v", err)
	}

	if document.NameValue() != "hello" {
		t.Fatalf("name mismatch: %q", document.NameValue())
	}
	if document.SectionValue() != "1" {
		t.Fatalf("section mismatch: %q", document.SectionValue())
	}
	if document.PathFor("html") != "./hello.1.html" {
		t.Fatalf("html path mismatch: %q", document.PathFor("html"))
	}
	if document.PathFor("roff") != "./hello.1" {
		t.Fatalf("roff path mismatch: %q", document.PathFor("roff"))
	}
}

func TestHTMLFixtures(t *testing.T) {
	var root string
	var entries []os.DirEntry
	var err error
	var index int
	var entry os.DirEntry
	var source string
	var expected_path string
	var expected []byte
	var document *Document
	var output string
	var ok bool

	root = fixture_root(t)
	entries, err = os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	for index = 0; index < len(entries); index++ {
		entry = entries[index]
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".gonn" {
			continue
		}
		_, ok = html_fixture_allowlist[entry.Name()]
		if !ok {
			continue
		}
		source = filepath.Join(root, entry.Name())
		expected_path = strings.TrimSuffix(source, ".gonn") + ".html"
		expected, err = os.ReadFile(expected_path)
		if err != nil {
			continue
		}

		document, err = NewDocument(source, DocumentOptions{})
		if err != nil {
			t.Fatalf("%s: new document: %v", entry.Name(), err)
		}
		output, err = document.ToHTMLFragment()
		if err != nil {
			t.Fatalf("%s: render html: %v", entry.Name(), err)
		}

		if canonicalize_html_text(output) != canonicalize_html_text(string(expected)) {
			t.Fatalf("%s: html mismatch\nexpected: %s\nactual:   %s", entry.Name(), canonicalize_html_text(string(expected)), canonicalize_html_text(output))
		}
	}
}

func TestCodeBlocksDoNotDoubleEscapeAngles(t *testing.T) {
	var text string
	var reader func(string) (string, error)
	var document *Document
	var output string
	var err error

	text = "# hello(1) -- hello world\n\n```c\n#include <hawk.h>\n```\n"
	reader = func(path string) (string, error) {
		return text, nil
	}

	document, err = NewDocument("hello.1.gonn", DocumentOptions{Reader: reader})
	if err != nil {
		t.Fatalf("new document: %v", err)
	}

	output, err = document.ToHTMLFragment()
	if err != nil {
		t.Fatalf("render html: %v", err)
	}

	if !strings.Contains(output, "&lt;hawk.h&gt;") {
		t.Fatalf("missing escaped include: %s", output)
	}
	if strings.Contains(output, "&amp;lt;hawk.h") {
		t.Fatalf("double-escaped include: %s", output)
	}
}

func TestGonnNgCodeBlocksHTML(t *testing.T) {
	compare_gonn_ng_html_fixture(t, "code_blocks")
}

func TestGonnNgCodeBlocksRoff(t *testing.T) {
	compare_gonn_ng_roff_fixture(t, "code_blocks")
}

func TestGonnNgCodeBlocksRegressionHTML(t *testing.T) {
	compare_gonn_ng_html_fixture(t, "code_blocks_regression")
}

func TestGonnNgOrderedListHTML(t *testing.T) {
	compare_gonn_ng_html_fixture(t, "ordered_list")
}

func TestGonnNgOrderedListRoff(t *testing.T) {
	compare_gonn_ng_roff_fixture(t, "ordered_list")
}

func TestGonnNgSingleQuotesHTML(t *testing.T) {
	compare_gonn_ng_html_fixture(t, "single_quotes")
}

func TestGonnNgSingleQuotesRoff(t *testing.T) {
	compare_gonn_ng_roff_fixture(t, "single_quotes")
}

func TestGonnNgTablesHTMLAndRoff(t *testing.T) {
	var root string
	var source string
	var document *Document
	var html_output string
	var roff_output string
	var err error
	var date time.Time

	root = gonn_ng_fixture_root(t)
	source = filepath.Join(root, "tables.gonn")
	document, err = NewDocument(source, DocumentOptions{})
	if err != nil {
		t.Fatalf("tables: new document: %v", err)
	}
	html_output, err = document.ToHTMLFragment()
	if err != nil {
		t.Fatalf("tables: render html: %v", err)
	}

	if !strings.Contains(html_output, "<table>") {
		t.Fatalf("tables: missing table element: %s", html_output)
	}
	if !strings.Contains(html_output, "<thead>") {
		t.Fatalf("tables: missing thead element: %s", html_output)
	}
	if !strings.Contains(html_output, "<tbody>") {
		t.Fatalf("tables: missing tbody element: %s", html_output)
	}
	if !strings.Contains(html_output, "<code>some code</code>") {
		t.Fatalf("tables: missing inline code markup: %s", html_output)
	}
	if !strings.Contains(html_output, "<em>foo</em>") || !strings.Contains(html_output, "<em>bar</em>") {
		t.Fatalf("tables: missing inline emphasis markup: %s", html_output)
	}

	date = time.Date(1979, time.January, 1, 0, 0, 0, 0, time.UTC)
	document, err = NewDocument(source, DocumentOptions{Date: &date})
	if err != nil {
		t.Fatalf("tables: new roff document: %v", err)
	}
	roff_output, err = document.ToRoff()
	if err != nil {
		t.Fatalf("tables: render roff: %v", err)
	}

	if !strings.Contains(roff_output, ".TS") {
		t.Fatalf("tables: missing .TS macro:\n%s", roff_output)
	}
	if !strings.Contains(roff_output, "allbox;") {
		t.Fatalf("tables: missing allbox directive:\n%s", roff_output)
	}
	if !strings.Contains(roff_output, "l c r.") {
		t.Fatalf("tables: missing alignment row:\n%s", roff_output)
	}
	if !strings.Contains(roff_output, "Syntax\tDescription\tTest Text With A Long Header Name") {
		t.Fatalf("tables: missing header row:\n%s", roff_output)
	}
	if !strings.Contains(roff_output, "Code: \\fBsome code\\fR") {
		t.Fatalf("tables: missing inline code rendering:\n%s", roff_output)
	}
	if !strings.Contains(roff_output, "Emphasis: \\fIfoo\\fR and \\fIbar\\fR") {
		t.Fatalf("tables: missing inline emphasis rendering:\n%s", roff_output)
	}
}

func TestRoffFixtures(t *testing.T) {
	var root string
	var entries []os.DirEntry
	var err error
	var index int
	var entry os.DirEntry
	var source string
	var expected_path string
	var expected []byte
	var document *Document
	var output string
	var date time.Time
	var ok bool

	root = fixture_root(t)
	entries, err = os.ReadDir(root)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	date = time.Date(1979, time.January, 1, 0, 0, 0, 0, time.UTC)

	for index = 0; index < len(entries); index++ {
		entry = entries[index]
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".gonn" {
			continue
		}
		_, ok = roff_fixture_allowlist[entry.Name()]
		if !ok {
			continue
		}
		source = filepath.Join(root, entry.Name())
		expected_path = strings.TrimSuffix(source, ".gonn") + ".roff"
		expected, err = os.ReadFile(expected_path)
		if err != nil {
			continue
		}

		document, err = NewDocument(source, DocumentOptions{Date: &date})
		if err != nil {
			t.Fatalf("%s: new document: %v", entry.Name(), err)
		}
		output, err = document.ToRoff()
		if err != nil {
			t.Fatalf("%s: render roff: %v", entry.Name(), err)
		}

		if strip_roff_prelude(output) != strip_roff_prelude(string(expected)) {
			t.Fatalf("%s: roff mismatch\nexpected:\n%s\nactual:\n%s", entry.Name(), strip_roff_prelude(string(expected)), strip_roff_prelude(output))
		}
	}
}
