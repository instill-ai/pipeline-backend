package gen

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	_ "embed"

	"github.com/go-playground/validator/v10"
	"github.com/russross/blackfriday/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/resources/schemas"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	definitionsFile = "definition.yaml"
	setupFile       = "setup.yaml"
	tasksFile       = "tasks.yaml"
)

//go:embed resources/templates/readme.mdx.tmpl
var readmeTmpl string

// READMEGenerator is used to generate the README file of a component.
type READMEGenerator struct {
	validate *validator.Validate

	configDir         string
	outputFile        string
	extraContentPaths map[string]string
}

// NewREADMEGenerator returns an initialized generator.
func NewREADMEGenerator(configDir, outputFile string, extraContentPaths map[string]string) *READMEGenerator {
	return &READMEGenerator{
		validate: validator.New(validator.WithRequiredStructEnabled()),

		configDir:         configDir,
		outputFile:        outputFile,
		extraContentPaths: extraContentPaths,
	}
}

func (g *READMEGenerator) parseDefinition(configDir string) (d definition, err error) {
	definitionYAML, err := os.ReadFile(filepath.Join(configDir, definitionsFile))
	if err != nil {
		return d, err
	}
	definitionJSON, err := convertYAMLToJSON(definitionYAML)
	if err != nil {
		return d, err
	}

	renderedDefinitionJSON, err := componentbase.RenderJSON(definitionJSON, nil)
	if err != nil {
		return d, err
	}

	def := definition{}
	if err := json.Unmarshal(renderedDefinitionJSON, &def); err != nil {
		return d, err
	}

	if err := g.validate.Var(def, "len=1,dive"); err != nil {
		return d, fmt.Errorf("invalid definitions file:\n%w", asValidationError(err))
	}

	return def, nil
}

func (g *READMEGenerator) parseSetup(configDir string) (s *objectSchema, err error) {
	setupYAML, err := os.ReadFile(filepath.Join(configDir, setupFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	setupJSON, err := convertYAMLToJSON(setupYAML)
	if err != nil {
		return nil, err
	}

	renderedSetupJSON, err := componentbase.RenderJSON(setupJSON, nil)
	if err != nil {
		return nil, err
	}

	setup := &objectSchema{}
	if err := json.Unmarshal(renderedSetupJSON, &setup); err != nil {
		return nil, err
	}

	if err := g.validate.Var(setup, "len=1,dive"); err != nil {
		return nil, fmt.Errorf("invalid definitions file:\n%w", asValidationError(err))
	}

	return setup, nil
}

func (g *READMEGenerator) parseTasks(configDir string) (map[string]task, error) {
	tasksYAML, err := os.ReadFile(filepath.Join(configDir, tasksFile))
	if err != nil {
		return nil, err
	}
	tasksJSON, err := convertYAMLToJSON(tasksYAML)
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}
	additionalJSONs := map[string][]byte{}
	for _, file := range files {
		additionalYAML, err := os.ReadFile(filepath.Join(configDir, file.Name()))
		if err != nil {
			return nil, err
		}
		additionalJSON, err := convertYAMLToJSON(additionalYAML)
		if err != nil {
			return nil, err
		}
		additionalJSONs[file.Name()] = additionalJSON

	}

	schemaJSON, err := convertYAMLToJSON(schemas.SchemaYAML)
	if err != nil {
		return nil, err
	}
	additionalJSONBytes := map[string][]byte{
		"schema.yaml": schemaJSON,
	}
	renderedTasksJSON, err := componentbase.RenderJSON(tasksJSON, additionalJSONBytes)
	if err != nil {
		return nil, err
	}
	tasks := map[string]task{}
	if err := json.Unmarshal(renderedTasksJSON, &tasks); err != nil {
		return nil, err
	}

	if err := g.validate.Var(tasks, "dive"); err != nil {
		return nil, fmt.Errorf("invalid tasks file:\n%w", asValidationError(err))
	}

	return tasks, nil
}

// This is used to build the cURL examples for Instill Core and Cloud.
type host struct {
	Name string
	URL  string
}

// Generate creates a MDX file with the component documentation from the
// component schema.
func (g *READMEGenerator) Generate() error {
	def, err := g.parseDefinition(g.configDir)
	if err != nil {
		return err
	}

	setup, err := g.parseSetup(g.configDir)
	if err != nil {
		return err
	}

	tasks, err := g.parseTasks(g.configDir)
	if err != nil {
		return err
	}

	readme, err := template.New("readme").Funcs(template.FuncMap{
		"firstToLower":             firstToLower,
		"asAnchor":                 blackfriday.SanitizedAnchorName,
		"loadExtraContent":         g.loadExtraContent,
		"enumValues":               enumValues,
		"anchorSetup":              anchorSetup,
		"anchorTaskObject":         anchorTaskObject,
		"insertHeaderByObjectKey":  insertHeaderByObjectKey,
		"insertHeaderByConstValue": insertHeaderByConstValue,
		"hosts": func() []host {
			return []host{
				{Name: "Instill-Cloud", URL: "https://api.instill.tech"},
				{Name: "Instill-Core", URL: "http://localhost:8080"},
			}
		},
	}).Parse(readmeTmpl)
	if err != nil {
		return err
	}

	out, err := os.Create(g.outputFile)
	if err != nil {
		return err
	}

	defer out.Close()

	p, err := readmeParams{}.parseDefinition(def, setup, tasks)
	if err != nil {
		return fmt.Errorf("converting to template params: %w", err)
	}

	return readme.Execute(out, p)
}

func (g *READMEGenerator) loadExtraContent(section string) (string, error) {
	if g.extraContentPaths[section] == "" {
		return "", nil
	}

	extra, err := os.ReadFile(g.extraContentPaths[section])
	if err != nil {
		return "", fmt.Errorf("reading extra contents for section %s: %w", section, err)
	}

	return string(extra), nil
}

type readmeTask struct {
	ID            string
	Title         string
	Description   string
	Input         []resourceProperty
	InputObjects  []map[string]objectSchema
	OneOfs        []map[string][]objectSchema
	Output        []resourceProperty
	OutputObjects []map[string]objectSchema
}

type resourceProperty struct {
	property
	ID       string
	Required bool
}

type setupConfig struct {
	Prerequisites string
	Properties    []resourceProperty
	OneOf         map[string][]objectSchema
}

type readmeParams struct {
	ID            string
	Title         string
	Description   string
	Vendor        string
	IsDraft       bool
	ComponentType ComponentType
	ReleaseStage  releaseStage
	SourceURL     string
	SetupConfig   setupConfig

	Tasks []readmeTask
}

// parseDefinition converts a component definition and its tasks to the README
// template params.
func (p readmeParams) parseDefinition(d definition, s *objectSchema, tasks map[string]task) (readmeParams, error) {
	p.ComponentType = toComponentType[d.Type]

	var err error
	if p.Tasks, err = parseREADMETasks(d.AvailableTasks, tasks); err != nil {
		return p, err
	}

	p.ID = d.ID
	p.Title = d.Title
	p.Vendor = d.Vendor
	p.Description = d.Description
	p.IsDraft = !d.Public
	p.ReleaseStage = d.ReleaseStage
	p.SourceURL = d.SourceURL

	p.SetupConfig = setupConfig{Prerequisites: d.Prerequisites}

	if s != nil {
		p.SetupConfig.Properties = parseResourceProperties(s)
		p.SetupConfig.parseOneOfProperties(s.Properties)
	}

	return p, nil
}

func parseREADMETasks(availableTasks []string, tasks map[string]task) ([]readmeTask, error) {
	readmeTasks := make([]readmeTask, len(availableTasks))
	for i, at := range availableTasks {
		t, ok := tasks[at]
		if !ok {
			return nil, fmt.Errorf("invalid tasks file:\nmissing %s", at)
		}

		rt := readmeTask{
			ID:          at,
			Description: t.Description,
			Input:       parseResourceProperties(t.Input),
			Output:      parseResourceProperties(t.Output),
		}

		rt.parseObjectProperties(t.Input.Properties, true)
		rt.parseObjectProperties(t.Output.Properties, false)
		rt.parseOneOfsProperties(t.Input.Properties)

		if rt.Title = t.Title; rt.Title == "" {
			rt.Title = titleCase(componentbase.TaskIDToTitle(at))
		}

		readmeTasks[i] = rt
	}

	return readmeTasks, nil
}

func parseResourceProperties(o *objectSchema) []resourceProperty {
	if o == nil {
		return []resourceProperty{}
	}

	o.Title = titleCase(o.Title)

	// We need a map first to set the Required property, then we'll
	// transform it to a slice.
	propMap := make(map[string]resourceProperty)
	for k, op := range o.Properties {
		if op.Deprecated {
			continue
		}

		prop := resourceProperty{
			ID:       k,
			property: op,
		}

		prop.Title = titleCase(prop.Title)
		prop.replaceFormat()

		// If type is array, extend the type with the element type.
		switch prop.Type {
		case "array":
			if prop.Items.Type != "" {
				if prop.Items.Type == "*" {
					prop.Type = "array[any]"
				} else {
					prop.Type += fmt.Sprintf("[%s]", prop.Items.Type)
				}
			}
		case "":
			prop.Type = "any"
		}
		prop.replaceDescription()

		propMap[k] = prop
	}

	for _, k := range o.Required {
		if prop, ok := propMap[k]; ok {
			prop.Required = true
			propMap[k] = prop
		}
	}

	props := make([]resourceProperty, len(propMap))
	idx := 0
	for k := range propMap {
		props[idx] = propMap[k]
		idx++
	}

	// Note: The order might not be consecutive numbers.
	slices.SortFunc(props, func(i, j resourceProperty) int {
		if cmp := cmp.Compare(*i.Order, *j.Order); cmp != 0 {
			return cmp
		}
		return cmp.Compare(i.ID, j.ID)
	})

	return props
}

func (rt *readmeTask) parseObjectProperties(properties map[string]property, isInput bool) {
	if properties == nil {
		return
	}

	sortedProperties := sortPropertiesByOrder(properties)

	for _, op := range sortedProperties {
		if op.Deprecated {
			continue
		}

		if op.Type != "object" && op.Type != "array[object]" && (op.Type != "array" || op.Items.Type != "object") {
			continue
		}

		if op.Type == "object" && op.Properties == nil {
			continue
		}

		if op.Type == "array[object]" && op.Items.Properties == nil {
			continue
		}

		if isSemiStructuredObject(op) {
			continue
		}

		if arrayToBeSkipped(op) {
			continue
		}

		op.replaceDescription()
		op.replaceFormat()

		if op.Type == "object" {

			if isInput {
				rt.InputObjects = append(rt.InputObjects, map[string]objectSchema{
					op.Title: {
						Properties:  op.Properties,
						Description: op.Description,
					},
				})
				rt.parseObjectProperties(op.Properties, isInput)
			} else {
				rt.OutputObjects = append(rt.OutputObjects, map[string]objectSchema{
					op.Title: {
						Properties:  op.Properties,
						Description: op.Description,
					},
				})
				rt.parseObjectProperties(op.Properties, isInput)
			}
		} else { // op.Type == "array[object]" || (op.Type == "array" || op.Items.Type == "object")

			props := op.Items.Properties
			for key := range props {
				prop := props[key]
				prop.replaceDescription()
				prop.replaceFormat()
				props[key] = prop
			}

			if isInput {
				rt.InputObjects = append(rt.InputObjects, map[string]objectSchema{
					op.Title: {
						Properties:  op.Items.Properties,
						Description: op.Description,
					},
				})

				rt.parseObjectProperties(op.Items.Properties, isInput)
			} else {
				rt.OutputObjects = append(rt.OutputObjects, map[string]objectSchema{
					op.Title: {
						Properties:  op.Items.Properties,
						Description: op.Description,
					},
				})
				rt.parseObjectProperties(op.Items.Properties, isInput)
			}
		}
	}

	return
}

func sortPropertiesByOrder(properties map[string]property) []property {
	// Extract the keys
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}

	// Sort the keys based on the Order field in the property
	sort.Slice(keys, func(i, j int) bool {
		// Default to 0 if Order is nil
		orderI := 0
		if properties[keys[i]].Order != nil {
			orderI = *properties[keys[i]].Order
		}

		orderJ := 0
		if properties[keys[j]].Order != nil {
			orderJ = *properties[keys[j]].Order
		}

		return orderI < orderJ
	})

	sortedProperties := make([]property, 0, len(properties))
	for _, key := range keys {
		sortedProperties = append(sortedProperties, properties[key])
	}

	return sortedProperties
}

func (rt *readmeTask) parseOneOfsProperties(properties map[string]property) {
	if properties == nil {
		return
	}

	for key, op := range properties {
		if op.Deprecated {
			continue
		}

		if op.Type != "object" && op.Type != "array" {
			continue
		}

		if op.Type == "array" {
			if op.Items.Type != "object" {
				continue
			}

			if op.Items.OneOf != nil {
				rt.OneOfs = append(rt.OneOfs, map[string][]objectSchema{
					key: op.Items.OneOf,
				})
			}

		}

		if op.OneOf != nil {
			rt.OneOfs = append(rt.OneOfs, map[string][]objectSchema{
				key: op.OneOf,
			})
		}
		rt.parseOneOfsProperties(op.Properties)
	}

	return
}

func (sc *setupConfig) parseOneOfProperties(properties map[string]property) {
	if properties == nil {
		return
	}

	for key, op := range properties {
		if op.Deprecated {
			continue
		}

		// Now, we only have 1 layer. So, we do not have to recursively parse.
		if op.OneOf != nil {
			sc.OneOf = map[string][]objectSchema{
				key: op.OneOf,
			}
		}
	}

	return
}

func firstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}

	mod := unicode.ToLower(r)
	if r == mod {
		return s
	}

	return string(mod) + s[size:]
}

func enumValues(enum []string) string {
	length := len(enum)
	var result string
	result = "<br/><details><summary><strong>Enum values</strong></summary><ul>"

	for i, e := range enum {
		result += fmt.Sprintf("<li>`%s`</li>", e)
		if i == length-1 {
			result += "</ul>"
		}
	}
	result += "</details>"

	return result
}

func anchorSetup(p interface{}) string {
	switch prop := p.(type) {
	case resourceProperty:
		return anchorSetupFromProperty(prop.property)
	case property:
		return anchorSetupFromProperty(prop)
	default:
		return ""
	}
}

func anchorSetupFromProperty(prop property) string {
	if isSemiStructuredObject(prop) {
		return prop.Title
	}
	if prop.Type == "object" ||
		(prop.Type == "array" && prop.Items.Type == "object") ||
		(prop.Type == "array[object]") {
		return fmt.Sprintf("[%s](#%s)", prop.Title, blackfriday.SanitizedAnchorName(prop.Title))
	}
	return prop.Title
}

func isSemiStructuredObject(p property) bool {
	return p.Type == "object" && p.Properties == nil && p.OneOf == nil
}

func arrayToBeSkipped(op property) bool {
	return op.Type == "array" && op.Items.Type == "object" && op.Items.Properties == nil
}

func anchorTaskObject(p interface{}, task readmeTask) string {
	switch prop := p.(type) {
	case resourceProperty:
		return anchorTaskWithProperty(prop.property, task.Title)
	case property:
		return anchorTaskWithProperty(prop, task.Title)
	default:
		return ""
	}

}

func anchorTaskWithProperty(prop property, taskName string) string {
	if isSemiStructuredObject(prop) {
		return prop.Title
	}
	if prop.Type == "object" ||
		(prop.Type == "array" && prop.Items.Type == "object") ||
		(prop.Type == "array[object]") {
		return fmt.Sprintf("[%s](#%s-%s)", prop.Title, blackfriday.SanitizedAnchorName(taskName), blackfriday.SanitizedAnchorName(prop.Title))
	}
	return prop.Title
}

func insertHeaderByObjectKey(key string, taskOrString interface{}) string {
	var prefix string
	switch v := taskOrString.(type) {
	case readmeTask:
		prefix = blackfriday.SanitizedAnchorName(v.Title)
	case string:
		prefix = blackfriday.SanitizedAnchorName(v)
	default:
		// Handle unexpected types, maybe return an error or use a default value
		prefix = "unknown"
	}
	return fmt.Sprintf(
		`<h4 id="%s-%s">%s</h4>`,
		prefix,
		blackfriday.SanitizedAnchorName(key),
		// Replace dashes and underscores with spaces, then title-case.
		titleCase(strings.ReplaceAll(strings.ReplaceAll(key, "-", " "), "_", " ")),
	)
}

func insertHeaderByConstValue(option objectSchema, taskOrString interface{}) string {
	var prefix string
	switch v := taskOrString.(type) {
	case readmeTask:
		prefix = blackfriday.SanitizedAnchorName(v.Title)
	case string:
		prefix = blackfriday.SanitizedAnchorName(v)
	default:
		// Handle unexpected types, maybe return an error or use a default value
		prefix = "unknown"
	}

	for _, prop := range option.Properties {
		if prop.Const != "" {
			return fmt.Sprintf(`<h5 id="%s-%s"><code>%s</code></h5>`, prefix, blackfriday.SanitizedAnchorName(option.Title), option.Title)
		}
	}
	return ""
}

func (prop *property) replaceDescription() {
	if strings.Contains(prop.Description, "{{") && strings.Contains(prop.Description, "}}") {
		prop.Description = strings.ReplaceAll(prop.Description, "{{", "`{{")
		prop.Description = strings.ReplaceAll(prop.Description, "}}", "}}`")
	} else {
		prop.Description = strings.ReplaceAll(prop.Description, "\n", " ")
		prop.Description = strings.ReplaceAll(prop.Description, "{", "\\{")
		prop.Description = strings.ReplaceAll(prop.Description, "}", "\\}")
	}
}

func (prop *property) replaceFormat() {
	if prop.Type == "*" {
		prop.Type = "any"
	}
	if prop.Type == "array" && prop.Items.Type == "*" {
		prop.Type = "array[any]"
	}
}
