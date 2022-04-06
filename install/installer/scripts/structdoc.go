package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fatih/structtag"
	log "github.com/sirupsen/logrus"
)

const (
	configDir = "./pkg/config" // todo(nvn): better ways to handle the config path
)

var version string

type configDoc struct {
	configName string
	doc        string
	fields     []fieldSpec
}

type fieldSpec struct {
	name     string
	required bool
	doc      string
	note     string
}

// extractTags strips the tags of each struct field and returns json name of the
// field and if the field is a mandatory one
func extractTags(tag string) (result fieldSpec, err error) {

	// unfortunately structtag doesn't support multiple keys,
	// so we have to handle this manually
	tag = strings.Trim(tag, "`")

	tagObj, err := structtag.Parse(tag) // we assume at least JSON tag is always present
	if err != nil {
		return
	}

	metadata, err := tagObj.Get("json")
	if err != nil {
		return
	}

	result.name = metadata.Name

	reqInfo, err := tagObj.Get("validate")
	if err != nil {
		result.required = false
	} else {
		result.required = reqInfo.Name == "required"
	}

	doc, err := tagObj.Get("doc")
	if err != nil {
		return result, nil // returning nil since `doc` field can be empty
	}

	result.doc = doc.Name

	return
}

// parseConfigDir parses the AST of the config package and returns metadata
// about the `Config` struct
func parseConfigDir(fileDir string) (configSpec configDoc, err error) {

	fset := token.NewFileSet()

	// we basically parse the AST of the config package
	pkgs, err := parser.ParseDir(fset, fileDir, nil, parser.ParseComments)
	if err != nil {
		return
	}

	configPkg := pkgs["config"]

	pkgData := doc.New(configPkg, "./", 0)

	var specs []fieldSpec

	for _, t := range pkgData.Types {
		// we only care about the `Config` struct
		if t.Name != "Config" {
			continue
		}

		configSpec.configName = t.Name
		configSpec.doc = t.Doc

		for _, spec := range t.Decl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			switch typeSpec.Type.(type) {
			case *ast.StructType:
				structType := typeSpec.Type.(*ast.StructType)

				var fieldInfo fieldSpec

				for _, field := range structType.Fields.List {
					fieldInfo, err = extractTags(field.Tag.Value)
					if err != nil {
						return
					}

					// more notes about the doc can be provided as a comment
					// above the field
					if field.Doc != nil {
						var comment string = ""

						// sometimes the comments are multi-line
						for _, line := range field.Doc.List {
							comment = fmt.Sprintf("%s %s", comment, strings.Trim(line.Text, "//"))
						}

						fieldInfo.note = comment
					}

					specs = append(specs, fieldInfo)
				}
			}
		}

		// we hardcode the value for apiVersion since it is not present in
		// Config struct
		specs = append(specs,
			fieldSpec{
				name:     "apiVersion",
				required: true,
				doc: fmt.Sprintf("API version of the Gitpod config defintion."+
					" `%s` in this version of Config", version)})

		configSpec.fields = specs

		break
	}

	return
}

func generateMarkdown(configSpec configDoc) string {

	mddoc := strings.Builder{}
	mddoc.WriteString(fmt.Sprintf("# %s %s\n%s\n", configSpec.configName, version, configSpec.doc))
	mddoc.WriteString("## Supported parameters\n")
	mddoc.WriteString("| Property | Required | Description | Notes |\n")
	mddoc.WriteString("| --- | --- | --- | --- |\n")

	for _, field := range configSpec.fields {
		var required string = "N"

		if field.required {
			required = "Y"
		}

		mddoc.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", field.name, required, field.doc, field.note))
	}

	return mddoc.String()
}

func main() {
	versionFlag := flag.String("version", "v1", "Config version for doc creation")
	flag.Parse()

	version = *versionFlag

	log.Infof("Generating doc for config version %s", version)

	fileDir := fmt.Sprintf("%s/%s", configDir, version)

	// get the `Config` struct field info from `config` pkg
	configSpec, err := parseConfigDir(fileDir)
	if err != nil {
		log.Fatal(err)
	}

	// generate markdown for the doc
	mddoc := generateMarkdown(configSpec)

	// write the md file of name config.md in the same directory as config
	mdfilename := filepath.Join(fileDir, "config.md")

	err = ioutil.WriteFile(mdfilename, []byte(mddoc), 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("The doc is written to the file %s", mdfilename)
}
