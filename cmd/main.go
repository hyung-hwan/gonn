package main

import "fmt"
import "os"
import "os/exec"
import "strings"
import "time"
import "github.com/hyung-hwan/gonn"

type cli_config struct {
	build bool
	view bool
	server bool
	server_address string
	write_index bool
	formats []string
	styles []string
	options gonn.DocumentOptions
	groff string
	pager string
}

const usage_text string = `Usage: gonn <options> <file>...
       gonn -m|--man <file>
       gonn -S [address] <file> ...
       gonn --pipe [<file>...]

Convert gonn source <file>s to roff or HTML manpage. In the first synopsis form,
build HTML and roff output files based on the input file names.

Mode options alter the default behavior of generating files:
  --pipe                write to standard output instead of generating files
  -m, --man                 show manual like with man(1)
  -S, --server[=ADDRESS]    serve <file>s using the optional bind address
                            (default: 0.0.0.0:1207)

Format options control which files / formats are generated:
  -r, --roff                generate roff output
  -5, --html                generate entire HTML page with layout
  -f, --fragment            generate HTML fragment
      --markdown            generate post-processed markdown output

Document attributes:
      --date=<date>          published date in YYYY-MM-DD format (bottom-center)
      --manual=<name>        name of the manual (top-center)
      --organization=<name>  publishing group or individual (bottom-left)

Misc options:
  -w, --warnings            show troff warnings on stderr
  -W                        disable previously enabled troff warnings
      --version             show gonn version and exit
      --help                show this help message
`

func main() {
	var config cli_config
	var files []string
	var err error

	config = default_config()
	files, err = parse_args(os.Args[1:], &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	err = finalize_config(&config, &files)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	if config.server {
		err = gonn.RunServer(files, config.options, config.server_address)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}

	err = run_documents(files, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func default_config() cli_config {
	var config cli_config
	var date_text string
	var parsed time.Time
	var err error

	config.build = true
	config.view = false
	config.server = false
	config.server_address = "0.0.0.0:1207"
	config.write_index = false
	config.formats = nil
	config.styles = []string{"man"}
	config.groff = "groff -Wall -mtty-char -mandoc -Tascii"
	config.pager = os.Getenv("MANPAGER")
	if config.pager == "" {
		config.pager = os.Getenv("PAGER")
	}
	if config.pager == "" {
		config.pager = "more"
	}

	config.options.Manual = os.Getenv("GONN_MANUAL")
	config.options.Organization = os.Getenv("GONN_ORGANIZATION")
	date_text = os.Getenv("GONN_DATE")
	if date_text != "" {
		parsed, err = time.Parse("2006-01-02", date_text)
		if err == nil {
			config.options.Date = &parsed
		}
	}

	return config
}

func parse_args(args []string, config *cli_config) ([]string, error) {
	var files []string
	var index int
	var arg string
	var value string
	var next string
	var parsed time.Time
	var err error

	files = make([]string, 0)

	for index = 0; index < len(args); index++ {
		arg = args[index]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			files = append(files, arg)
			continue
		}

		switch {
		case arg == "--pipe":
			config.build = false
			config.server = false
		case arg == "-b" || arg == "--build":
			config.build = true
			config.server = false
		case arg == "-m" || arg == "--man":
			config.build = false
			config.view = true
			config.server = false
		case strings.HasPrefix(arg, "-S=") || strings.HasPrefix(arg, "--server="):
			value = arg[strings.Index(arg, "=")+1:]
			config.build = false
			config.view = false
			config.server = true
			config.server_address = value
		case arg == "-S" || arg == "--server":
			config.build = false
			config.view = false
			config.server = true
			if index+1 < len(args) {
				next = args[index+1]
				if is_bind_address(next) {
					config.server_address = next
					index++
				}
			}
		case arg == "-i" || arg == "--index":
			config.write_index = true
		case arg == "-r" || arg == "--roff":
			config.formats = append(config.formats, "roff")
		case arg == "-5" || arg == "--html":
			config.formats = append(config.formats, "html")
		case arg == "-f" || arg == "--fragment":
			config.formats = append(config.formats, "html_fragment")
		case arg == "--markdown":
			config.formats = append(config.formats, "markdown")
		case strings.HasPrefix(arg, "-s=") || strings.HasPrefix(arg, "--style="):
			value = arg[strings.Index(arg, "=")+1:]
			config.styles = append(config.styles, split_style_list(value)...)
		case arg == "-s" || arg == "--style":
			index++
			if index >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			config.styles = append(config.styles, split_style_list(args[index])...)
		case strings.HasPrefix(arg, "--name="):
			config.options.Name = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "--section="):
			config.options.Section = strings.TrimPrefix(arg, "--section=")
		case strings.HasPrefix(arg, "--manual="):
			config.options.Manual = strings.TrimPrefix(arg, "--manual=")
		case strings.HasPrefix(arg, "--organization="):
			config.options.Organization = strings.TrimPrefix(arg, "--organization=")
		case strings.HasPrefix(arg, "--date="):
			value = strings.TrimPrefix(arg, "--date=")
			parsed, err = time.Parse("2006-01-02", value)
			if err != nil {
				return nil, err
			}
			config.options.Date = &parsed
		case arg == "-w" || arg == "--warnings":
			config.groff = config.groff + " -ww"
		case arg == "-W":
			config.groff = config.groff + " -Ww"
		case arg == "-v" || arg == "--version":
			if gonn.Release() {
				fmt.Printf("Gonn v%s\n", gonn.Version())
			} else {
				fmt.Printf("Gonn v%s (%s)\n", gonn.Version(), gonn.Revision)
			}
			fmt.Printf("https://github.com/hyung-hwan/gonn/tree/%s\n", gonn.Revision)
			os.Exit(0)
		case arg == "--help":
			fmt.Print(usage_text)
			os.Exit(0)
		default:
			return nil, fmt.Errorf("unknown option: %s", arg)
		}
	}

	return files, nil
}

func is_bind_address(value string) bool {
	return strings.Contains(value, ":")
}

func split_style_list(value string) []string {
	var replaced string

	replaced = strings.ReplaceAll(value, ",", " ")
	return strings.Fields(replaced)
}

func finalize_config(config *cli_config, files *[]string) error {
	var input_info os.FileInfo
	var err error

	if len(*files) == 0 && is_tty(os.Stdin) {
		fmt.Print(usage_text)
		return fmt.Errorf("no input files")
	}

	if len(*files) == 0 && !config.server {
		*files = append(*files, "-")
		config.build = false
		if len(config.formats) == 0 {
			config.formats = []string{"roff"}
		}
	}

	if config.view && len(config.formats) == 0 {
		config.formats = []string{"roff"}
	}

	if config.build && len(config.formats) == 0 {
		config.formats = []string{"roff", "html"}
	}

	if contains_format(config.formats, "html_fragment") {
		config.formats = remove_format(config.formats, "html")
	}

	config.styles = gonn_styles(config.styles)
	config.options.Styles = config.styles

	if len(*files) == 1 && (*files)[0] == "-" {
		input_info, err = os.Stdin.Stat()
		if err == nil && input_info.Mode()&os.ModeCharDevice != 0 && !config.server {
			fmt.Print(usage_text)
			return fmt.Errorf("no input files")
		}
	}

	return nil
}

func is_tty(file *os.File) bool {
	var info os.FileInfo
	var err error

	info, err = file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func contains_format(formats []string, value string) bool {
	var index int

	for index = 0; index < len(formats); index++ {
		if formats[index] == value {
			return true
		}
	}

	return false
}

func remove_format(formats []string, value string) []string {
	var output []string
	var index int

	output = make([]string, 0, len(formats))
	for index = 0; index < len(formats); index++ {
		if formats[index] != value {
			output = append(output, formats[index])
		}
	}
	return output
}

func gonn_styles(styles []string) []string {
	return gonn_unique(styles)
}

func gonn_unique(values []string) []string {
	var seen map[string]struct{}
	var output []string
	var index int
	var value string
	var ok bool

	seen = make(map[string]struct{})
	output = make([]string, 0, len(values))

	for index = 0; index < len(values); index++ {
		value = values[index]
		if value == "" {
			continue
		}
		_, ok = seen[value]
		if ok {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}

	return output
}

func run_documents(files []string, config cli_config) error {
	var documents []*gonn.Document
	var index int
	var document *gonn.Document
	var err error
	var format string
	var output string
	var path string
	var writer *os.File
	var command *exec.Cmd
	var indexes map[string]*gonn.Index
	var current_index *gonn.Index
	var format_index int
	var index_path string

	documents = make([]*gonn.Document, 0, len(files))
	for index = 0; index < len(files); index++ {
		document, err = gonn.NewDocument(files[index], config.options)
		if err != nil {
			return err
		}
		documents = append(documents, document)
	}

	writer = os.Stdout
	for index = 0; index < len(documents); index++ {
		document = documents[index]
		for format_index = 0; format_index < len(config.formats); format_index++ {
			format = config.formats[format_index]
			output, err = document.Convert(format)
			if err != nil {
				return err
			}
			if config.build {
				path = document.PathFor(format)
				err = os.WriteFile(path, []byte(output+"\n"), 0644)
				if err != nil {
					return err
				}
				if format == "html" {
					fmt.Fprintf(os.Stderr, "%9s: %-43s%15s\n", format, path, "+"+strings.Join(document.Styles, ","))
				} else {
					fmt.Fprintf(os.Stderr, "%9s: %-43s\n", format, path)
				}
				if format == "roff" && config.view {
					command = exec.Command("sh", "-lc", "man "+path)
					command.Stdout = os.Stdout
					command.Stderr = os.Stderr
					command.Stdin = os.Stdin
					command.Run()
				}
			} else {
				fmt.Fprintln(writer, output)
				if config.view && format == "roff" {
					command = exec.Command("sh", "-lc", config.groff+" | "+config.pager)
					command.Stdout = os.Stdout
					command.Stderr = os.Stderr
					command.Stdin = strings.NewReader(output)
					command.Run()
				}
			}
		}
	}

	if config.write_index {
		indexes = make(map[string]*gonn.Index)
		for index = 0; index < len(documents); index++ {
			current_index = documents[index].Index
			if current_index != nil {
				indexes[current_index.Path] = current_index
			}
		}
		for index_path = range indexes {
			current_index = indexes[index_path]
			err = os.WriteFile(current_index.Path, []byte(current_index.ToText()+"\n"), 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
