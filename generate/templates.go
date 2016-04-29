package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	inPath  = "web"
	outFile = "templates.go"
)

var fns = template.FuncMap{
	"join": strings.Join,
}

var t = `
// generated by go generate; DO NOT EDIT

package main

import "html/template"

var (
	{{range .Templates}}
	{{.Name}} = template.Must(template.New("").Funcs(fns).Parse({{join .TemplateFilenamesBase "+"}}))
	{{- end}}
)

const(

	{{range .TemplateFiles}}{{.}}{{end}}
)
`

var templatesTmpl = template.Must(template.New("t").Funcs(fns).Parse(t))

func main() {
	if err := generateTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type templateFile struct {
	Name    string
	Content []byte
	Suffix  string
}

func (tf templateFile) String() string {
	baseName := strings.TrimSuffix(tf.Name, filepath.Ext(tf.Name))
	return fmt.Sprintf("%v%v = `\n%v`\n", baseName, tf.Suffix, string(tf.Content))
}

type fullTemplate struct {
	Name          string
	TemplateFiles []*templateFile
}

func (ft *fullTemplate) GetTemplateFilenames() []string {
	filenames := make([]string, 0, len(ft.TemplateFiles))
	for _, tfile := range ft.TemplateFiles {
		filenames = append(filenames, tfile.Name)
	}
	return filenames
}

func (ft *fullTemplate) TemplateFilenamesBase() []string {
	filenames := make([]string, 0, len(ft.TemplateFiles))
	for _, tfile := range ft.TemplateFiles {
		baseName := strings.TrimSuffix(tfile.Name, filepath.Ext(tfile.Name))
		baseName = fmt.Sprintf("%s%s", baseName, tfile.Suffix)
		filenames = append(filenames, baseName)
	}
	return filenames
}

type templateMap struct {
	fullTemplates map[string]*fullTemplate
	templateFiles map[string]*templateFile
}

func NewTemplateMap() *templateMap {
	return &templateMap{
		fullTemplates: make(map[string]*fullTemplate),
		templateFiles: make(map[string]*templateFile),
	}
}

func (tm *templateMap) Add(key string, tfiles []*templateFile) {
	for _, tf := range tfiles {
		tm.templateFiles[tf.Name] = tf
	}
	tm.fullTemplates[key] = &fullTemplate{Name: key, TemplateFiles: tfiles}
}

func (tm *templateMap) Get(key string) *fullTemplate {
	return tm.fullTemplates[key]
}

func (tm *templateMap) Templates() []*fullTemplate {
	templates := make([]*fullTemplate, 0, len(tm.fullTemplates))
	for k := range tm.fullTemplates {
		templates = append(templates, tm.fullTemplates[k])
	}
	return templates
}

func (tm *templateMap) WriteTemplateFile(file string, content []byte) {
	f := tm.templateFiles[file]
	if f != nil {
		f.Content = content
	}
}

func (tm *templateMap) GetTemplateFile(file string) *templateFile {
	return tm.templateFiles[file]
}

func (tm *templateMap) TemplateFiles() []*templateFile {
	files := make([]*templateFile, 0, len(tm.templateFiles))
	for k := range tm.templateFiles {
		files = append(files, tm.templateFiles[k])
	}
	return files
}

func (tm *templateMap) GetTemplateFilenames() []string {
	keys := make([]string, 0, len(tm.templateFiles))
	for k := range tm.templateFiles {
		keys = append(keys, k)
	}
	return keys
}

func makeWalkFunc(root string, tmap *templateMap) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		templateFile := strings.TrimPrefix(path, fmt.Sprintf("%s%c", root, filepath.Separator))
		if tmap.GetTemplateFile(templateFile) != nil {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			b, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}
			tmap.WriteTemplateFile(templateFile, b)
		}
		return nil
	}
}

func find(root string, tmap *templateMap) {
	filepath.Walk(root, makeWalkFunc(root, tmap))
}

func loadTemplateMap(reader io.Reader) (*templateMap, error) {
	var temp struct {
		Suffix    string
		Templates []struct {
			Name  string
			Files []string
		}
	}
	if err := json.NewDecoder(reader).Decode(&temp); err != nil {
		return nil, err
	}
	tmap := NewTemplateMap()
	for _, tmp := range temp.Templates {
		var tfiles []*templateFile
		for _, filename := range tmp.Files {
			tfiles = append(tfiles, &templateFile{Name: filename, Suffix: temp.Suffix})
		}
		tmap.Add(tmp.Name, tfiles)
	}

	find(inPath, tmap)
	return tmap, nil
}

func generateTemplates() error {
	in, err := os.Open(filepath.Join(inPath, "templates.json"))
	if err != nil {
		return err
	}
	tmap, err := loadTemplateMap(in)
	if err != nil {
		return err
	}

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	err = templatesTmpl.Execute(out, tmap)
	if err != nil {
		return err
	}

	cmd := exec.Command("gofmt", "-w", outFile)
	err = cmd.Run()
	if err != nil {
		return err
	}
	return err
}
