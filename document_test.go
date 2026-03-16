package gonn

import "html"
import "os"
import "path/filepath"
import "regexp"
import "strings"
import "testing"
import "time"

var leading_space_pattern *regexp.Regexp = regexp.MustCompile(`(?m)^ +`)
var whitespace_pattern *regexp.Regexp = regexp.MustCompile(`\s+`)
var html_fixture_allowlist map[string]struct{} = map[string]struct{}{
	"angle_bracket_syntax.gonn": {},
	"basic_document.gonn": {},
	"custom_title_document.gonn": {},
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

	root, err = filepath.Abs("../test")
	if err != nil {
		t.Fatalf("fixture root: %v", err)
	}

	return root
}

func canonicalize_html_text(text string) string {
	var output string

	output = html.UnescapeString(text)
	output = leading_space_pattern.ReplaceAllString(output, "")
	output = strings.ReplaceAll(output, "\n", "")
	output = whitespace_pattern.ReplaceAllString(output, " ")
	output = strings.ReplaceAll(output, "\"", "'")
	output = strings.TrimSpace(output)
	return output
}

func strip_roff_prelude(text string) string {
	var parts []string
	var index int
	var output []string

	parts = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(parts) <= 3 {
		return strings.TrimRight(strings.Join(parts, "\n"), "\n")
	}
	if !strings.HasPrefix(parts[0], ".\\\" generated with Gonn/") {
		return strings.TrimRight(strings.Join(parts, "\n"), "\n")
	}

	output = make([]string, 0, len(parts)-3)
	for index = 3; index < len(parts); index++ {
		output = append(output, strings.TrimRight(parts[index], " \t"))
	}

	return strings.TrimRight(strings.Join(output, "\n"), "\n")
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
