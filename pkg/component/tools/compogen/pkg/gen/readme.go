package gen

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	// embed is used to include the README template at build time.
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

// originalTasksByID holds the unrendered tasks (before $ref resolution), populated by parseTasks
var originalTasksByID map[string]task

// global signature to original $defs key mapping
var sigToDefsKey map[string]string

// global enum values by $defs key
var defEnumsByKey map[string][]string

// original $defs by key for ref resolution
var defsByKey map[string]objectSchema

// NewREADMEGenerator returns an initialized generator.
func NewREADMEGenerator(configDir, outputFile string, extraContentPaths map[string]string) *READMEGenerator {
	return &READMEGenerator{
		validate: validator.New(validator.WithRequiredStructEnabled()),

		configDir:         configDir,
		outputFile:        outputFile,
		extraContentPaths: extraContentPaths,
	}
}

type host struct {
	Name string
	URL  string
}

// buildMergedSchema returns a new object schema with Properties/Required from
// the base schema plus any contributions from AllOf, without mutating the
// original object.
func buildMergedSchema(o *objectSchema) *objectSchema {
	if o == nil {
		return nil
	}
	merged := &objectSchema{
		Title:       o.Title,
		Description: o.Description,
		Required:    append([]string{}, o.Required...),
		Properties:  map[string]property{},
	}
	// Sort keys for deterministic processing
	propKeys := make([]string, 0, len(o.Properties))
	for k := range o.Properties {
		propKeys = append(propKeys, k)
	}
	sort.Strings(propKeys)

	for _, k := range propKeys {
		merged.Properties[k] = o.Properties[k]
	}
	for _, s := range o.AllOf {
		// Compute base offset so this group's orders begin after existing max
		base := maxOrderValue(merged.Properties)
		offset := 0
		if base >= 0 {
			offset = base + 1
		}
		// Sort keys for deterministic processing
		sPropKeys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			sPropKeys = append(sPropKeys, k)
		}
		sort.Strings(sPropKeys)

		for _, k := range sPropKeys {
			v := s.Properties[k]
			// Offset this group's uiOrder so groups don't collide
			if v.Order != nil {
				newOrder := *v.Order + offset
				v.Order = &newOrder
			}
			if existing, exists := merged.Properties[k]; exists {
				merged.Properties[k] = mergeProperty(existing, v)
			} else {
				merged.Properties[k] = v
			}
		}
		if len(s.Required) > 0 {
			merged.Required = append(merged.Required, s.Required...)
		}
	}
	return merged
}

// maxOrderValue returns the maximum uiOrder among the given properties, or -1 if none.
func maxOrderValue(props map[string]property) int {
	max := -1
	for _, p := range props {
		if p.Order != nil && *p.Order > max {
			max = *p.Order
		}
		// Also consider nested arrays of objects or objects with uiOrder at this level only
	}
	return max
}

// mergeProperty merges src into dst, preferring non-empty/non-nil fields from src
func mergeProperty(dst, src property) property {
	// Prefer uiOrder when missing
	if dst.Order == nil && src.Order != nil {
		dst.Order = src.Order
	}
	// Prefer title/description/type when empty
	if dst.Title == "" && src.Title != "" {
		dst.Title = src.Title
	}
	if dst.Description == "" && src.Description != "" {
		dst.Description = src.Description
	}
	if dst.Type == "" && src.Type != "" {
		dst.Type = src.Type
	}
	// Preserve $ref if destination missing
	if dst.Ref == "" && src.Ref != "" {
		dst.Ref = src.Ref
	}
	// Merge items
	if dst.Type == "array" {
		if dst.Items.Type == "" && src.Items.Type != "" {
			dst.Items.Type = src.Items.Type
		}
		if dst.Items.Properties == nil && src.Items.Properties != nil {
			dst.Items.Properties = src.Items.Properties
		}
		if dst.Items.OneOf == nil && src.Items.OneOf != nil && dst.Items.Ref == "" {
			dst.Items.OneOf = src.Items.OneOf
		}
		if dst.Items.Ref == "" && src.Items.Ref != "" {
			dst.Items.Ref = src.Items.Ref
		}
	}
	// Merge object properties
	if dst.Properties == nil && src.Properties != nil {
		dst.Properties = src.Properties
	}
	// Merge oneOf if missing - but NOT for $ref properties
	// $ref properties should not inherit oneOf from their targets
	if dst.OneOf == nil && src.OneOf != nil && dst.Ref == "" {
		dst.OneOf = src.OneOf
	}
	// Merge enum if missing
	if len(dst.Enum) == 0 && len(src.Enum) > 0 {
		dst.Enum = src.Enum
	}
	// Deprecated flag: if any marks deprecated, keep it
	if src.Deprecated {
		dst.Deprecated = true
	}
	return dst
}

// Generate creates a MDX file with the component documentation from the component schema.
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
		"firstToLower":               firstToLower,
		"asAnchor":                   blackfriday.SanitizedAnchorName,
		"loadExtraContent":           g.loadExtraContent,
		"enumValues":                 enumValues,
		"anchorSetup":                anchorSetup,
		"anchorTaskObject":           anchorTaskObject,
		"insertHeaderByObjectKey":    insertHeaderByObjectKey,
		"insertHeaderByConstValue":   insertHeaderByConstValue,
		"renderObjectDetails":        renderObjectDetails,
		"renderObjectDetailsWithKey": renderObjectDetailsWithKey,
		"insertHeaderWithParents":    insertHeaderWithParents,
		"parents":                    makeParents,
		"getParents":                 getParents,
		"typeWithRef":                typeWithRefAny,
		"typeWithEnum":               typeWithEnumAny,
		"sortedKeys":                 getSortedKeysByAppearanceWithTask,
		"hosts": func() []host {
			return []host{
				{Name: "Instill-Cloud", URL: "https://api.instill-ai.com"},
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
	// Preserve original (unrendered) tasks to keep $ref information
	{
		orig := map[string]task{}
		if err := json.Unmarshal(tasksJSON, &orig); err == nil {
			originalTasksByID = orig
		} else {
			originalTasksByID = nil
		}
		// Build signature index from original $defs if present
		type defsWrapper struct {
			Defs map[string]json.RawMessage `json:"$defs"`
		}
		var dw defsWrapper
		if err := json.Unmarshal(tasksJSON, &dw); err == nil && len(dw.Defs) > 0 {
			sigToDefsKey = map[string]string{}
			defsByKey = map[string]objectSchema{}

			// Sort keys for deterministic processing
			defKeys := make([]string, 0, len(dw.Defs))
			for k := range dw.Defs {
				defKeys = append(defKeys, k)
			}
			sort.Strings(defKeys)

			for _, k := range defKeys {
				rawDef := dw.Defs[k]
				// Try to parse as property first (for enums)
				var prop property
				if err := json.Unmarshal(rawDef, &prop); err == nil && prop.Type != "" {
					// Convert property to objectSchema for storage
					os := objectSchema{
						Title:       prop.Title,
						Description: prop.Description,
						Properties:  prop.Properties,
					}
					// Store enum values in a custom way - we'll access them from the original property
					defsByKey[k] = os
					// Store the original property for enum access
					if len(prop.Enum) > 0 {
						// We need a way to access enum values - let's store them in a global map
						if defEnumsByKey == nil {
							defEnumsByKey = make(map[string][]string)
						}
						defEnumsByKey[k] = prop.Enum
					}
				} else {
					// Try to parse as objectSchema
					var os objectSchema
					if err := json.Unmarshal(rawDef, &os); err == nil {
						defsByKey[k] = os
					}
				}

				// Build signature for object schemas only
				if os, ok := defsByKey[k]; ok && len(os.Properties) > 0 {
					sig := buildObjectSignature(os.Properties)
					// prefer first seen key
					if _, exists := sigToDefsKey[sig]; !exists {
						sigToDefsKey[sig] = k
					}
				}
			}
		}
	}
	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}
	additionalJSONs := map[string][]byte{}
	for _, file := range files {
		name := file.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		additionalYAML, err := os.ReadFile(filepath.Join(configDir, name))
		if err != nil {
			return nil, err
		}
		additionalJSON, err := convertYAMLToJSON(additionalYAML)
		if err != nil {
			return nil, err
		}
		additionalJSONs[name] = additionalJSON

	}

	schemaJSON, err := convertYAMLToJSON(schemas.SchemaYAML)
	if err != nil {
		return nil, err
	}
	chatSchemaJSON, err := convertYAMLToJSON(schemas.ChatSchemaYAML)
	if err != nil {
		return nil, err
	}
	additionalJSONBytes := map[string][]byte{
		"schema.yaml":      schemaJSON,
		"chat-schema.yaml": chatSchemaJSON,
	}
	// Merge local configDir YAMLs (so refs like 'something.yaml#/' work)
	for name, b := range additionalJSONs {
		additionalJSONBytes[name] = b
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

// mergeAllOf flattens properties/required from allOf entries into the target schema.
func mergeAllOf(target *objectSchema) {
	if target == nil || len(target.AllOf) == 0 {
		return
	}
	if target.Properties == nil {
		target.Properties = map[string]property{}
	}
	for _, s := range target.AllOf {
		// Merge properties - sort keys for deterministic processing
		propKeys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			propKeys = append(propKeys, k)
		}
		sort.Strings(propKeys)

		for _, k := range propKeys {
			if _, exists := target.Properties[k]; !exists {
				target.Properties[k] = s.Properties[k]
			}
		}
		// Merge required
		if len(s.Required) > 0 {
			target.Required = append(target.Required, s.Required...)
		}
	}
	// Clear AllOf after merge to avoid double-processing
	target.AllOf = nil
}

func parseREADMETasks(availableTasks []string, tasks map[string]task) ([]readmeTask, error) {
	readmeTasks := make([]readmeTask, len(availableTasks))
	// helper: copy $ref from original into merged schema when missing
	injectRefsFromOriginal := func(merged, original *objectSchema) {
		if merged == nil || original == nil || merged.Properties == nil {
			return
		}
		// helper to find property by key across original.Properties and original.AllOf[*].Properties
		findProp := func(orig *objectSchema, key string) (property, bool) {
			if orig == nil {
				return property{}, false
			}
			if orig.Properties != nil {
				if p, ok := orig.Properties[key]; ok {
					return p, true
				}
			}
			for _, s := range orig.AllOf {
				if s.Properties != nil {
					if p, ok := s.Properties[key]; ok {
						return p, true
					}
				}
			}
			return property{}, false
		}
		// recursive injection for a property pair
		var injectProp func(mp *property, op *property)
		injectProp = func(mp *property, op *property) {
			if mp == nil || op == nil {
				return
			}
			// object $ref
			if mp.Ref == "" && op.Ref != "" {
				mp.Ref = op.Ref
			}
			if strings.HasPrefix(mp.Ref, "#/$defs/") {
				rk := extractRefKey(mp.Ref)
				if def, ok := defsByKey[rk]; ok {
					if mp.Title == "" && def.Title != "" {
						mp.Title = def.Title
					}
					if mp.Title == "" {
						mp.Title = titleCase(strings.ReplaceAll(rk, "-", " "))
					}
					if mp.Description == "" && def.Description != "" {
						mp.Description = def.Description
					}
					// For enum $refs, always prefer the enum's description even if property has one
					if len(def.Properties) == 0 && def.Description != "" {
						mp.Description = def.Description
					}
					// Copy enum values from the referenced definition
					if mp.Enum == nil && defEnumsByKey != nil {
						if enumValues, ok := defEnumsByKey[rk]; ok && len(enumValues) > 0 {
							mp.Enum = enumValues
						}
					}
					if mp.Properties == nil && len(def.Properties) > 0 {
						mp.Properties = def.Properties
					}
					if mp.Type == "" {
						if len(def.Properties) > 0 {
							mp.Type = "object"
						} else {
							mp.Type = "string" // most enums are strings
						}
					}
					// For enums, we need to create a property-like structure to hold enum values
					// This will be handled in the defsByKey parsing, not here
				}
			}
			// array item $ref
			if mp.Items.Ref == "" && op.Items.Ref != "" {
				mp.Items.Ref = op.Items.Ref
			}
			if strings.HasPrefix(mp.Items.Ref, "#/$defs/") {
				rk := extractRefKey(mp.Items.Ref)
				if def, ok := defsByKey[rk]; ok {
					if mp.Items.Title == "" && def.Title != "" {
						mp.Items.Title = def.Title
					}
					if mp.Items.Title == "" {
						mp.Items.Title = titleCase(strings.ReplaceAll(rk, "-", " "))
					}
					// Only set properties if the def has properties (don't override for enums)
					if len(def.Properties) > 0 && mp.Items.Properties == nil {
						mp.Items.Properties = def.Properties
					}
					// Copy enum values for array items
					if mp.Items.Enum == nil && defEnumsByKey != nil {
						if enumValues, ok := defEnumsByKey[rk]; ok && len(enumValues) > 0 {
							mp.Items.Enum = enumValues
						}
					}
					if mp.Items.Type == "" {
						if len(def.Properties) > 0 {
							mp.Items.Type = "object"
						} else {
							mp.Items.Type = "string" // most enums are strings
						}
					}
				}
			}
			// descend into object properties
			if mp.Properties != nil {
				// Sort keys for deterministic processing
				childKeys := make([]string, 0, len(mp.Properties))
				for ck := range mp.Properties {
					childKeys = append(childKeys, ck)
				}
				sort.Strings(childKeys)

				for _, ck := range childKeys {
					childMerged := mp.Properties[ck]
					var childOrig property
					if op.Properties != nil {
						if p, ok := op.Properties[ck]; ok {
							childOrig = p
							injectProp(&childMerged, &childOrig)
							mp.Properties[ck] = childMerged
							continue
						}
					}
					// search in op.OneOf options if present
					if len(op.OneOf) > 0 {
						for _, opt := range op.OneOf {
							if p, ok := opt.Properties[ck]; ok {
								childOrig = p
								injectProp(&childMerged, &childOrig)
								mp.Properties[ck] = childMerged
								break
							}
						}
					}
				}
			}
			// descend into array item object properties
			if mp.Items.Type == "object" && mp.Items.Properties != nil {
				// Sort keys for deterministic processing
				itemKeys := make([]string, 0, len(mp.Items.Properties))
				for ck := range mp.Items.Properties {
					itemKeys = append(itemKeys, ck)
				}
				sort.Strings(itemKeys)

				for _, ck := range itemKeys {
					childMerged := mp.Items.Properties[ck]
					var childOrig property
					if op.Items.Properties != nil {
						if p, ok := op.Items.Properties[ck]; ok {
							childOrig = p
							injectProp(&childMerged, &childOrig)
							mp.Items.Properties[ck] = childMerged
							continue
						}
					}
					if len(op.Items.OneOf) > 0 {
						for _, opt := range op.Items.OneOf {
							if p, ok := opt.Properties[ck]; ok {
								childOrig = p
								injectProp(&childMerged, &childOrig)
								mp.Items.Properties[ck] = childMerged
								break
							}
						}
					}
				}
			}
		}
		// Sort keys for deterministic processing
		mergedKeys := make([]string, 0, len(merged.Properties))
		for k := range merged.Properties {
			mergedKeys = append(mergedKeys, k)
		}
		sort.Strings(mergedKeys)

		for _, k := range mergedKeys {
			mp := merged.Properties[k]
			if op, ok := findProp(original, k); ok {
				// top-level refs
				if mp.Ref == "" && op.Ref != "" {
					mp.Ref = op.Ref
				}
				if mp.Items.Ref == "" && op.Items.Ref != "" {
					mp.Items.Ref = op.Items.Ref
				}
				// recurse into children
				injectProp(&mp, &op)
				merged.Properties[k] = mp
			}
		}
	}
	for i, at := range availableTasks {
		t, ok := tasks[at]
		if !ok {
			return nil, fmt.Errorf("invalid tasks file:\nmissing %s", at)
		}

		// Build merged schemas (without mutating the originals) so the tables
		// include contributions from allOf entries.
		inSchema := buildMergedSchema(t.Input)
		outSchema := buildMergedSchema(t.Output)

		rt := readmeTask{
			ID:                    at,
			Description:           t.Description,
			TitleToOneOf:          map[string][]objectSchema{},
			SignatureToCanonical:  map[string]string{},
			FieldKeyToCanonical:   map[string]string{},
			CanonicalToParents:    map[string]map[string]bool{},
			CanonicalToParentsIn:  map[string]map[string]bool{},
			CanonicalToParentsOut: map[string]map[string]bool{},
			RootKeys:              map[string]bool{},
			CanonicalToParentMeta: map[string]map[string]parentMeta{},
			ObjectOrder:           []string{},
			TopLevelObjects:       map[string]bool{},
		}

		// Use merged properties to include those coming from allOf
		if inSchema != nil {
			rt.parseContext = "input"
			// Save root input schema as the original (pre-render) to preserve $ref when possible
			if originalTasksByID != nil {
				if orig, ok := originalTasksByID[at]; ok {
					rt.RootInput = orig.Input
					injectRefsFromOriginal(inSchema, rt.RootInput)
				} else {
					rt.RootInput = t.Input
					injectRefsFromOriginal(inSchema, rt.RootInput)
				}
			} else {
				rt.RootInput = t.Input
				injectRefsFromOriginal(inSchema, rt.RootInput)
			}
			// Pass "input" as container key so objects referenced from input fields get "input" as parent
			rt.parseObjectPropertiesInto(inSchema.Properties, &rt.InputObjects, "input")
			rt.parseOneOfsProperties(inSchema.Properties)
			// Dedupe by canonical key so objects like Parts render once
			rt.InputObjects = rt.dedupeObjects(rt.InputObjects)
			// Now build input properties list after injection
			rt.Input = parseResourceProperties(inSchema)
			// Build input key set for anchor membership detection
			rt.InputKeySet = make(map[string]bool, len(rt.Input))
			for _, p := range rt.Input {
				rt.InputKeySet[p.ID] = true
			}
		}
		if outSchema != nil {
			rt.parseContext = "output"
			if originalTasksByID != nil {
				if orig, ok := originalTasksByID[at]; ok {
					rt.RootOutput = orig.Output
					injectRefsFromOriginal(outSchema, rt.RootOutput)
				} else {
					rt.RootOutput = t.Output
					injectRefsFromOriginal(outSchema, rt.RootOutput)
				}
			} else {
				rt.RootOutput = t.Output
				injectRefsFromOriginal(outSchema, rt.RootOutput)
			}
			// Pass "output" as container key so objects referenced from output fields get "output" as parent
			rt.parseObjectPropertiesInto(outSchema.Properties, &rt.OutputObjects, "output")
			rt.parseOneOfsProperties(outSchema.Properties)
			// Dedupe as well for outputs
			rt.OutputObjects = rt.dedupeObjects(rt.OutputObjects)
			// Build output properties after injection
			rt.Output = parseResourceProperties(outSchema)
			// Build output key set for anchor membership detection
			rt.OutputKeySet = make(map[string]bool, len(rt.Output))
			for _, p := range rt.Output {
				rt.OutputKeySet[p.ID] = true
			}
		}

		// Finalize canonicals across input/output using structural and root-level priority
		rt.finalizeCanonicals()

		// Build unified referenced objects list after canonical finalization
		if len(rt.InputObjects) > 0 || len(rt.OutputObjects) > 0 {
			combined := make([]map[string]objectSchema, 0, len(rt.InputObjects)+len(rt.OutputObjects))
			combined = append(combined, rt.InputObjects...)
			combined = append(combined, rt.OutputObjects...)

			// Recursively collect all referenced $defs
			referencedDefs := map[string]bool{}
			var collectRefs func(props map[string]property)
			collectRefs = func(props map[string]property) {
				for _, p := range props {
					if strings.HasPrefix(p.Items.Ref, "#/$defs/") {
						refKey := extractRefKey(p.Items.Ref)
						referencedDefs[refKey] = true
					}
					if strings.HasPrefix(p.Ref, "#/$defs/") {
						refKey := extractRefKey(p.Ref)
						referencedDefs[refKey] = true
					}
					if p.Properties != nil {
						collectRefs(p.Properties)
					}
					if p.Items.Properties != nil {
						collectRefs(p.Items.Properties)
					}
				}
			}

			// Initial scan of existing combined objects
			for _, m := range combined {
				for _, schema := range m {
					if schema.Properties != nil {
						collectRefs(schema.Properties)
					}
				}
			}
			// Also scan the original task schemas to catch $refs that were lost during merge
			if rt.RootInput != nil && rt.RootInput.Properties != nil {
				collectRefs(rt.RootInput.Properties)
			}
			if rt.RootOutput != nil && rt.RootOutput.Properties != nil {
				collectRefs(rt.RootOutput.Properties)
			}
			// Scan all original $defs to catch nested $refs
			// Sort keys for deterministic iteration
			defKeys := make([]string, 0, len(defsByKey))
			for key := range defsByKey {
				defKeys = append(defKeys, key)
			}
			sort.Strings(defKeys)

			for _, key := range defKeys {
				def := defsByKey[key]
				if def.Properties != nil {
					collectRefs(def.Properties)
				}
			}

			// Recursively add all referenced $defs until no new ones are found
			for {
				initialCount := len(referencedDefs)
				// Add missing $defs and scan them for more references
				newlyAdded := []string{}
				for refKey := range referencedDefs {
					found := false
					for _, m := range combined {
						if _, ok := m[refKey]; ok {
							found = true
							break
						}
					}
					if !found {
						if def, ok := defsByKey[refKey]; ok {
							// Skip enum objects and objects without properties from Referenced Objects section
							shouldSkip := false

							// Skip if it's an enum object (no properties but has enum values in defEnumsByKey)
							if len(def.Properties) == 0 {
								if defEnumsByKey != nil {
									if _, isEnum := defEnumsByKey[refKey]; isEnum {
										shouldSkip = true
									}
								}
								// Also skip objects that have no properties and no meaningful structure
								// BUT preserve objects with OneOf (union types) as they need their own sections
								if !shouldSkip && len(def.OneOf) == 0 && def.Description != "" {
									// Objects with only description and no structure don't need their own section
									shouldSkip = true
								}
							}

							if !shouldSkip {
								combined = append(combined, map[string]objectSchema{
									refKey: def,
								})
								newlyAdded = append(newlyAdded, refKey)
								rt.trackObjectOrder(refKey)
							}
						}
					}
				}
				// Scan newly added definitions for more $refs
				for _, refKey := range newlyAdded {
					if def, ok := defsByKey[refKey]; ok {
						if def.Properties != nil {
							collectRefs(def.Properties)
						}
					}
				}
				// If no new refs found, we're done
				if len(referencedDefs) == initialCount {
					break
				}
			}

			rt.AllObjects = rt.dedupeObjects(combined)

			// Additional deduplication: merge objects that resolve to the same header title
			rt.AllObjects = rt.dedupeByReferencedKey(rt.AllObjects)

			// Final ordering: object-level render-occasion ordering based on references
			// (This already incorporates appearance order via rt.ObjectOrder)
			rt.orderObjectsByRenderOccasion()

			// Ensure all $ref relationships are captured by scanning all final objects
			rt.scanAllObjectsForRefs()

		}

		if rt.Title = t.Title; rt.Title == "" {
			rt.Title = titleCase(componentbase.TaskIDToTitle(at))
		}

		readmeTasks[i] = rt
	}

	return readmeTasks, nil
}

func (rt *readmeTask) findReferencedObjectKey(fieldKey string) string {
	// Check input properties for $ref
	if rt.RootInput != nil && rt.RootInput.Properties != nil {
		if prop, ok := rt.RootInput.Properties[fieldKey]; ok {
			if strings.HasPrefix(prop.Ref, "#/$defs/") {
				return extractRefKey(prop.Ref)
			}
		}
	}
	// Check output properties for $ref
	if rt.RootOutput != nil && rt.RootOutput.Properties != nil {
		if prop, ok := rt.RootOutput.Properties[fieldKey]; ok {
			if strings.HasPrefix(prop.Ref, "#/$defs/") {
				return extractRefKey(prop.Ref)
			}
		}
	}
	// Check in defsByKey for nested references
	// Sort keys for deterministic iteration
	defKeys := make([]string, 0, len(defsByKey))
	for key := range defsByKey {
		defKeys = append(defKeys, key)
	}
	sort.Strings(defKeys)

	for _, key := range defKeys {
		def := defsByKey[key]
		if def.Properties != nil {
			if prop, ok := def.Properties[fieldKey]; ok {
				if strings.HasPrefix(prop.Ref, "#/$defs/") {
					return extractRefKey(prop.Ref)
				}
			}
		}
	}
	return ""
}

func (rt *readmeTask) finalizeCanonicals() {
	// Group keys by structural signature and record first-seen order
	sigToKeys := map[string][]string{}
	keyOrder := map[string]int{}
	order := 0
	addFrom := func(list []map[string]objectSchema) {
		for _, m := range list {
			for key, schema := range m {
				sig := buildObjectSignature(schema.Properties)

				// Try to find the referenced object key for $ref resolutions
				referencedKey := rt.findReferencedObjectKey(key)
				if referencedKey != "" {
					// Use the referenced object key in signature for proper merging
					sig = fmt.Sprintf("%s|ref:%s", sig, referencedKey)
				} else {
					// Use the object key itself for base definitions
					sig = fmt.Sprintf("%s|key:%s", sig, key)
				}

				sigToKeys[sig] = append(sigToKeys[sig], key)
				if _, exists := keyOrder[key]; !exists {
					keyOrder[key] = order
					order++
				}
			}
		}
	}
	addFrom(rt.InputObjects)
	addFrom(rt.OutputObjects)

	// Decide best canonical per signature and merge parents/meta
	// Sort signatures for deterministic processing
	signatures := make([]string, 0, len(sigToKeys))
	for sig := range sigToKeys {
		signatures = append(signatures, sig)
	}
	sort.Strings(signatures)

	for _, sig := range signatures {
		keys := sigToKeys[sig]
		if len(keys) == 0 {
			continue
		}
		sort.SliceStable(keys, func(i, j int) bool { return keyOrder[keys[i]] < keyOrder[keys[j]] })
		// Build title frequency among this signature's keys
		titleFreq := map[string]int{}
		keyToTitle := map[string]string{}
		getSchema := func(k string) (objectSchema, bool) {
			if schema, ok := getObjectSchemaByKey(*rt, k); ok {
				return schema, true
			}
			return objectSchema{}, false
		}
		for _, k := range keys {
			if schema, ok := getSchema(k); ok {
				t := strings.TrimSpace(schema.Title)
				keyToTitle[k] = t
				if t != "" {
					titleFreq[t] = titleFreq[t] + 1
				}
			}
		}
		// Find majority title (most frequent, break ties by shortest then alphabetical)
		majorityTitle := ""
		bestCount := -1
		bestLen := 1 << 30
		// Sort titles for deterministic processing
		titles := make([]string, 0, len(titleFreq))
		for t := range titleFreq {
			titles = append(titles, t)
		}
		sort.Strings(titles)

		for _, t := range titles {
			c := titleFreq[t]
			// Prefer simpler, more canonical titles
			// Single words are more canonical than multi-word titles
			// Shorter titles are preferred over longer ones
			isSimpler := !strings.Contains(t, " ") && len(t) <= len(majorityTitle)
			currentIsSimpler := !strings.Contains(majorityTitle, " ")

			shouldReplace := false
			if c > bestCount {
				shouldReplace = true
			} else if c == bestCount {
				// For ties, prefer simpler titles, then shorter, then alphabetical
				if isSimpler && !currentIsSimpler {
					shouldReplace = true
				} else if isSimpler == currentIsSimpler {
					if len(t) < bestLen || (len(t) == bestLen && t < majorityTitle) {
						shouldReplace = true
					}
				}
			}

			if shouldReplace {
				bestCount = c
				bestLen = len(t)
				majorityTitle = t
			}
		}
		// Candidate set: keys whose schema.Title equals majorityTitle (if any), else all keys
		candidateKeys := keys
		if majorityTitle != "" {
			filtered := make([]string, 0, len(keys))
			for _, k := range keys {
				if keyToTitle[k] == majorityTitle {
					filtered = append(filtered, k)
				}
			}
			if len(filtered) > 0 {
				candidateKeys = filtered
			}
		}
		// Prefer earliest-seen root-level key among candidates; fallback to earliest-seen overall
		best := candidateKeys[0]
		bestOrder := 1 << 30
		for _, k := range candidateKeys {
			if rt.RootKeys[k] {
				if o, ok := keyOrder[k]; ok && o < bestOrder {
					best = k
					bestOrder = o
				}
			}
		}
		if bestOrder == 1<<30 {
			for _, k := range candidateKeys {
				if keyOrder[k] < keyOrder[best] {
					best = k
				}
			}
		}
		// Update mapping for all keys with this signature. Use best as canonical for all equivalent keys
		rt.SignatureToCanonical[sig] = best
		for _, k := range keys {
			rt.FieldKeyToCanonical[k] = best
		}
		// Merge parents/meta into best
		if rt.CanonicalToParentMeta[best] == nil {
			rt.CanonicalToParentMeta[best] = map[string]parentMeta{}
		}
		mergeMetaFrom := func(fromKey string) {
			if fromKey == "" {
				return
			}
			if metas, ok := rt.CanonicalToParentMeta[fromKey]; ok {
				for parent, m := range metas {
					existing := rt.CanonicalToParentMeta[best][parent]
					if m.IsRoot {
						existing.IsRoot = true
					}
					rt.CanonicalToParentMeta[best][parent] = existing
				}
			}
			if parents, ok := rt.CanonicalToParents[fromKey]; ok {
				if rt.CanonicalToParents[best] == nil {
					rt.CanonicalToParents[best] = map[string]bool{}
				}
				for p := range parents {
					rt.CanonicalToParents[best][p] = true
				}
			}
			if parents, ok := rt.CanonicalToParentsIn[fromKey]; ok {
				if rt.CanonicalToParentsIn[best] == nil {
					rt.CanonicalToParentsIn[best] = map[string]bool{}
				}
				for p := range parents {
					rt.CanonicalToParentsIn[best][p] = true
				}
			}
			if parents, ok := rt.CanonicalToParentsOut[fromKey]; ok {
				if rt.CanonicalToParentsOut[best] == nil {
					rt.CanonicalToParentsOut[best] = map[string]bool{}
				}
				for p := range parents {
					rt.CanonicalToParentsOut[best][p] = true
				}
			}
		}
		for _, k := range keys {
			mergeMetaFrom(k)
		}
	}
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
	ID           string
	Title        string
	Description  string
	Input        []resourceProperty
	InputObjects []map[string]objectSchema
	OneOfs       []map[string][]objectSchema
	// TitleToOneOf ensures stable ordering: object title -> its oneOf options
	TitleToOneOf          map[string][]objectSchema
	Output                []resourceProperty
	OutputObjects         []map[string]objectSchema
	SignatureToCanonical  map[string]string
	FieldKeyToCanonical   map[string]string
	CanonicalToParents    map[string]map[string]bool
	CanonicalToParentsIn  map[string]map[string]bool
	CanonicalToParentsOut map[string]map[string]bool
	parseContext          string
	RootInput             *objectSchema
	RootOutput            *objectSchema
	AllObjects            []map[string]objectSchema
	RootKeys              map[string]bool
	CanonicalToParentMeta map[string]map[string]parentMeta
	// Mapping from dedupe group to the final key used in AllObjects
	GroupToRenderedKey map[string]string
	// Key sets for quick membership checks
	InputKeySet  map[string]bool
	OutputKeySet map[string]bool
	// Track appearance order of objects for rendering
	ObjectOrder []string
	// TopLevelObjects tracks objects that should always render at top level (not nested)
	TopLevelObjects map[string]bool
}

type parentMeta struct {
	IsRoot bool
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

func parseResourceProperties(o *objectSchema) []resourceProperty {
	if o == nil {
		return []resourceProperty{}
	}

	o.Title = titleCase(o.Title)

	// We need a map first to set the Required property, then we'll
	// transform it to a slice.
	propMap := make(map[string]resourceProperty)

	// Sort keys for deterministic processing
	propKeys := make([]string, 0, len(o.Properties))
	for k := range o.Properties {
		propKeys = append(propKeys, k)
	}
	sort.Strings(propKeys)

	for _, k := range propKeys {
		op := o.Properties[k]
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
	// Sort keys for deterministic processing
	mapKeys := make([]string, 0, len(propMap))
	for k := range propMap {
		mapKeys = append(mapKeys, k)
	}
	sort.Strings(mapKeys)

	for _, k := range mapKeys {
		props[idx] = propMap[k]
		idx++
	}

	// Note: The order might not be consecutive numbers.
	slices.SortFunc(props, func(i, j resourceProperty) int {
		orderI := int(^uint(0) >> 1) // max int when uiOrder is nil
		if i.Order != nil {
			orderI = *i.Order
		}
		orderJ := int(^uint(0) >> 1)
		if j.Order != nil {
			orderJ = *j.Order
		}
		if cmp := cmp.Compare(orderI, orderJ); cmp != 0 {
			return cmp
		}
		return cmp.Compare(i.ID, j.ID)
	})

	return props
}

// processPropertyMap processes a map of properties, applying replaceDescription and replaceFormat
func processPropertyMap(props map[string]property) {
	for key := range props {
		prop := props[key]
		prop.replaceDescription()
		prop.replaceFormat()
		props[key] = prop
	}
}

// recordParentRelationship records parent-child relationships in all relevant maps
func (rt *readmeTask) recordParentRelationship(childKey, parentKey string) {
	// Record in CanonicalToParentMeta
	if rt.CanonicalToParentMeta[childKey] == nil {
		rt.CanonicalToParentMeta[childKey] = map[string]parentMeta{}
	}
	meta := rt.CanonicalToParentMeta[childKey][parentKey]
	if rt.RootKeys[parentKey] {
		meta.IsRoot = true
	}
	rt.CanonicalToParentMeta[childKey][parentKey] = meta

	// Record in CanonicalToParents
	if rt.CanonicalToParents[childKey] == nil {
		rt.CanonicalToParents[childKey] = map[string]bool{}
	}
	rt.CanonicalToParents[childKey][parentKey] = true

	// Record in context-specific maps
	switch rt.parseContext {
	case "input":
		if rt.CanonicalToParentsIn[childKey] == nil {
			rt.CanonicalToParentsIn[childKey] = map[string]bool{}
		}
		rt.CanonicalToParentsIn[childKey][parentKey] = true
	case "output":
		if rt.CanonicalToParentsOut[childKey] == nil {
			rt.CanonicalToParentsOut[childKey] = map[string]bool{}
		}
		rt.CanonicalToParentsOut[childKey][parentKey] = true
	}
}

func (rt *readmeTask) parseObjectPropertiesInto(properties map[string]property, sink *[]map[string]objectSchema, containerKey string) {
	if properties == nil {
		return
	}

	orderedKeys := sortPropertyKeysByOrder(properties)

	for _, fieldKey := range orderedKeys {
		op := properties[fieldKey]
		if op.Deprecated {
			continue
		}

		// Determine parent key for backlink collection
		parentKey := containerKey
		// If we're at the root level (input/output), mark this field as a root-level key
		if containerKey == "" || containerKey == "input" || containerKey == "output" {
			rt.RootKeys[fieldKey] = true
			// For root-level tables, use the container name as parent
			if containerKey == "input" || containerKey == "output" {
				parentKey = containerKey
			} else {
				// Legacy case: containerKey == ""
				parentKey = fieldKey
			}
		}

		if op.Type != "object" && op.Type != "array[object]" && (op.Type != "array" || op.Items.Type != "object") {
			continue
		}

		if op.Type == "object" && op.Properties == nil {
			// Handle $ref objects by resolving them and establishing parent-child relationships
			if op.Ref != "" && strings.HasPrefix(op.Ref, "#/$defs/") {
				refKey := extractRefKey(op.Ref)
				// Track the referenced object immediately when $ref is encountered
				rt.trackObjectOrder(refKey)
				if defsByKey != nil {
					if refObj, ok := defsByKey[refKey]; ok && len(refObj.Properties) > 0 {

						// Use the resolved object properties
						props := refObj.Properties
						processPropertyMap(props)

						sig := buildObjectSignature(props)
						objTitle := refObj.Title
						if objTitle == "" {
							objTitle = titleCase(strings.ReplaceAll(refKey, "-", " "))
						}

						// Include title in signature key
						sigKey := sig
						if objTitle != "" {
							sigKey = sig + "|title:" + strings.ToLower(objTitle)
						}

						canonical, exists := rt.SignatureToCanonical[sigKey]
						if exists {
							// Map canonical for this field key
							rt.FieldKeyToCanonical[fieldKey] = canonical
							rt.recordParentRelationship(canonical, parentKey)
						} else {
							// First occurrence becomes canonical - use the refKey as canonical
							rt.SignatureToCanonical[sigKey] = refKey
							rt.recordParentRelationship(refKey, parentKey)

							// Create the object in the sink
							*sink = append(*sink, map[string]objectSchema{
								refKey: {
									Title:       objTitle,
									Properties:  props,
									Description: refObj.Description,
								},
							})
							rt.trackObjectOrder(refKey)
						}

						// Recurse into the resolved object
						rt.parseObjectPropertiesInto(props, sink, refKey)
						continue
					}
				}
			}
			continue
		}

		if op.Type == "array[object]" && op.Items.Properties == nil {
			// Handle array items with $ref
			if op.Items.Ref != "" && strings.HasPrefix(op.Items.Ref, "#/$defs/") {
				refKey := extractRefKey(op.Items.Ref)
				// Track the referenced object immediately when $ref is encountered
				rt.trackObjectOrder(refKey)
				if defsByKey != nil {
					if refObj, ok := defsByKey[refKey]; ok && len(refObj.Properties) > 0 {

						// Use the resolved object properties
						props := refObj.Properties
						processPropertyMap(props)

						sig := buildObjectSignature(props)
						arrTitle := refObj.Title
						if arrTitle == "" {
							arrTitle = titleCase(strings.ReplaceAll(refKey, "-", " "))
						}

						// Include title in signature key
						sigKey := sig
						if arrTitle != "" {
							sigKey = sig + "|title:" + strings.ToLower(arrTitle)
						}

						canonical, exists := rt.SignatureToCanonical[sigKey]
						if exists {
							rt.FieldKeyToCanonical[fieldKey] = canonical
							rt.recordParentRelationship(canonical, parentKey)
						} else {
							rt.SignatureToCanonical[sigKey] = refKey
							rt.recordParentRelationship(refKey, parentKey)

							*sink = append(*sink, map[string]objectSchema{
								refKey: {
									Title:       arrTitle,
									Properties:  props,
									Description: refObj.Description,
								},
							})
							rt.trackObjectOrder(refKey)
						}

						// Recurse into the resolved object
						rt.parseObjectPropertiesInto(props, sink, refKey)
						continue
					}
				}
			}
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
			props := op.Properties
			processPropertyMap(props)

			sig := buildObjectSignature(props)
			// Title for object: use schema title directly; if $ref present, prefer referenced title
			objTitle := op.Title
			if op.Ref != "" && strings.HasPrefix(op.Ref, "#/$defs/") {
				rk := extractRefKey(op.Ref)
				objTitle = titleCase(strings.ReplaceAll(rk, "-", " "))
			}
			// include title in signature key to prevent cross-merging different titled objects
			sigKey := sig
			if objTitle != "" {
				sigKey = sig + "|title:" + strings.ToLower(objTitle)
			}
			canonical, exists := rt.SignatureToCanonical[sigKey]
			if exists {
				// map canonical for this field key
				rt.FieldKeyToCanonical[fieldKey] = canonical
				rt.recordParentRelationship(canonical, parentKey)
			} else {
				// First occurrence becomes canonical
				rt.SignatureToCanonical[sigKey] = fieldKey
				rt.recordParentRelationship(fieldKey, parentKey)
				// Title for object already derived in objTitle
				*sink = append(*sink, map[string]objectSchema{
					fieldKey: {
						Title:       objTitle,
						Properties:  props,
						Description: op.Description,
					},
				})
				rt.trackObjectOrder(fieldKey)
			}
			// Recurse: current field becomes the new container for its children
			rt.parseObjectPropertiesInto(props, sink, fieldKey)
		} else { // array of object
			// Check for $ref in array items and track immediately
			if op.Items.Ref != "" && strings.HasPrefix(op.Items.Ref, "#/$defs/") {
				refKey := extractRefKey(op.Items.Ref)
				rt.trackObjectOrder(refKey)
			}

			props := op.Items.Properties

			// Handle resolved $ref arrays that don't have properties but need parent-child relationships
			if props == nil && op.Items.Type == "object" && op.Items.Ref == "" {
				// This is likely a resolved $ref array - try to establish the relationship via post-processing
				// For now, skip processing but don't skip the relationship establishment
				continue
			}
			processPropertyMap(props)

			sig := buildObjectSignature(props)
			// Title for array of object: use items.title; if items.$ref present, prefer referenced title
			arrTitle := op.Items.Title
			if op.Items.Ref != "" && strings.HasPrefix(op.Items.Ref, "#/$defs/") {
				rk := extractRefKey(op.Items.Ref)
				arrTitle = titleCase(strings.ReplaceAll(rk, "-", " "))
			}
			// include title in signature key
			sigKey := sig
			if arrTitle != "" {
				sigKey = sig + "|title:" + strings.ToLower(arrTitle)
			}
			canonical, exists := rt.SignatureToCanonical[sigKey]
			if exists {
				rt.FieldKeyToCanonical[fieldKey] = canonical
				rt.recordParentRelationship(canonical, parentKey)
			} else {
				rt.SignatureToCanonical[sigKey] = fieldKey
				rt.recordParentRelationship(fieldKey, parentKey)
				*sink = append(*sink, map[string]objectSchema{
					fieldKey: {
						Title:      arrTitle,
						Properties: props,
						Description: func() string {
							if op.Items.Description != "" {
								return op.Items.Description
							}
							return op.Description
						}(),
					},
				})
				rt.trackObjectOrder(fieldKey)
			}

			// Recurse: current field becomes the new container for its children
			rt.parseObjectPropertiesInto(props, sink, fieldKey)
		}
	}
}

func sortPropertyKeysByOrder(properties map[string]property) []string {
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		orderI := int(^uint(0) >> 1)
		if properties[keys[i]].Order != nil {
			orderI = *properties[keys[i]].Order
		}
		orderJ := int(^uint(0) >> 1)
		if properties[keys[j]].Order != nil {
			orderJ = *properties[keys[j]].Order
		}
		if orderI != orderJ {
			return orderI < orderJ
		}
		return keys[i] < keys[j]
	})
	return keys
}

func buildObjectSignature(props map[string]property) string {
	if props == nil {
		return "{}"
	}
	// Build child signatures independent of field names to dedupe by structure
	childSigs := make([]string, 0, len(props))

	// Sort keys for deterministic processing
	propKeys := make([]string, 0, len(props))
	for k := range props {
		propKeys = append(propKeys, k)
	}
	sort.Strings(propKeys)

	for _, k := range propKeys {
		p := props[k]
		var sb strings.Builder
		// Base type - skip empty types to avoid signature pollution
		if p.Type != "" {
			sb.WriteString("t:")
			sb.WriteString(p.Type)
		}
		// Array element type/signature
		if p.Type == "array" {
			sb.WriteString("|ai:")
			sb.WriteString(p.Items.Type)
			if p.Items.Type == "object" && p.Items.Properties != nil {
				sb.WriteString("|ao:")
				sb.WriteString(buildObjectSignature(p.Items.Properties))
			}
		}
		// Object nested signature
		if p.Type == "object" && p.Properties != nil {
			sb.WriteString("|o:")
			sb.WriteString(buildObjectSignature(p.Properties))
		}
		// Presence of oneOf contributes to shape
		if len(p.OneOf) > 0 {
			sb.WriteString("|oneof:")
			// capture count of options only to avoid heavy recursion here
			sb.WriteString(fmt.Sprintf("%d", len(p.OneOf)))
		}
		sigStr := sb.String()
		if sigStr != "" { // Only include non-empty signatures
			childSigs = append(childSigs, sigStr)
		}
	}
	// Sort to make order-insensitive
	sort.Strings(childSigs)
	return "{" + strings.Join(childSigs, ",") + "}"
}

func (rt *readmeTask) parseOneOfsProperties(properties map[string]property) {
	if properties == nil {
		return
	}

	// Sort keys for deterministic processing
	propKeys := make([]string, 0, len(properties))
	for key := range properties {
		propKeys = append(propKeys, key)
	}
	sort.Strings(propKeys)

	for _, key := range propKeys {
		op := properties[key]
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
				// Only store oneOf options for array items that are not $ref references
				if op.Items.Ref == "" {
					rt.OneOfs = append(rt.OneOfs, map[string][]objectSchema{
						key: op.Items.OneOf,
					})
					if rt.TitleToOneOf != nil {
						rt.TitleToOneOf[op.Title] = op.Items.OneOf
					}
				}
			}

		}

		if op.OneOf != nil {
			// Only store oneOf options for fields that are not $ref references
			// $ref fields should have their oneOf options rendered on the target object, not here
			if op.Ref == "" {
				rt.OneOfs = append(rt.OneOfs, map[string][]objectSchema{
					key: op.OneOf,
				})
				if rt.TitleToOneOf != nil {
					rt.TitleToOneOf[op.Title] = op.OneOf
				}
			}
		}
		rt.parseOneOfsProperties(op.Properties)
	}
}

func (sc *setupConfig) parseOneOfProperties(properties map[string]property) {
	if properties == nil {
		return
	}

	// Sort keys for deterministic processing
	propKeys := make([]string, 0, len(properties))
	for key := range properties {
		propKeys = append(propKeys, key)
	}
	sort.Strings(propKeys)

	for _, key := range propKeys {
		op := properties[key]
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
	result = "<br/><small><small><details><summary>Enum values</summary><ul>"

	for i, e := range enum {
		result += fmt.Sprintf("<li>`%s`</li>", e)
		if i == length-1 {
			result += "</ul>"
		}
	}
	result += "</details></small></small>"

	return result
}

func anchorSetup(p any) string {
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
	// Skip only when there is no way to resolve item object structure
	if op.Type != "array" {
		return false
	}
	if op.Items.Type == "object" && op.Items.Properties == nil && op.Items.Ref == "" {
		// Don't skip if this might be a resolved $ref that needs parent-child relationship tracking
		// We'll let the parsing logic handle it and establish relationships via post-processing
		return false
	}
	return false
}

func anchorTaskObject(p any, task readmeTask) string {
	switch prop := p.(type) {
	case resourceProperty:
		label := prop.Title
		if label == "" {
			label = titleCase(prop.ID)
		}
		// No longer add field-level anchors - just return the label
		return label
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

	// Do not create clickable links in the Field column; Type column will carry links
	if prop.Type == "object" ||
		(prop.Type == "array" && prop.Items.Type == "object") ||
		(prop.Type == "array[object]") {
		return prop.Title
	}
	return prop.Title
}

// resolveRefTitle tries to find the title (header) for a referenced object, preferring registered sections
func resolveRefTitle(refKey string, task readmeTask) string {
	// Prefer exact key lookup
	if schema, ok := getObjectSchemaByKey(task, refKey); ok && schema.Title != "" {
		return schema.Title
	}
	// Fallback: scan by structural signature
	if schema, ok := getObjectSchemaByKey(task, refKey); ok {
		sig := buildObjectSignature(schema.Properties)
		for _, m := range append(append([]map[string]objectSchema{}, task.AllObjects...), append(task.InputObjects, task.OutputObjects...)...) {
			// Sort keys for deterministic iteration
			keys := make([]string, 0, len(m))
			for key := range m {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, key := range keys {
				s := m[key]
				if s.Title != "" && buildObjectSignature(s.Properties) == sig {
					return s.Title
				}
			}
		}
	}
	// Last resort: make a title-ish string from key
	return titleCase(strings.ReplaceAll(refKey, "-", " "))
}

// typeWithRef formats the Type column, appending linked $ref when present
func typeWithRef(prop property, task readmeTask) string {
	base := prop.Type
	if base == "" {
		base = "any"
	}
	// object with $ref (link when $ref even if Type is empty)
	if strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		// Anchor directly to the referenced key section id
		return fmt.Sprintf("object ([%s](#%s-%s))", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refKey))
	}
	// If no $ref and it's an object type, just return "object" without any reference link
	if prop.Type == "object" || (prop.Type == "" && prop.Properties != nil) {
		return "object"
	}
	// array cases
	if prop.Type == "array" || prop.Type == "array[object]" || prop.Items.Type != "" || prop.Items.Properties != nil || strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
		if strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
			refKey := extractRefKey(prop.Items.Ref)
			// Anchor directly to the referenced key section id
			return fmt.Sprintf("array[object ([%s](#%s-%s))]", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refKey))
		}
		if prop.Items.Type == "object" || prop.Items.Properties != nil || prop.Type == "array[object]" {
			if prop.Items.Properties != nil {
				// structural fallback for array item object
				sig := buildObjectSignature(prop.Items.Properties)
				for _, m := range append(append([]map[string]objectSchema{}, task.AllObjects...), append(task.InputObjects, task.OutputObjects...)...) {
					// Sort keys for deterministic iteration
					keys := make([]string, 0, len(m))
					for key := range m {
						keys = append(keys, key)
					}
					sort.Strings(keys)

					for _, key := range keys {
						s := m[key]
						if s.Title != "" && buildObjectSignature(s.Properties) == sig {
							slug := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s.Title, " ", "-"), "_", "-"))
							return fmt.Sprintf("array[object ([%s](#%s-%s))]", slug, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(s.Title))
						}
					}
				}
			}
			return "array[object]"
		}
		// primitive arrays
		if prop.Items.Type == "*" {
			return "array[any]"
		}
		if prop.Items.Type != "" {
			return "array[" + prop.Items.Type + "]"
		}
		return "array"
	}
	return base
}

// typeWithRefAny adapts for template calls that pass resourceProperty
func typeWithRefAny(p any, task readmeTask) string {
	switch v := p.(type) {
	case resourceProperty:
		// Use key-aware resolution to avoid wrong structural matches
		return typeWithRefUsingKey(v.property, v.ID, task)
	case property:
		return typeWithRef(v, task)
	default:
		return ""
	}
}

// typeWithEnumAny combines type information with enum values for display in the Type column
func typeWithEnumAny(p any, task readmeTask) string {
	var prop property
	var fieldKey string

	switch v := p.(type) {
	case resourceProperty:
		prop = v.property
		fieldKey = v.ID
	case property:
		prop = v
		fieldKey = ""
	default:
		return ""
	}

	// Get base type using existing logic
	var baseType string
	if fieldKey != "" {
		baseType = typeWithRefUsingKey(prop, fieldKey, task)
	} else {
		baseType = typeWithRef(prop, task)
	}

	// If this is a string type with enum values, append them
	if baseType == "string" && len(prop.Enum) > 0 {
		return baseType + enumValues(prop.Enum)
	}

	// For $ref enums, check if we have enum values in defEnumsByKey
	if baseType == "string" && strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		if defEnumsByKey != nil {
			if enumVals, ok := defEnumsByKey[refKey]; ok && len(enumVals) > 0 {
				return baseType + enumValues(enumVals)
			}
		}
	}

	return baseType
}

// getTypeWithEnum combines type with enum values for any property
func getTypeWithEnum(prop property, propKey string, taskObj readmeTask) string {
	// Get base type
	baseType := typeWithRefUsingKey(prop, propKey, taskObj)

	// If this is a string type with enum values, append them
	if baseType == "string" && len(prop.Enum) > 0 {
		return baseType + enumValues(prop.Enum)
	}

	// For $ref enums, check if we have enum values in defEnumsByKey
	if baseType == "string" && strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		if defEnumsByKey != nil {
			if enumVals, ok := defEnumsByKey[refKey]; ok && len(enumVals) > 0 {
				return baseType + enumValues(enumVals)
			}
		}
	}

	// Handle array types with enum items
	if strings.HasPrefix(baseType, "array[") && prop.Type == "array" {
		// Check if array items have enum values through $ref
		if strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
			refKey := extractRefKey(prop.Items.Ref)
			if defEnumsByKey != nil {
				if enumVals, ok := defEnumsByKey[refKey]; ok && len(enumVals) > 0 {
					// Replace array[object ([ref](#link))] with array[string] + enum details
					return "array[string]" + enumValues(enumVals)
				}
			}
		}
		// Check if array items have direct enum values
		if len(prop.Items.Enum) > 0 {
			return "array[string]" + enumValues(prop.Items.Enum)
		}
	}

	return baseType
}

func anchorTaskWithPropertyAndKey(prop property, taskObj readmeTask, sectionKey, key string, doLink bool) string {
	// Determine label
	label := prop.Title
	if label == "" {
		label = titleCase(key)
	}
	return label
}

func insertHeaderByObjectKey(key string, taskOrString any) string {
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

func insertHeaderByConstValue(option objectSchema, taskOrString any) string {
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

// renderObjectDetails renders an object's table and recursively renders any nested oneOf sections
func renderObjectDetails(sectionKey string, obj objectSchema, taskOrString any) string {
	var b strings.Builder

	// Skip rendering table for objects with no properties (e.g., oneOf-only objects)
	if len(obj.Properties) == 0 {
		return b.String()
	}

	// Compute stable, template-consistent property order: by uiOrder then key
	keys := make([]string, 0, len(obj.Properties))
	for k := range obj.Properties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		orderI := int(^uint(0) >> 1)
		if obj.Properties[keys[i]].Order != nil {
			orderI = *obj.Properties[keys[i]].Order
		}
		orderJ := int(^uint(0) >> 1)
		if obj.Properties[keys[j]].Order != nil {
			orderJ = *obj.Properties[keys[j]].Order
		}
		if orderI != orderJ {
			return orderI < orderJ
		}
		return keys[i] < keys[j]
	})

	// Render the object's own properties table
	b.WriteString("<div class=\"markdown-col-no-wrap\" data-col-1 data-col-2>\n\n")
	b.WriteString("| Field | Field ID | Type | Note |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")

	var taskObj readmeTask
	switch v := taskOrString.(type) {
	case readmeTask:
		taskObj = v
	case string:
		_ = v // unused in this context
	}

	for _, propKey := range keys {
		prop := obj.Properties[propKey]
		p := prop
		p.replaceDescription()
		p.replaceFormat()

		note := p.Description
		if p.Const != "" {
			note = fmt.Sprintf("Must be \"%s\"", p.Const)
		}

		// Title normalization
		if p.Title == "" {
			p.Title = titleCase(propKey)
		}

		// For enum $refs, use the referenced enum's description in Note
		if strings.HasPrefix(p.Ref, "#/$defs/") {
			refKey := extractRefKey(p.Ref)
			if def, ok := defsByKey[refKey]; ok && len(def.Properties) == 0 && def.Description != "" {
				// Apply YAML folded formatting to enum descriptions
				enumDesc := def.Description
				// Collapse single newlines to spaces, preserve double newlines as paragraph breaks
				paragraphs := strings.Split(enumDesc, "\n\n")
				for i, para := range paragraphs {
					para = strings.TrimSpace(para)
					para = strings.ReplaceAll(para, "\n", " ")
					para = strings.Join(strings.Fields(para), " ")
					paragraphs[i] = para
				}
				enumDescFormatted := strings.Join(paragraphs, " ")
				// If we already added enum details, preserve them
				if strings.Contains(note, "<details>") {
					// Replace just the description part, keep the <details>
					if enumVals, ok := defEnumsByKey[refKey]; ok && len(enumVals) > 0 {
						note = enumDescFormatted + " " + enumValues(enumVals)
					} else {
						note = enumDescFormatted
					}
				} else {
					note = enumDescFormatted
				}
			}
		}

		// Get type with enum values for display in Type column
		typeLabel := getTypeWithEnum(p, propKey, taskObj)
		hasObjectLink := strings.Contains(typeLabel, "object([") || strings.Contains(typeLabel, "object ([")
		fieldLabel := anchorTaskWithPropertyAndKey(p, taskObj, sectionKey, propKey, hasObjectLink)

		fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n", fieldLabel, propKey, typeLabel, note)
	}
	b.WriteString("</div>\n")

	// Recursively render OneOfs AFTER the table, per property in the same order
	for _, propKey := range keys {
		prop := obj.Properties[propKey]
		// Inline property-level oneOf only (do not render array item oneOf here)
		// Skip oneOf rendering for resolved $ref properties - they should be rendered on the target object instead
		// A resolved $ref property has OneOf but no Properties and a Title that matches the target object
		shouldSkipOneOf := false
		if prop.OneOf != nil && len(prop.Properties) == 0 && prop.Title != "" {
			// This looks like a resolved $ref to a union type - skip rendering oneOf here
			shouldSkipOneOf = true
		}

		if prop.OneOf != nil && !shouldSkipOneOf {
			b.WriteString("\nalso including one of the fields:\n\n")
			// Qualify anchors by the current object section key (e.g., part) to avoid using the property name (e.g., parts)
			b.WriteString(renderOneOfOptionsAsSingleTable(prop.OneOf, []string{sectionKey}, taskOrString))
		}
	}

	return b.String()
}

// renderOneOfOptionsAsSingleTable renders a single consolidated table from all oneOf option properties
func renderOneOfOptionsAsSingleTable(options []objectSchema, parentChain []string, taskOrString interface{}) string {
	var b strings.Builder
	var taskObj readmeTask
	switch v := taskOrString.(type) {
	case readmeTask:
		taskObj = v
	case string:
		_ = v // unused here
	}

	b.WriteString("<div class=\"markdown-col-no-wrap\" data-col-1 data-col-2>\n\n")
	b.WriteString("| Field | Field ID | Type | Note |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")

	// Merge properties from all options and sort by uiOrder then key
	type merged struct {
		prop  property
		key   string
		order int
	}
	mergedMap := make(map[string]merged)
	for _, option := range options {
		// Sort keys for deterministic processing
		optionKeys := make([]string, 0, len(option.Properties))
		for propKey := range option.Properties {
			optionKeys = append(optionKeys, propKey)
		}
		sort.Strings(optionKeys)

		for _, propKey := range optionKeys {
			prop := option.Properties[propKey]
			p := prop
			p.replaceDescription()
			p.replaceFormat()
			ord := 0
			if p.Order == nil {
				ord = int(^uint(0) >> 1)
			} else {
				ord = *p.Order
			}
			if existing, ok := mergedMap[propKey]; ok {
				// Keep lower order if present
				if ord < existing.order {
					existing.order = ord
				}
				// Prefer filled title/type/desc if empty in existing
				if existing.prop.Title == "" && p.Title != "" {
					existing.prop.Title = p.Title
				}
				if existing.prop.Type == "" && p.Type != "" {
					existing.prop.Type = p.Type
				}
				if existing.prop.Description == "" && p.Description != "" {
					existing.prop.Description = p.Description
				}
				if len(existing.prop.Enum) == 0 && len(p.Enum) > 0 {
					existing.prop.Enum = p.Enum
				}
				mergedMap[propKey] = existing
			} else {
				mergedMap[propKey] = merged{prop: p, key: propKey, order: ord}
			}
		}
	}
	// Sort keys for deterministic processing
	keys := make([]merged, 0, len(mergedMap))
	mapKeys := make([]string, 0, len(mergedMap))
	for k := range mergedMap {
		mapKeys = append(mapKeys, k)
	}
	sort.Strings(mapKeys)

	for _, k := range mapKeys {
		keys = append(keys, mergedMap[k])
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].order != keys[j].order {
			return keys[i].order < keys[j].order
		}
		return keys[i].key < keys[j].key
	})

	// Track nested object properties to render their tables after the consolidated table
	type nestedObj struct {
		key     string
		obj     objectSchema
		desc    string
		parents []string
	}
	nestedQueue := make([]nestedObj, 0)

	for _, m := range keys {
		p := m.prop
		note := p.Description
		if p.Const != "" {
			note = fmt.Sprintf("Must be \"%s\"", p.Const)
		}
		if p.Title == "" {
			p.Title = titleCase(m.key)
		}
		// Always link in oneOf tables for object/array[object] fields using field id
		sectionKey := ""
		if len(parentChain) > 0 {
			sectionKey = parentChain[len(parentChain)-1]
		}
		// Prefer singular form if it corresponds to an object section (e.g., parts -> part)
		if sectionKey != "" {
			singular := singularize(sectionKey)
			if singular != sectionKey && hasObjectSection(taskObj, singular) {
				sectionKey = singular
			}
		}
		// Get type with enum values for display in Type column
		typeLabel := getTypeWithEnum(p, m.key, taskObj)
		hasObjectLink := strings.Contains(typeLabel, "object([") || strings.Contains(typeLabel, "object ([")
		fieldLabel := anchorTaskWithPropertyAndKey(p, taskObj, sectionKey, m.key, hasObjectLink)
		fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n", fieldLabel, m.key, typeLabel, note)

		// Queue nested object schemas for rendering after the consolidated table
		if p.Type == "object" && p.Properties != nil {
			nestedQueue = append(nestedQueue, nestedObj{
				key:  m.key,
				obj:  objectSchema{Properties: p.Properties, Title: p.Title, Description: p.Description},
				desc: p.Description,
				parents: func() []string {
					if sectionKey != "" {
						return []string{sectionKey + ":" + m.key}
					}
					return []string{m.key}
				}(),
			})
		} else if p.Type == "array" && p.Items.Type == "object" && p.Items.Properties != nil {
			nestedQueue = append(nestedQueue, nestedObj{
				key:  m.key,
				obj:  objectSchema{Properties: p.Items.Properties, Title: p.Title, Description: p.Description},
				desc: p.Description,
				parents: func() []string {
					if sectionKey != "" {
						return []string{sectionKey + ":" + m.key}
					}
					return []string{m.key}
				}(),
			})
		}
	}
	b.WriteString("</div>\n")

	// Render nested object tables immediately after the consolidated oneOf table, in the same sorted order
	for _, nested := range nestedQueue {
		if t, ok := taskOrString.(readmeTask); ok {
			if hasObjectSection(t, nested.key) {
				// Already rendered in unified section; skip duplicate nested table
				continue
			}
		}
		b.WriteString(insertHeaderWithParents(nested.key, nested.parents, taskOrString))
		if nested.desc != "" {
			b.WriteString("\n")
			b.WriteString(nested.desc)
			b.WriteString("\n\n")
		}
		b.WriteString(renderObjectDetails(nested.key, nested.obj, taskOrString))
		b.WriteString(renderNestedObjectsRecursively(nested.obj.Properties, nested.key, taskOrString))
	}

	return b.String()
}

// renderNestedObjectsRecursively walks properties and renders child object tables depth-first in order
func renderNestedObjectsRecursively(properties map[string]property, parentKey string, taskOrString interface{}) string {
	if properties == nil {
		return ""
	}
	// sort keys by uiOrder then key
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		orderI := int(^uint(0) >> 1)
		if properties[keys[i]].Order != nil {
			orderI = *properties[keys[i]].Order
		}
		orderJ := int(^uint(0) >> 1)
		if properties[keys[j]].Order != nil {
			orderJ = *properties[keys[j]].Order
		}
		if orderI != orderJ {
			return orderI < orderJ
		}
		return keys[i] < keys[j]
	})

	var b strings.Builder
	for _, key := range keys {
		prop := properties[key]
		if prop.Type == "object" && prop.Properties != nil {
			if t, ok := taskOrString.(readmeTask); ok {
				// Skip nested rendering if object has its own section OR is marked as top-level
				if hasObjectSection(t, key) || shouldRenderAtTopLevel(t, key) {
					continue
				}
			}
			child := objectSchema{Properties: prop.Properties, Title: prop.Title, Description: prop.Description}
			// Link back to the row anchor of this property in the parent's table
			b.WriteString(insertHeaderWithParents(key, []string{parentKey + ":" + key}, taskOrString))
			if prop.Description != "" {

				b.WriteString(prop.Description)
				b.WriteString("\n\n")
			}
			b.WriteString(renderObjectDetails(key, child, taskOrString))
			b.WriteString(renderNestedObjectsRecursively(child.Properties, key, taskOrString))
		} else if prop.Type == "array" && prop.Items.Type == "object" && prop.Items.Properties != nil {
			if t, ok := taskOrString.(readmeTask); ok {
				// Skip nested rendering if object has its own section OR is marked as top-level
				if hasObjectSection(t, key) || shouldRenderAtTopLevel(t, key) {
					continue
				}
			}
			child := objectSchema{Properties: prop.Items.Properties, Title: prop.Title, Description: prop.Description}
			// Link back to the row anchor of this property in the parent's table
			b.WriteString(insertHeaderWithParents(key, []string{parentKey + ":" + key}, taskOrString))
			if prop.Description != "" {

				b.WriteString(prop.Description)
				b.WriteString("\n\n")
			}
			b.WriteString(renderObjectDetails(key, child, taskOrString))
			b.WriteString(renderNestedObjectsRecursively(child.Properties, key, taskOrString))
		}
	}
	return b.String()
}

// renderObjectDetailsWithKey inserts the header by key then renders details
func renderObjectDetailsWithKey(key string, obj objectSchema, taskOrString any) string {
	var b strings.Builder

	// Header is handled in the template to place description right below the title
	// Prefer the object's title for anchor qualification so row anchors match the header id (e.g., part)
	sectionAnchorKey := key
	if strings.TrimSpace(obj.Title) != "" {
		sectionAnchorKey = obj.Title
	}

	b.WriteString(renderObjectDetails(sectionAnchorKey, obj, taskOrString))

	// Handle object-level oneOf first (for union types)
	if len(obj.OneOf) > 0 {
		b.WriteString("\nalso including one of the fields:\n\n")
		b.WriteString(renderOneOfOptionsAsSingleTable(obj.OneOf, []string{sectionAnchorKey}, taskOrString))
		return b.String()
	}

	// Only render oneOf options for objects that don't have regular properties
	// This prevents objects that reference union types from showing the union's options
	if len(obj.Properties) > 0 {
		// This object has its own properties, so it's not a pure union type
		// Don't render any associated oneOf options
		return b.String()
	}

	// Append OneOfs associated with this key (prefer match by Title via TitleToOneOf), AFTER the object's table
	if task, ok := taskOrString.(readmeTask); ok {
		if task.TitleToOneOf != nil {
			lookupKeys := []string{}
			if strings.TrimSpace(obj.Title) != "" {
				lookupKeys = append(lookupKeys, obj.Title)
			}
			lookupKeys = append(lookupKeys, key)
			for _, lk := range lookupKeys {
				if options, exists := task.TitleToOneOf[lk]; exists {
					b.WriteString("\nalso including one of the fields:\n\n")
					b.WriteString(renderOneOfOptionsAsSingleTable(options, []string{sectionAnchorKey}, taskOrString))
					return b.String()
				}
			}
		}
		for _, oneOfMap := range task.OneOfs {
			lookupKeys := []string{key}
			for _, lk := range lookupKeys {
				if options, exists := oneOfMap[lk]; exists {
					b.WriteString("\nalso including one of the fields:\n\n")
					b.WriteString(renderOneOfOptionsAsSingleTable(options, []string{sectionAnchorKey}, taskOrString))
				}
			}
		}
	}

	return b.String()
}

func (prop *property) replaceDescription() {
	// Normalize YAML block scalars (|, >, |+, |-, >+, >-).
	// We do not have scalar style metadata here, so we approximate YAML folding semantics:
	// - Treat single newlines as soft wraps (fold to spaces)
	// - Preserve paragraph breaks (double newlines)
	// This matches common usage of ">" and ">-" while keeping paragraphs intact.
	if strings.Contains(prop.Description, "{{") && strings.Contains(prop.Description, "}}") {
		prop.Description = strings.ReplaceAll(prop.Description, "{{", "`{{")
		prop.Description = strings.ReplaceAll(prop.Description, "}}", "}}`")
	}

	// Escape curly braces for Markdown safety while preserving content
	prop.Description = strings.ReplaceAll(prop.Description, "{", "\\{")
	prop.Description = strings.ReplaceAll(prop.Description, "}", "\\}")

	// Approximate YAML folded style: collapse single newlines within paragraphs to spaces,
	// preserving blank lines as paragraph separators.
	if prop.Description != "" {
		paragraphs := strings.Split(prop.Description, "\n\n")
		for i, p := range paragraphs {
			// Trim trailing whitespace in each line and fold remaining single newlines to spaces
			p = strings.TrimRight(p, " \t\n")
			p = strings.ReplaceAll(p, "\n", " ")
			// Collapse multiple spaces
			p = strings.Join(strings.Fields(p), " ")
			paragraphs[i] = p
		}
		prop.Description = strings.Join(paragraphs, "\n\n")
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

func insertHeaderWithParents(key string, parents []string, taskOrString any) string {
	var prefix string
	var taskObj readmeTask
	switch v := taskOrString.(type) {
	case readmeTask:
		prefix = blackfriday.SanitizedAnchorName(v.Title)
		taskObj = v
	case string:
		prefix = blackfriday.SanitizedAnchorName(v)
	default:
		prefix = "unknown"
	}

	// Determine header id from schema title if available
	headerKey := key
	if taskObj.Title != "" {
		if schema, ok := getObjectSchemaByKey(taskObj, key); ok {
			if schema.Title != "" {
				headerKey = schema.Title
			}
		}
	}

	// Prefer referenced title when this key is a $ref
	refTitle := ""
	if taskObj.RootInput != nil && taskObj.RootInput.Properties != nil {
		if p, ok := taskObj.RootInput.Properties[key]; ok {
			r := p.Ref
			if r == "" {
				r = p.Items.Ref
			}
			if strings.HasPrefix(r, "#/$defs/") {
				rk := extractRefKey(r)
				refTitle = titleCase(strings.ReplaceAll(rk, "-", " "))
			}
		}
	}
	if refTitle == "" && taskObj.RootOutput != nil && taskObj.RootOutput.Properties != nil {
		if p, ok := taskObj.RootOutput.Properties[key]; ok {
			r := p.Ref
			if r == "" {
				r = p.Items.Ref
			}
			if strings.HasPrefix(r, "#/$defs/") {
				rk := extractRefKey(r)
				refTitle = titleCase(strings.ReplaceAll(rk, "-", " "))
			}
		}
	}

	// If still not found, scan nested object schemas for a property with this key that is a $ref
	if refTitle == "" {
	outer:
		for _, m := range append(append([]map[string]objectSchema{}, taskObj.AllObjects...), append(taskObj.InputObjects, taskObj.OutputObjects...)...) {
			// Sort keys for deterministic iteration
			objKeys := make([]string, 0, len(m))
			for key := range m {
				objKeys = append(objKeys, key)
			}
			sort.Strings(objKeys)

			for _, objKey := range objKeys {
				os := m[objKey]
				if os.Properties == nil {
					continue
				}

				// Sort property keys for deterministic iteration
				propKeys := make([]string, 0, len(os.Properties))
				for pk := range os.Properties {
					propKeys = append(propKeys, pk)
				}
				sort.Strings(propKeys)

				for _, pk := range propKeys {
					pp := os.Properties[pk]
					// match on original key or its canonicalized mapping
					matchKey := pk
					if c, ok := taskObj.FieldKeyToCanonical[pk]; ok {
						matchKey = c
					}
					if matchKey != key && pk != key {
						continue
					}
					r := pp.Ref
					if r == "" {
						r = pp.Items.Ref
					}
					if strings.HasPrefix(r, "#/$defs/") {
						rk := extractRefKey(r)
						refTitle = titleCase(strings.ReplaceAll(rk, "-", " "))
						break outer
					}
				}
			}
		}
	}
	if refTitle != "" {
		headerKey = refTitle
	}

	// Get all parents using existing metadata
	var backlinkParents []string
	if taskObj.Title != "" {
		allParents := getParents(key, taskObj, "")
		backlinkParents = append(backlinkParents, allParents...)

	}

	// Build backlink text in new format
	backlinkText := ""
	if len(backlinkParents) > 0 {
		containerTitles := make([]string, 0)
		seen := make(map[string]bool)

		for _, parent := range backlinkParents {
			var containerTitle string
			var href string

			switch parent {
			case "input":
				containerTitle = "Input"
				href = fmt.Sprintf("#%s-input", prefix)
			case "output":
				containerTitle = "Output"
				href = fmt.Sprintf("#%s-output", prefix)
			default:
				// For nested containers, get the container's title and use proper anchor
				if schema, ok := getObjectSchemaByKey(taskObj, parent); ok && schema.Title != "" {
					containerTitle = schema.Title
					// Use the schema title for the anchor, not the parent key
					href = fmt.Sprintf("#%s-%s", prefix, blackfriday.SanitizedAnchorName(schema.Title))
				} else {
					// Generic handling for all objects
					containerTitle = titleCase(strings.ReplaceAll(parent, "-", " "))
					href = fmt.Sprintf("#%s-%s", prefix, blackfriday.SanitizedAnchorName(containerTitle))
				}
			}

			if !seen[containerTitle] {
				seen[containerTitle] = true
				containerTitles = append(containerTitles, fmt.Sprintf(`<a href="%s">%s</a>`, href, containerTitle))
			}
		}

		if len(containerTitles) > 0 {
			backlinkText = fmt.Sprintf(" <sup><small><small>Referenced in %s</small></small></sup>", strings.Join(containerTitles, ", "))
		}
	}

	// Determine display title
	displayTitle := titleCase(strings.ReplaceAll(strings.ReplaceAll(key, "-", " "), "_", " "))
	if refTitle != "" {
		displayTitle = refTitle
	} else if taskObj.Title != "" {
		if schema, ok := getObjectSchemaByKey(taskObj, key); ok {
			if schema.Title != "" {
				displayTitle = schema.Title
			}
		}
	}

	headerID := fmt.Sprintf("%s-%s", prefix, blackfriday.SanitizedAnchorName(headerKey))

	return fmt.Sprintf(
		`<h4 id="%s">%s%s</h4>`,
		headerID,
		displayTitle,
		backlinkText,
	)
}

func makeParents(keys ...string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k)
	}
	return out
}

func getParents(key string, task readmeTask, scope string) []string {

	// resolve canonical
	if c, ok := task.FieldKeyToCanonical[key]; ok {
		key = c
	}

	// Use canonical meta (scope-agnostic), but keep existing maps as fallback
	metas := task.CanonicalToParentMeta[key]
	if len(metas) == 0 {
		// Fallback to previous logic
		var m map[string]bool
		switch scope {
		case "input":
			m = task.CanonicalToParentsIn[key]
		case "output":
			m = task.CanonicalToParentsOut[key]
		default:
			m = task.CanonicalToParents[key]
		}
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	// Build candidate list
	candidates := make([]string, 0, len(metas))
	for k := range metas {
		candidates = append(candidates, k)
	}
	sort.Strings(candidates)

	// Consolidate field-level parents into container-level parents
	consolidated := make([]string, 0)
	seen := make(map[string]bool)

	// Check if this object is directly referenced by any input/output field
	hasDirectInputRef := false
	hasDirectOutputRef := false

	if task.RootInput != nil && task.RootInput.Properties != nil {
		for _, fieldProp := range task.RootInput.Properties {
			// Check if this field directly references our object
			if (fieldProp.Ref != "" && extractRefKey(fieldProp.Ref) == key) ||
				(fieldProp.Type == "array" && fieldProp.Items.Ref != "" && extractRefKey(fieldProp.Items.Ref) == key) {
				hasDirectInputRef = true
				break
			}
		}
	}

	if task.RootOutput != nil && task.RootOutput.Properties != nil {
		for _, fieldProp := range task.RootOutput.Properties {
			// Check if this field directly references our object
			if (fieldProp.Ref != "" && extractRefKey(fieldProp.Ref) == key) ||
				(fieldProp.Type == "array" && fieldProp.Items.Ref != "" && extractRefKey(fieldProp.Items.Ref) == key) {
				hasDirectOutputRef = true
				break
			}
		}
	}

	// Now process candidates
	for _, parent := range candidates {
		// Only consolidate field-level parents if this object is DIRECTLY referenced by input/output
		if hasDirectInputRef && task.InputKeySet != nil && task.InputKeySet[parent] {
			if !seen["input"] {
				consolidated = append(consolidated, "input")
				seen["input"] = true
			}
		} else if hasDirectOutputRef && task.OutputKeySet != nil && task.OutputKeySet[parent] {
			if !seen["output"] {
				consolidated = append(consolidated, "output")
				seen["output"] = true
			}
		} else {
			// For objects not directly referenced by input/output, apply smart filtering:
			// - Skip pure field-level parents (field names that don't also represent objects)
			// - Keep object-level parents (even if they're also field names)

			isPureFieldParent := false

			// Check if this is a pure field-level parent by seeing if it's in InputKeySet/OutputKeySet
			// but NOT in the object definitions (meaning it's only a field, not an object)
			isInInputKeySet := task.InputKeySet != nil && task.InputKeySet[parent]
			isInOutputKeySet := task.OutputKeySet != nil && task.OutputKeySet[parent]

			// Check if this parent also represents an object (has its own definition)
			isObjectDefinition := false
			for _, objectMap := range task.AllObjects {
				if _, exists := objectMap[parent]; exists {
					isObjectDefinition = true
					break
				}
			}

			// It's a pure field parent if it's in KeySet but not an object definition
			isPureFieldParent = (isInInputKeySet || isInOutputKeySet) && !isObjectDefinition

			if !isPureFieldParent && !seen[parent] {
				// This is an object-level parent (or mixed), keep it
				consolidated = append(consolidated, parent)
				seen[parent] = true
			}
		}
	}

	sort.Strings(consolidated)
	return consolidated
}

func (rt *readmeTask) dedupeObjects(list []map[string]objectSchema) []map[string]objectSchema {
	// Collapse by structural signature canonical to avoid duplicate tables across input/output/titles
	if len(list) == 0 {
		return list
	}
	groupOrder := make([]string, 0)
	sigToGroup := map[string]string{}
	groupToCanon := map[string]string{}
	best := map[string]objectSchema{}
	// Track per-group schemas by original key so we can prefer canonical's own title
	groupKeySchemas := map[string]map[string]objectSchema{}
	// Track the earliest appearance position for each group to preserve appearance order
	groupToEarliestPosition := map[string]int{}

	for _, m := range list {
		// Sort keys for deterministic processing
		mapKeys := make([]string, 0, len(m))
		for key := range m {
			mapKeys = append(mapKeys, key)
		}
		sort.Strings(mapKeys)

		for _, key := range mapKeys {
			schema := m[key]
			sig := buildObjectSignature(schema.Properties)

			// Use referenced object key in signature for proper merging of $ref resolutions
			referencedKey := rt.findReferencedObjectKey(key)
			var group string
			if referencedKey != "" {
				// Use the referenced object key in signature for proper merging
				// Also override the schema title to use the referenced object's title
				if defsByKey != nil {
					if refSchema, ok := defsByKey[referencedKey]; ok && refSchema.Title != "" {
						schema.Title = refSchema.Title
					}
				}
				group = fmt.Sprintf("%s|ref:%s", sig, referencedKey)
			} else {
				// Use the object key itself for base definitions
				group = fmt.Sprintf("%s|key:%s", sig, key)
			}

			// Enhanced grouping: for objects with titles, use title-only grouping
			// This consolidates objects with same title regardless of structural differences
			if schema.Title != "" && len(schema.Properties) > 0 {
				titleKey := strings.ToLower(strings.TrimSpace(schema.Title))
				// For objects with substantial structure, use title as the only grouping key
				if len(schema.Properties) >= 2 {
					// Use title as the only grouping key to force consolidation
					group = fmt.Sprintf("title:%s", titleKey)
				}
			}

			if _, ok := sigToGroup[group]; !ok {
				sigToGroup[group] = group
				groupOrder = append(groupOrder, group)
				// Track the earliest appearance position for this group
				if pos, exists := rt.getObjectOrderPosition(key); exists {
					groupToEarliestPosition[group] = pos
				}
			} else {
				// Update to earlier position if this key appears earlier
				if pos, exists := rt.getObjectOrderPosition(key); exists {
					if existingPos, hasPos := groupToEarliestPosition[group]; !hasPos || pos < existingPos {
						groupToEarliestPosition[group] = pos
					}
				}
			}
			// Record schema by key for this group
			if groupKeySchemas[group] == nil {
				groupKeySchemas[group] = map[string]objectSchema{}
			}
			groupKeySchemas[group][key] = schema
			// Choose canonical key for this group: prefer earliest root-level occurrence
			if curr, ok := groupToCanon[group]; !ok {
				groupToCanon[group] = key
			} else if !rt.RootKeys[curr] && rt.RootKeys[key] {
				groupToCanon[group] = key
			}
			// Update canonical mapping for this key
			rt.FieldKeyToCanonical[key] = groupToCanon[group]

			// Pick best schema per group: prefer non-empty title/description and richer properties
			if existing, ok := best[group]; ok {
				if existing.Title == "" && schema.Title != "" {
					existing.Title = schema.Title
				}
				if len(existing.Properties) < len(schema.Properties) {
					existing.Properties = schema.Properties
				}
				if existing.Description == "" && schema.Description != "" {
					existing.Description = schema.Description
				}
				// Preserve OneOf from either schema
				if len(existing.OneOf) == 0 && len(schema.OneOf) > 0 {
					existing.OneOf = schema.OneOf
				}
				best[group] = existing
			} else {
				best[group] = objectSchema{Title: schema.Title, Properties: schema.Properties, Description: schema.Description, OneOf: schema.OneOf}
			}
		}
	}

	// Emit canonical only, avoiding duplicate sections
	seenCanon := map[string]bool{}
	seenTitle := map[string]bool{}
	out := make([]map[string]objectSchema, 0, len(best))

	// Sort groups by their earliest appearance position to preserve appearance order
	sort.Slice(groupOrder, func(i, j int) bool {
		posI, hasI := groupToEarliestPosition[groupOrder[i]]
		posJ, hasJ := groupToEarliestPosition[groupOrder[j]]

		// If both have positions, sort by position
		if hasI && hasJ {
			return posI < posJ
		}
		// If only one has position, prioritize it
		if hasI && !hasJ {
			return true
		}
		if !hasI && hasJ {
			return false
		}
		// If neither has position, fall back to string comparison
		return groupOrder[i] < groupOrder[j]
	})

	for _, group := range groupOrder {
		canon := groupToCanon[group]
		if seenCanon[canon] {
			continue
		}
		// Use canonical key's own title/description to avoid cross-title bleed
		finalSchema := best[group]
		if perKey, ok := groupKeySchemas[group]; ok {
			if s, ok2 := perKey[canon]; ok2 {
				if strings.TrimSpace(s.Title) != "" {
					finalSchema.Title = s.Title
				}
				if strings.TrimSpace(s.Description) != "" {
					finalSchema.Description = s.Description
				}
				// Properties are structurally equivalent; keep best[group].Properties
			}
		}
		// Skip if we've already seen this title to avoid duplicates
		// BUT: never skip critical objects
		titleKey := strings.ToLower(strings.TrimSpace(finalSchema.Title))
		if titleKey != "" && seenTitle[titleKey] && !rt.isCriticalObject(canon) {
			continue
		}
		seenCanon[canon] = true
		if titleKey != "" {
			seenTitle[titleKey] = true
		}
		// Always include the object, even if it has no properties (for enums)
		out = append(out, map[string]objectSchema{canon: finalSchema})
	}
	return out
}

// dedupeByReferencedKey removes objects that reference the same $defs key
func (rt *readmeTask) dedupeByReferencedKey(list []map[string]objectSchema) []map[string]objectSchema {
	if len(list) == 0 {
		return list
	}

	// Track seen referenced keys across ALL maps
	seenRefKeys := make(map[string]string)          // referencedKey -> bestKey
	refKeyToSchema := make(map[string]objectSchema) // referencedKey -> bestSchema
	refKeyToEarliestPos := make(map[string]int)     // referencedKey -> earliest position
	out := make([]map[string]objectSchema, 0, len(list))

	// First pass: find the best (most canonical) object for each referenced key
	for _, objMap := range list {
		for key, schema := range objMap {
			referencedKey := rt.getReferencedKey(key, schema)

			if existingKey, exists := seenRefKeys[referencedKey]; !exists {
				// First time seeing this referenced key
				seenRefKeys[referencedKey] = key
				refKeyToSchema[referencedKey] = schema
				// Track earliest position
				if pos, hasPos := rt.getObjectOrderPosition(key); hasPos {
					refKeyToEarliestPos[referencedKey] = pos
				}
			} else {
				// Referenced key already seen, determine which one to keep based on generic criteria
				shouldReplace := rt.shouldReplaceWithGenericCriteria(key, existingKey, referencedKey, refKeyToEarliestPos)

				if shouldReplace {
					seenRefKeys[referencedKey] = key
					refKeyToSchema[referencedKey] = schema
					// Update earliest position
					if pos, hasPos := rt.getObjectOrderPosition(key); hasPos {
						refKeyToEarliestPos[referencedKey] = pos
					}
				}
			}
		}
	}

	// Second pass: collect all chosen objects with their positions
	type objectWithPosition struct {
		objMap   map[string]objectSchema
		position int
	}
	var objectsWithPos []objectWithPosition

	for _, objMap := range list {
		filteredMap := make(map[string]objectSchema)
		minPosition := math.MaxInt

		for key, schema := range objMap {
			referencedKey := rt.getReferencedKey(key, schema)

			// Only include if this is the chosen canonical object for this referenced key
			if seenRefKeys[referencedKey] == key {
				filteredMap[key] = schema
				// Track the minimum position for this object map
				if pos, hasPos := refKeyToEarliestPos[referencedKey]; hasPos && pos < minPosition {
					minPosition = pos
				}
			}
		}
		if len(filteredMap) > 0 {
			objectsWithPos = append(objectsWithPos, objectWithPosition{
				objMap:   filteredMap,
				position: minPosition,
			})
		}
	}

	// Sort by appearance order
	sort.Slice(objectsWithPos, func(i, j int) bool {
		return objectsWithPos[i].position < objectsWithPos[j].position
	})

	// Build final output in appearance order
	for _, obj := range objectsWithPos {
		out = append(out, obj.objMap)
	}
	return out
}

// getReferencedKey determines what $defs key this object references (for deduplication)
func (rt *readmeTask) getReferencedKey(key string, schema objectSchema) string {
	// Check if this key is found in root input/output properties and has a $ref
	if rt.RootInput != nil && rt.RootInput.Properties != nil {
		if p, ok := rt.RootInput.Properties[key]; ok {
			r := p.Ref
			if r == "" {
				r = p.Items.Ref
			}
			if strings.HasPrefix(r, "#/$defs/") {
				return extractRefKey(r)
			}
		}
	}
	if rt.RootOutput != nil && rt.RootOutput.Properties != nil {
		if p, ok := rt.RootOutput.Properties[key]; ok {
			r := p.Ref
			if r == "" {
				r = p.Items.Ref
			}
			if strings.HasPrefix(r, "#/$defs/") {
				return extractRefKey(r)
			}
		}
	}

	// Check nested object schemas for a property with this key that has a $ref
	for _, m := range append(append([]map[string]objectSchema{}, rt.AllObjects...), append(rt.InputObjects, rt.OutputObjects...)...) {
		// Sort keys for deterministic iteration
		objKeys := make([]string, 0, len(m))
		for key := range m {
			objKeys = append(objKeys, key)
		}
		sort.Strings(objKeys)

		for _, objKey := range objKeys {
			os := m[objKey]
			if os.Properties == nil {
				continue
			}

			// Sort property keys for deterministic iteration
			propKeys := make([]string, 0, len(os.Properties))
			for pk := range os.Properties {
				propKeys = append(propKeys, pk)
			}
			sort.Strings(propKeys)

			for _, pk := range propKeys {
				pp := os.Properties[pk]
				// match on original key or its canonicalized mapping
				matchKey := pk
				if c, ok := rt.FieldKeyToCanonical[pk]; ok {
					matchKey = c
				}
				if matchKey != key && pk != key {
					continue
				}
				r := pp.Ref
				if r == "" {
					r = pp.Items.Ref
				}
				if strings.HasPrefix(r, "#/$defs/") {
					return extractRefKey(r)
				}
			}
		}
	}

	// If no $ref found, use a normalized key based on the title for consolidation
	// This ensures objects with same title but different keys get consolidated
	if schema.Title != "" {
		// Use the title as the basis for the referenced key
		// This allows objects with same title to be consolidated
		titleKey := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(schema.Title), " ", "-"))
		return titleKey
	}

	return key
}

// shouldReplaceWithGenericCriteria determines if a new key should replace an existing key
// Uses simple appearance order - first encountered wins
func (rt *readmeTask) shouldReplaceWithGenericCriteria(newKey, existingKey, referencedKey string, refKeyToEarliestPos map[string]int) bool {
	// Primary criterion: appearance order (earlier is better)
	newPos, hasNewPos := rt.getObjectOrderPosition(newKey)
	existingPos, hasExistingPos := refKeyToEarliestPos[referencedKey]

	if hasNewPos && hasExistingPos {
		if newPos < existingPos {
			return true // New key appears earlier
		} else if newPos > existingPos {
			return false // Existing key appears earlier
		}
	}

	// If no position info or positions are equal, prefer lexicographically earlier key for consistency
	return newKey < existingKey
}

func getObjectSchemaByKey(task readmeTask, key string) (objectSchema, bool) {
	if c, ok := task.FieldKeyToCanonical[key]; ok {
		key = c
	}
	// Prefer unified list if present
	for _, m := range task.AllObjects {
		if schema, ok := m[key]; ok {
			return schema, true
		}
	}
	for _, m := range task.InputObjects {
		if schema, ok := m[key]; ok {
			return schema, true
		}
	}
	for _, m := range task.OutputObjects {
		if schema, ok := m[key]; ok {
			return schema, true
		}
	}
	return objectSchema{}, false
}

func hasObjectSection(task readmeTask, key string) bool {
	// Resolve canonical
	if c, ok := task.FieldKeyToCanonical[key]; ok {
		key = c
	}
	// Prefer unified list
	for _, m := range task.AllObjects {
		if _, ok := m[key]; ok {
			return true
		}
	}
	// Fallback to input/output lists
	for _, m := range task.InputObjects {
		if _, ok := m[key]; ok {
			return true
		}
	}
	for _, m := range task.OutputObjects {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

// shouldRenderAtTopLevel determines if an object should be rendered at the top level
// rather than as a nested object, based on the TopLevelObjects classification
func shouldRenderAtTopLevel(task readmeTask, key string) bool {
	// Check if this key is directly marked as top-level
	if task.TopLevelObjects[key] {
		return true
	}

	// Also check canonical key
	if c, ok := task.FieldKeyToCanonical[key]; ok {
		if task.TopLevelObjects[c] {
			return true
		}
	}

	return false
}

// typeWithRefUsingKey formats Type like typeWithRef but uses the field key to prefer matching
// sections by title equal to the singularized, title-cased key. It also renders the link text
// as a kebab-case ref-like token (e.g., Generation Config -> generation-config).
func typeWithRefUsingKey(prop property, fieldKey string, task readmeTask) string {

	// 1) Direct $ref on the property (object or array item) - always use this first
	if strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)

		// Check if this is an enum/primitive type without properties
		if def, ok := defsByKey[refKey]; ok && len(def.Properties) == 0 {
			// For enums/primitives, return the base type (the description will be in Note column)
			baseType := prop.Type
			if baseType == "" {
				// For objects with no properties (like url-context, code-execution), just return "object"
				if prop.Type == "object" || (prop.Type == "" && prop.Properties == nil) {
					return "object"
				}
				baseType = "string" // most enums are strings
			}
			return baseType
		}
		refTitle := resolveRefTitle(refKey, task)
		return fmt.Sprintf("object ([%s](#%s-%s))", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
	}
	if strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Items.Ref)
		refTitle := resolveRefTitle(refKey, task)
		return fmt.Sprintf("array[object ([%s](#%s-%s))]", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
	}

	// 2) If the property already has a primitive type, don't try to find references
	if prop.Type == "string" || prop.Type == "number" || prop.Type == "integer" || prop.Type == "boolean" {
		return prop.Type
	}

	// 2.1) If this is an object with no properties, just return "object" without any reference link
	if prop.Type == "object" && prop.Properties == nil {
		return "object"
	}

	// 3) Prefer using original schema reference from RootInput/RootOutput when available
	lookupRootRef := func(root *objectSchema) string {
		if root == nil {
			return ""
		}
		// Recursive search through all nested properties in the original unmerged schema
		var findRecursive func(os *objectSchema) (property, bool)
		findRecursive = func(os *objectSchema) (property, bool) {
			if os == nil {
				return property{}, false
			}
			if os.Properties != nil {
				if p, ok := os.Properties[fieldKey]; ok {
					return p, true
				}
				// Search nested object properties recursively
				for _, prop := range os.Properties {
					if prop.Type == "object" && prop.Properties != nil {
						if p, ok := prop.Properties[fieldKey]; ok {
							return p, true
						}
						// Recurse deeper
						if p, found := findRecursive(&objectSchema{Properties: prop.Properties}); found {
							return p, true
						}
					}
					if prop.Items.Type == "object" && prop.Items.Properties != nil {
						if p, ok := prop.Items.Properties[fieldKey]; ok {
							return p, true
						}
						// Recurse deeper into array items
						if p, found := findRecursive(&objectSchema{Properties: prop.Items.Properties}); found {
							return p, true
						}
					}
				}
			}
			for _, s := range os.AllOf {
				if p, found := findRecursive(&s); found {
					return p, true
				}
			}
			return property{}, false
		}
		if p, ok := findRecursive(root); ok {
			if strings.HasPrefix(p.Ref, "#/$defs/") {
				return extractRefKey(p.Ref)
			}
			if strings.HasPrefix(p.Items.Ref, "#/$defs/") {
				return extractRefKey(p.Items.Ref)
			}
		}
		// Also search in all $defs for this field
		// Sort keys for deterministic iteration
		defKeys := make([]string, 0, len(defsByKey))
		for key := range defsByKey {
			defKeys = append(defKeys, key)
		}
		sort.Strings(defKeys)

		for _, key := range defKeys {
			def := defsByKey[key]
			if p, found := findRecursive(&def); found {
				if strings.HasPrefix(p.Ref, "#/$defs/") {
					return extractRefKey(p.Ref)
				}
				if strings.HasPrefix(p.Items.Ref, "#/$defs/") {
					return extractRefKey(p.Items.Ref)
				}
			}
		}
		return ""
	}
	if refKey := lookupRootRef(task.RootInput); refKey != "" {
		refTitle := resolveRefTitle(refKey, task)
		// Determine array vs object based on the current property (not root)
		if prop.Type == "array" || (prop.Items.Type == "object") {
			return fmt.Sprintf("array[object ([%s](#%s-%s))]", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
		}
		return fmt.Sprintf("object ([%s](#%s-%s))", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
	}
	if refKey := lookupRootRef(task.RootOutput); refKey != "" {
		refTitle := resolveRefTitle(refKey, task)
		if prop.Type == "array" || (prop.Items.Type == "object") {
			return fmt.Sprintf("array[object ([%s](#%s-%s))]", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
		}
		return fmt.Sprintf("object ([%s](#%s-%s))", refKey, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
	}

	// 2.5) Map by original $defs via structural signature when $ref is missing in the rendered object
	if prop.Items.Type == "object" {
		if prop.Items.Properties != nil {
			sig := buildObjectSignature(prop.Items.Properties)
			if rk, ok := sigToDefsKey[sig]; ok && rk != "" {
				refTitle := resolveRefTitle(rk, task)
				return fmt.Sprintf("array[object ([%s](#%s-%s))]", rk, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
			}
		}
	}
	if prop.Type == "object" && prop.Properties != nil {
		sig := buildObjectSignature(prop.Properties)
		if rk, ok := sigToDefsKey[sig]; ok && rk != "" {
			refTitle := resolveRefTitle(rk, task)
			return fmt.Sprintf("object ([%s](#%s-%s))", rk, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(refTitle))
		}
	}

	// 3) If an object section is explicitly registered for this field key, link to its section title
	// BUT only if this property is actually an object type
	if prop.Type == "object" || prop.Items.Type == "object" {
		if schema, ok := getObjectSchemaByKey(task, fieldKey); ok && schema.Title != "" {
			ref := strings.ToLower(fieldKey)
			// Determine array vs object by checking items on the current property
			if prop.Items.Type == "object" || prop.Type == "array" || prop.Type == "array[object]" {
				return fmt.Sprintf("array[object ([%s](#%s-%s))]", ref, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(schema.Title))
			}
			return fmt.Sprintf("object ([%s](#%s-%s))", ref, blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(schema.Title))
		}
	}

	// 4) Generic structural fallback: match to original $defs key by structure when available
	// Only for properties that actually have object structure
	if prop.Type == "object" && prop.Properties != nil {
		// Prefer anchoring to the field's own section id (#task-fieldKey)
		return fmt.Sprintf("object ([%s](#%s-%s))", strings.ToLower(fieldKey), blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(fieldKey))
	}
	if prop.Items.Type == "object" && prop.Items.Properties != nil {
		// Prefer anchoring to the field's own section id (#task-fieldKey)
		return fmt.Sprintf("array[object ([%s](#%s-%s))]", strings.ToLower(fieldKey), blackfriday.SanitizedAnchorName(task.Title), blackfriday.SanitizedAnchorName(fieldKey))
	}

	// Fallback to original formatter (no guessing)
	return typeWithRef(prop, task)
}

func singularize(s string) string {
	if len(s) > 1 && strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

// getSortedKeysByAppearance returns keys from a map[string]objectSchema ordered by their appearance in ObjectOrder
func (rt *readmeTask) getSortedKeysByAppearance(m map[string]objectSchema) []string {
	// First, collect all keys that exist in the map
	availableKeys := make(map[string]bool)
	for k := range m {
		availableKeys[k] = true
	}

	// Return keys in the order they appear in ObjectOrder, but only if they exist in the map
	result := make([]string, 0, len(m))
	for _, key := range rt.ObjectOrder {
		if availableKeys[key] {
			result = append(result, key)
			delete(availableKeys, key) // Mark as processed
		}
	}

	// Add any remaining keys that weren't in ObjectOrder (fallback to alphabetical)
	remaining := make([]string, 0, len(availableKeys))
	for key := range availableKeys {
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	result = append(result, remaining...)

	return result
}

// getSortedKeysByAppearanceWithTask is a wrapper function for template usage
func getSortedKeysByAppearanceWithTask(m map[string]objectSchema, task readmeTask) []string {
	return task.getSortedKeysByAppearance(m)
}

// extractRefKey extracts the key from a $ref string (e.g., "#/$defs/my-key" -> "my-key")
func extractRefKey(ref string) string {
	return strings.TrimPrefix(ref, "#/$defs/")
}

// trackObjectOrder adds a key to the ObjectOrder slice if it's not already present
// and classifies it as top-level if it appears early enough
func (rt *readmeTask) trackObjectOrder(key string) {
	// Check if key is already tracked
	for _, existing := range rt.ObjectOrder {
		if existing == key {
			return
		}
	}

	position := len(rt.ObjectOrder)
	rt.ObjectOrder = append(rt.ObjectOrder, key)

	// Classify as top-level if it appears within the first 60 positions
	// This ensures important objects get their own sections rather than being nested
	if position < 60 {
		rt.TopLevelObjects[key] = true
	}

}

// isCriticalObject determines if an object is critical and must be preserved regardless of deduplication
func (rt *readmeTask) isCriticalObject(key string) bool {
	// Preserve all objects that are classified as top-level based on appearance order
	// This ensures important objects that appear early get their own sections
	return rt.TopLevelObjects[key]
}

// getObjectOrderPosition returns the position of a key in ObjectOrder and whether it was found
func (rt *readmeTask) getObjectOrderPosition(key string) (int, bool) {
	for i, orderKey := range rt.ObjectOrder {
		if orderKey == key {
			return i, true
		}
	}
	return -1, false
}

// findObjectReferences finds all objects that this schema references
func (rt *readmeTask) findObjectReferences(schema objectSchema) []string {
	var refs []string
	refSet := make(map[string]bool)

	// Check properties for $ref
	if schema.Properties != nil {
		// Sort property keys for deterministic iteration
		propKeys := make([]string, 0, len(schema.Properties))
		for key := range schema.Properties {
			propKeys = append(propKeys, key)
		}
		sort.Strings(propKeys)

		for _, key := range propKeys {
			rt.collectPropertyReferences(schema.Properties[key], refSet)
		}
	}

	// Check oneOf for references
	if schema.OneOf != nil {
		for _, oneOfSchema := range schema.OneOf {
			subRefs := rt.findObjectReferences(oneOfSchema)
			for _, ref := range subRefs {
				refSet[ref] = true
			}
		}
	}

	// Convert set to slice in sorted order for determinism
	for ref := range refSet {
		refs = append(refs, ref)
	}
	sort.Strings(refs)

	return refs
}

// collectPropertyReferences recursively collects $ref references from a property
func (rt *readmeTask) collectPropertyReferences(prop property, refSet map[string]bool) {
	// Direct $ref
	if prop.Ref != "" && strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		refSet[refKey] = true
	}

	// Array items $ref
	if prop.Items.Ref != "" && strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Items.Ref)
		refSet[refKey] = true
	}

	// Nested properties
	if prop.Properties != nil {
		for _, nestedProp := range prop.Properties {
			rt.collectPropertyReferences(nestedProp, refSet)
		}
	}

	// OneOf references
	if prop.OneOf != nil {
		for _, oneOfSchema := range prop.OneOf {
			subRefs := rt.findObjectReferences(oneOfSchema)
			for _, ref := range subRefs {
				refSet[ref] = true
			}
		}
	}
}

// objectReferences checks if a schema references a specific object key
func (rt *readmeTask) objectReferences(schema objectSchema, targetKey string) bool {
	// Check properties
	if schema.Properties != nil {
		for _, prop := range schema.Properties {
			if rt.propertyReferences(prop, targetKey) {
				return true
			}
		}
	}

	// Check oneOf
	if schema.OneOf != nil {
		for _, oneOfSchema := range schema.OneOf {
			if rt.objectReferences(oneOfSchema, targetKey) {
				return true
			}
		}
	}

	return false
}

// propertyReferences checks if a property references a specific object key
func (rt *readmeTask) propertyReferences(prop property, targetKey string) bool {
	// Check direct $ref
	if prop.Ref != "" && strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		if refKey == targetKey {
			return true
		}
	}

	// Check array items $ref
	if prop.Items.Ref != "" && strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Items.Ref)
		if refKey == targetKey {
			return true
		}
	}

	// Check nested properties
	if prop.Properties != nil {
		for _, nestedProp := range prop.Properties {
			if rt.propertyReferences(nestedProp, targetKey) {
				return true
			}
		}
	}

	// Check oneOf
	if prop.OneOf != nil {
		for _, oneOfSchema := range prop.OneOf {
			if rt.objectReferences(oneOfSchema, targetKey) {
				return true
			}
		}
	}

	return false
}

// scanAllObjectsForRefs scans all final objects and establishes parent-child relationships for $ref objects
func (rt *readmeTask) scanAllObjectsForRefs() {
	// Scan input properties
	if rt.RootInput != nil && rt.RootInput.Properties != nil {
		rt.scanPropertiesForRefs(rt.RootInput.Properties, "input")
	}

	// Scan output properties
	if rt.RootOutput != nil && rt.RootOutput.Properties != nil {
		rt.scanPropertiesForRefs(rt.RootOutput.Properties, "output")
	}

	// Scan all object properties
	for _, objMap := range rt.AllObjects {
		for objKey, objSchema := range objMap {
			if objSchema.Properties != nil {
				rt.scanPropertiesForRefs(objSchema.Properties, objKey)
			}
		}
	}

	// Consolidate backlinks from merged objects to their canonical keys
	for originalKey, canonicalKey := range rt.FieldKeyToCanonical {
		if originalKey != canonicalKey {
			// Transfer backlinks from CanonicalToParents
			if rt.CanonicalToParents[originalKey] != nil {
				if rt.CanonicalToParents[canonicalKey] == nil {
					rt.CanonicalToParents[canonicalKey] = map[string]bool{}
				}
				for parent := range rt.CanonicalToParents[originalKey] {
					rt.CanonicalToParents[canonicalKey][parent] = true
				}
			}

			// Also transfer backlinks from CanonicalToParentMeta
			if rt.CanonicalToParentMeta[originalKey] != nil {
				if rt.CanonicalToParentMeta[canonicalKey] == nil {
					rt.CanonicalToParentMeta[canonicalKey] = map[string]parentMeta{}
				}
				for parent, meta := range rt.CanonicalToParentMeta[originalKey] {
					rt.CanonicalToParentMeta[canonicalKey][parent] = meta
				}
			}
		}
	}

	// Scan for union type objects and establish backlinks from their referencers
	rt.establishUnionTypeBacklinks()
}

// scanPropertiesForRefs recursively scans properties and establishes parent-child relationships for $ref objects
func (rt *readmeTask) scanPropertiesForRefs(properties map[string]property, containerKey string) {
	for fieldKey, prop := range properties {

		// Determine parent key - use container name for table-level backlinks
		parentKey := containerKey
		if containerKey == "" {
			// For root-level properties, determine if it's input or output
			if rt.InputKeySet != nil && rt.InputKeySet[fieldKey] {
				parentKey = "input"
			} else if rt.OutputKeySet != nil && rt.OutputKeySet[fieldKey] {
				parentKey = "output"
			}
		}

		// Handle direct $ref objects
		if prop.Ref != "" && strings.HasPrefix(prop.Ref, "#/$defs/") && parentKey != "" {
			refKey := extractRefKey(prop.Ref)
			rt.recordParentRelationship(refKey, parentKey)
		}

		// Handle array items with $ref
		if prop.Type == "array" && prop.Items.Ref != "" && strings.HasPrefix(prop.Items.Ref, "#/$defs/") && parentKey != "" {
			refKey := extractRefKey(prop.Items.Ref)
			rt.recordParentRelationship(refKey, parentKey)
		}

		// Note: Complex structural matching removed - respecting field keys genuinely

		// Recurse into nested object properties
		if prop.Properties != nil {
			rt.scanPropertiesForRefs(prop.Properties, fieldKey)
		}
		if prop.Type == "array" && prop.Items.Properties != nil {
			rt.scanPropertiesForRefs(prop.Items.Properties, fieldKey)
		}
	}
}

// establishUnionTypeBacklinks scans for union type objects and establishes backlinks from their referencers
func (rt *readmeTask) establishUnionTypeBacklinks() {
	// Find all union type objects (objects with oneOf in defsByKey)
	unionTypes := make(map[string]bool)
	// Sort keys for deterministic iteration
	defKeys := make([]string, 0, len(defsByKey))
	for key := range defsByKey {
		defKeys = append(defKeys, key)
	}
	sort.Strings(defKeys)

	for _, key := range defKeys {
		def := defsByKey[key]
		if len(def.OneOf) > 0 {
			unionTypes[key] = true
		}
	}

	// For each union type, find objects that reference it
	for unionKey := range unionTypes {
		// Scan all $defs to find ones that reference this union type
		// Sort keys for deterministic iteration
		defKeys := make([]string, 0, len(defsByKey))
		for key := range defsByKey {
			defKeys = append(defKeys, key)
		}
		sort.Strings(defKeys)

		for _, defKey := range defKeys {
			def := defsByKey[defKey]
			if defKey == unionKey {
				continue // Skip self-reference
			}
			if rt.objectReferencesTarget(def, unionKey) {
				rt.recordParentRelationship(unionKey, defKey)
			}
		}

		// Scan all objects to find ones that reference this union type
		for _, objMap := range rt.AllObjects {
			for objKey, objSchema := range objMap {
				// Check if this object references the union type
				if rt.objectReferencesTarget(objSchema, unionKey) {
					rt.recordParentRelationship(unionKey, objKey)
				}
			}
		}

		// Also scan input/output schemas
		if rt.RootInput != nil {
			rt.scanSchemaForUnionReferences(rt.RootInput, unionKey, "input")
		}
		if rt.RootOutput != nil {
			rt.scanSchemaForUnionReferences(rt.RootOutput, unionKey, "output")
		}
	}
}

// objectReferencesTarget checks if a schema references a specific target key
func (rt *readmeTask) objectReferencesTarget(schema objectSchema, targetKey string) bool {
	// Check properties for references to the target
	if schema.Properties != nil {
		for _, prop := range schema.Properties {
			if rt.propertyReferencesTarget(prop, targetKey) {
				return true
			}
		}
	}
	return false
}

// propertyReferencesTarget checks if a property references a specific target key
func (rt *readmeTask) propertyReferencesTarget(prop property, targetKey string) bool {
	// Check direct $ref
	if prop.Ref != "" && strings.HasPrefix(prop.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Ref)
		if refKey == targetKey {
			return true
		}
	}

	// Check array items $ref
	if prop.Items.Ref != "" && strings.HasPrefix(prop.Items.Ref, "#/$defs/") {
		refKey := extractRefKey(prop.Items.Ref)
		if refKey == targetKey {
			return true
		}
	}

	// Check nested properties
	if prop.Properties != nil {
		for _, nestedProp := range prop.Properties {
			if rt.propertyReferencesTarget(nestedProp, targetKey) {
				return true
			}
		}
	}

	return false
}

// scanSchemaForUnionReferences scans a schema for references to a union type
func (rt *readmeTask) scanSchemaForUnionReferences(schema *objectSchema, unionKey, containerKey string) {
	if schema == nil || schema.Properties == nil {
		return
	}

	for _, prop := range schema.Properties {
		if rt.propertyReferencesTarget(prop, unionKey) {
			rt.recordParentRelationship(unionKey, containerKey)
			return
		}
	}
}

func (rt *readmeTask) orderObjectsByRenderOccasion() {
	// Map keys to their object index
	keyToIndex := make(map[string]int)
	for i, objMap := range rt.AllObjects {
		for key := range objMap {
			keyToIndex[key] = i
		}
	}

	// DFS over objects using object-level references
	visitedObj := make(map[int]bool)
	orderIdx := make([]int, 0, len(rt.AllObjects))

	var visit func(objIdx int)
	visit = func(objIdx int) {
		if objIdx < 0 || objIdx >= len(rt.AllObjects) || visitedObj[objIdx] {
			return
		}
		visitedObj[objIdx] = true
		orderIdx = append(orderIdx, objIdx)

		// Get references from this object
		objMap := rt.AllObjects[objIdx]
		var allRefs []string

		// Sort keys for deterministic iteration
		keys := make([]string, 0, len(objMap))
		for key := range objMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			schema := objMap[key]
			// Try defs first, then use the schema directly
			if def, exists := defsByKey[key]; exists {
				allRefs = append(allRefs, rt.findObjectReferences(def)...)
			} else {
				allRefs = append(allRefs, rt.findObjectReferences(schema)...)
			}
		}

		// Sort references by their current object positions for stable ordering
		sort.SliceStable(allRefs, func(i, j int) bool {
			iIdx, iOk := keyToIndex[allRefs[i]]
			jIdx, jOk := keyToIndex[allRefs[j]]
			if iOk && jOk {
				return iIdx < jIdx
			}
			return allRefs[i] < allRefs[j] // Lexical fallback
		})

		// Visit referenced objects
		for _, refKey := range allRefs {
			if refObjIdx, exists := keyToIndex[refKey]; exists {
				visit(refObjIdx)
			}
		}
	}

	// Seed DFS with deterministic key order to ensure consistent traversal
	// Collect all object indices with their primary keys
	type objWithKey struct {
		index int
		key   string
	}
	objsWithKeys := make([]objWithKey, 0, len(rt.AllObjects))

	for i, objMap := range rt.AllObjects {
		// Get the first (lexicographically smallest) key as primary key for seeding
		var primaryKey string
		keys := make([]string, 0, len(objMap))
		for key := range objMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			primaryKey = keys[0]
		}
		objsWithKeys = append(objsWithKeys, objWithKey{index: i, key: primaryKey})
	}

	// Sort by primary key for deterministic seeding order
	sort.Slice(objsWithKeys, func(i, j int) bool {
		return objsWithKeys[i].key < objsWithKeys[j].key
	})

	// Seed DFS in deterministic key order
	for _, obj := range objsWithKeys {
		visit(obj.index)
	}

	// Rebuild AllObjects in the derived order
	newAll := make([]map[string]objectSchema, 0, len(rt.AllObjects))
	added := make(map[int]bool)
	for _, idx := range orderIdx {
		if !added[idx] && idx >= 0 && idx < len(rt.AllObjects) {
			newAll = append(newAll, rt.AllObjects[idx])
			added[idx] = true
		}
	}
	// Append any remaining objects not reached by DFS, preserving their relative order
	for i := range rt.AllObjects {
		if !added[i] {
			newAll = append(newAll, rt.AllObjects[i])
		}
	}

	rt.AllObjects = newAll
}
