package gonn

import "fmt"
import "html"
import "net/http"
import "path/filepath"
import "strings"
import "time"

type Server struct {
	files map[string]string
	options DocumentOptions
}

func NewServer(files []string, options DocumentOptions) (*Server, error) {
	var server *Server
	var mapping map[string]string
	var index int
	var file string
	var basename string

	if len(files) == 0 {
		return nil, fmt.Errorf("no files")
	}

	mapping = make(map[string]string)
	for index = 0; index < len(files); index++ {
		file = files[index]
		basename = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		mapping[basename] = file
	}

	server = &Server{
		files: mapping,
		options: options,
	}

	return server, nil
}

func (server *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var path string
	var basename string
	var file string
	var options DocumentOptions
	var document *Document
	var output string
	var err error

	path = request.URL.Path

	if path == "/" {
		server.render_index(writer)
		return
	}

	if strings.HasSuffix(path, ".html") {
		basename = strings.TrimSuffix(strings.TrimPrefix(path, "/"), ".html")
		file = server.files[basename]
		if file == "" {
			http.NotFound(writer, request)
			return
		}
		options = server.request_options(request)
		document, err = NewDocument(file, options)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		output, err = document.ToHTML()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(writer, output)
		return
	}

	if strings.HasSuffix(path, ".roff") {
		basename = strings.TrimSuffix(strings.TrimPrefix(path, "/"), ".roff")
		file = server.files[basename]
		if file == "" {
			http.NotFound(writer, request)
			return
		}
		document, err = NewDocument(file, server.options)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		output, err = document.ToRoff()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(writer, output)
		return
	}

	http.NotFound(writer, request)
}

func (server *Server) render_index(writer http.ResponseWriter) {
	var names []string
	var name string
	var builder strings.Builder
	var index int

	names = make([]string, 0, len(server.files))
	for name = range server.files {
		names = append(names, name)
	}

	builder.WriteString("<ul>")
	for index = 0; index < len(names); index++ {
		name = names[index]
		builder.WriteString("<li><a href='./")
		builder.WriteString(html.EscapeString(name))
		builder.WriteString(".html'>")
		builder.WriteString(html.EscapeString(name))
		builder.WriteString("</a></li>")
	}
	builder.WriteString("</ul>")

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(writer, builder.String())
}

func (server *Server) request_options(request *http.Request) DocumentOptions {
	var options DocumentOptions
	var styles []string
	var values []string
	var value string
	var parsed time.Time
	var err error
	var index int

	options = server.options
	styles = make([]string, 0)

	values = request.URL.Query()["styles"]
	for index = 0; index < len(values); index++ {
		value = values[index]
		styles = append(styles, split_style_values(value)...)
	}

	values = request.URL.Query()["style"]
	for index = 0; index < len(values); index++ {
		value = values[index]
		styles = append(styles, split_style_values(value)...)
	}

	if len(styles) != 0 {
		options.Styles = styles
	}

	value = request.URL.Query().Get("manual")
	if value != "" {
		options.Manual = value
	}

	value = request.URL.Query().Get("organization")
	if value != "" {
		options.Organization = value
	}

	value = request.URL.Query().Get("date")
	if value != "" {
		parsed, err = time.Parse("2006-01-02", value)
		if err == nil {
			options.Date = &parsed
		}
	}

	return options
}

func split_style_values(value string) []string {
	var fields []string
	var replaced string

	replaced = strings.ReplaceAll(value, ",", " ")
	fields = strings.Fields(replaced)
	return fields
}

func RunServer(files []string, options DocumentOptions, address string) error {
	var server *Server
	var err error

	if address == "" {
		address = "0.0.0.0:1207"
	}

	server, err = NewServer(files, options)
	if err != nil {
		return err
	}

	return http.ListenAndServe(address, server)
}
