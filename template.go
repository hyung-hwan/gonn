package gonn

import "bytes"
import "embed"
import "fmt"
import "html/template"
import "os"
import "path/filepath"
import "strings"

//go:embed templates/*.css
var template_files embed.FS

type stylesheet_info struct {
	Name string
	Base string
	Media string
	Data string
}

type template_data struct {
	Generator string
	Title string
	PageName string
	Manual string
	Organization string
	Date string
	HTML template.HTML
	StylesheetTags template.HTML
	SectionHeads []SectionHead
}

const default_layout string = `<!DOCTYPE html>
<html>
<head>
  <meta http-equiv='content-type' value='text/html;charset=utf8'>
  <meta name='generator' value='{{ .Generator }}'>
  <title>{{ .Title }}</title>
  {{ .StylesheetTags }}
</head>
<!--
  The following styles are deprecated and will be removed at some point:
  div#man, div#man ol.man, div#man ol.head, div#man ol.man.

  The .man-page, .man-decor, .man-head, .man-foot, .man-title, and
  .man-navigation should be used instead.
-->
<body id='manpage'>
  <div class='mp' id='man'>

  <div class='man-navigation' style='display:none'>
    {{ range .SectionHeads }}<a href="#{{ .ID }}">{{ .Text }}</a>
    {{ end }}
  </div>

  <ol class='man-decor man-head man head'>
    <li class='tl'>{{ .PageName }}</li>
    <li class='tc'>{{ .Manual }}</li>
    <li class='tr'>{{ .PageName }}</li>
  </ol>

  {{ .HTML }}

  <ol class='man-decor man-foot man foot'>
    <li class='tl'>{{ .Organization }}</li>
    <li class='tc'>{{ .Date }}</li>
    <li class='tr'>{{ .PageName }}</li>
  </ol>

  </div>
</body>
</html>
`

func RenderHTMLPage(document *Document, body string) (string, error) {
	var buffer bytes.Buffer
	var parsed *template.Template
	var err error
	var data template_data
	var title string
	var page_name string
	var generator string
	var tags string

	title = template_title(document)
	page_name = template_page_name(document)
	generator = fmt.Sprintf("Gonn/v%s (https://github.com/hyung-hwan/gonn/tree/%s)", Version(), Revision)
	tags, err = stylesheet_tags(document)
	if err != nil {
		return "", err
	}

	data = template_data{
		Generator: generator,
		Title: title,
		PageName: page_name,
		Manual: document.Manual,
		Organization: document.Organization,
		Date: document.DateValue().Format("January 2006"),
		HTML: template.HTML(body),
		StylesheetTags: template.HTML(tags),
		SectionHeads: document.TOC(),
	}

	parsed, err = template.New("default").Parse(default_layout)
	if err != nil {
		return "", err
	}

	err = parsed.Execute(&buffer, data)
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func template_title(document *Document) string {
	var page_name string

	if document.TitleMode() && document.Tagline != "" {
		return document.Tagline
	}

	page_name = template_page_name(document)
	if page_name == "" {
		return document.Tagline
	}
	if document.Tagline == "" {
		return page_name
	}

	return page_name + " - " + document.Tagline
}

func template_page_name(document *Document) string {
	var name string
	var section string

	name = document.NameValue()
	section = document.SectionValue()
	if section == "" {
		return name
	}
	return fmt.Sprintf("%s(%s)", name, section)
}

func stylesheet_tags(document *Document) (string, error) {
	var styles []stylesheet_info
	var tags []string
	var index int
	var err error

	styles, err = collect_stylesheets(document)
	if err != nil {
		return "", err
	}

	tags = make([]string, 0, len(styles))
	for index = 0; index < len(styles); index++ {
		tags = append(tags, inline_stylesheet_tag(styles[index]))
	}

	return strings.Join(tags, "\n  "), nil
}

func collect_stylesheets(document *Document) ([]stylesheet_info, error) {
	var styles []stylesheet_info
	var style_names []string
	var index int
	var name string
	var data string
	var err error
	var info stylesheet_info
	var base string
	var media string

	style_names = document.Styles
	styles = make([]stylesheet_info, 0, len(style_names))

	for index = 0; index < len(style_names); index++ {
		name = style_names[index]
		data, err = load_stylesheet_data(name)
		if err != nil {
			return nil, err
		}
		base = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		media = "all"
		if strings.HasSuffix(base, "print") {
			media = "print"
		}
		if strings.HasSuffix(base, "screen") {
			media = "screen"
		}
		info = stylesheet_info{
			Name: name,
			Base: base,
			Media: media,
			Data: minify_css(data),
		}
		styles = append(styles, info)
	}

	return styles, nil
}

func load_stylesheet_data(name string) (string, error) {
	var search_path string
	var candidate string
	var data []byte
	var err error
	var paths []string
	var index int
	var embedded_path string

	if strings.Contains(name, "/") || strings.HasSuffix(name, ".css") {
		data, err = os.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	search_path = os.Getenv("GONN_STYLE")
	paths = strings.Split(search_path, ":")
	for index = 0; index < len(paths); index++ {
		if strings.TrimSpace(paths[index]) == "" {
			continue
		}
		candidate = filepath.Join(paths[index], name+".css")
		data, err = os.ReadFile(candidate)
		if err == nil {
			return string(data), nil
		}
	}

	embedded_path = "templates/" + name + ".css"
	data, err = template_files.ReadFile(embedded_path)
	if err != nil {
		return "", fmt.Errorf("style not found: %s", name)
	}

	return string(data), nil
}

func inline_stylesheet_tag(info stylesheet_info) string {
	return "<style type='text/css' media='" + info.Media + "'>\n" +
		"/* style: " + info.Base + " */\n" +
		info.Data + "\n" +
		"</style>"
}
