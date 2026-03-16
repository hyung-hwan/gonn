package gonn

import "bufio"
import "errors"
import "os"
import "path/filepath"
import "strings"
import "sync"

type Reference struct {
	index *Index
	Name string
	Location string
}

type Index struct {
	Path string
	References []*Reference
	manuals map[string]*Document
}

var index_cache map[string]*Index = make(map[string]*Index)
var index_cache_lock sync.Mutex

func IndexForPath(path string) *Index {
	var key string
	var index *Index

	key = IndexPathForFile(path)

	index_cache_lock.Lock()
	defer index_cache_lock.Unlock()

	index = index_cache[key]
	if index != nil {
		return index
	}

	index = NewIndex(key, "")
	index_cache[key] = index
	return index
}

func IndexPathForFile(path string) string {
	var info os.FileInfo
	var err error

	info, err = os.Stat(path)
	if err == nil && info.IsDir() {
		return canonical_path(filepath.Join(path, "index.txt"))
	}

	return canonical_path(filepath.Join(filepath.Dir(path), "index.txt"))
}

func NewIndex(path string, data string) *Index {
	var index *Index
	var file_data []byte

	index = &Index{
		Path: path,
		References: make([]*Reference, 0),
		manuals: make(map[string]*Document),
	}

	if data != "" {
		index.Read(data)
		return index
	}

	if index.Exists() {
		file_data, _ = os.ReadFile(path)
		index.Read(string(file_data))
	}

	return index
}

func (index *Index) Exists() bool {
	var err error

	_, err = os.Stat(index.Path)
	return err == nil
}

func (index *Index) Read(data string) {
	var scanner *bufio.Scanner
	var line string
	var fields []string
	var name string
	var location string

	scanner = bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line = scanner.Text()
		line = strip_comment(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields = strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name = fields[0]
		location = strings.TrimSpace(line[len(name):])
		index.References = append(index.References, index.Reference(name, location))
	}
}

func strip_comment(line string) string {
	var offset int

	offset = strings.Index(line, "#")
	if offset < 0 {
		return line
	}

	return strings.TrimRight(line[:offset], " \t")
}

func (index *Index) Size() int {
	return len(index.References)
}

func (index *Index) Empty() bool {
	return len(index.References) == 0
}

func (index *Index) First() *Reference {
	if len(index.References) == 0 {
		return nil
	}
	return index.References[0]
}

func (index *Index) Last() *Reference {
	if len(index.References) == 0 {
		return nil
	}
	return index.References[len(index.References)-1]
}

func (index *Index) Get(name string) *Reference {
	var item *Reference
	var position int

	for position = 0; position < len(index.References); position++ {
		item = index.References[position]
		if item.Name == name {
			return item
		}
	}

	return nil
}

func (index *Index) Reference(name string, location string) *Reference {
	return &Reference{index: index, Name: name, Location: location}
}

func (index *Index) AddPath(path string) error {
	var absolute string
	var relative string
	var document *Document
	var reference *Reference
	var item *Reference
	var position int

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "mailto:") {
		return errors.New("local paths only")
	}

	absolute = canonical_path(path)

	for position = 0; position < len(index.References); position++ {
		item = index.References[position]
		if item.Path() == absolute {
			return nil
		}
	}

	relative = index.RelativeToIndex(absolute)

	if strings.HasSuffix(path, ".gonn") || strings.HasSuffix(path, ".ron") {
		document = index.Manual(absolute)
		reference = index.Reference(document.ReferenceName(), relative)
	} else {
		reference = index.Reference(filepath.Base(absolute), relative)
	}

	index.References = append(index.References, reference)
	return nil
}

func (index *Index) AddManual(document *Document) error {
	index.manuals[canonical_path(document.Path)] = document
	return index.AddPath(document.Path)
}

func (index *Index) Manual(path string) *Document {
	var absolute string
	var document *Document
	var err error

	absolute = canonical_path(path)
	document = index.manuals[absolute]
	if document != nil {
		return document
	}

	document, err = NewDocument(path, DocumentOptions{})
	if err != nil {
		return nil
	}

	index.manuals[absolute] = document
	return document
}

func (index *Index) Manuals() []*Document {
	var manuals []*Document
	var reference *Reference
	var position int
	var document *Document

	manuals = make([]*Document, 0)

	for position = 0; position < len(index.References); position++ {
		reference = index.References[position]
		if !reference.Relative() || !reference.Gonn() {
			continue
		}
		document = index.Manual(reference.Path())
		if document != nil {
			manuals = append(manuals, document)
		}
	}

	return manuals
}

func (index *Index) ToText() string {
	var lines []string
	var position int
	var reference *Reference

	lines = make([]string, 0, len(index.References))

	for position = 0; position < len(index.References); position++ {
		reference = index.References[position]
		lines = append(lines, reference.Name+" "+reference.LocationForText())
	}

	return strings.Join(lines, "\n")
}

func (index *Index) RelativeToIndex(path string) string {
	var absolute string
	var dir string
	var relative string
	var err error

	absolute = canonical_path(path)
	dir = filepath.Dir(canonical_path(index.Path))
	relative, err = filepath.Rel(dir, absolute)
	if err != nil {
		return absolute
	}

	return filepath.ToSlash(relative)
}

func (reference *Reference) Manual() bool {
	return strings.HasSuffix(reference.Name, ")") && strings.Contains(reference.Name, "(")
}

func (reference *Reference) Gonn() bool {
	return strings.HasSuffix(reference.Location, ".gonn") || strings.HasSuffix(reference.Location, ".ron")
}

func (reference *Reference) Remote() bool {
	return strings.HasPrefix(reference.Location, "http://") || strings.HasPrefix(reference.Location, "https://") || strings.HasPrefix(reference.Location, "mailto:")
}

func (reference *Reference) Relative() bool {
	return !reference.Remote()
}

func (reference *Reference) URL() string {
	if reference.Remote() {
		return reference.Location
	}

	return strings.TrimSuffix(strings.TrimSuffix(reference.Location, ".gonn"), ".ron") + ".html"
}

func (reference *Reference) Path() string {
	if !reference.Relative() {
		return ""
	}

	return canonical_path(filepath.Join(filepath.Dir(reference.index.Path), filepath.FromSlash(reference.Location)))
}

func (reference *Reference) LocationForText() string {
	return reference.Location
}
