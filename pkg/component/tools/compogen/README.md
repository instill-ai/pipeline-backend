# `compogen`

`compogen` is a generation tool for Instill AI component schemas. It uses the
information in a component schema to automatically generate the component
documentation.

## Installation

```shell
go install github.com/instill-ai/pipeline-backend/pkg/component/tools/compogen@latest
```

## Generate the documentation of a component

`compogen` can generate the README of a component by reading its schemas. The
format of the documentation is MDX, so the generated files can directly be used
in the Instill AI website.

```shell
compogen readme path/to/component/config path/to/component/README.mdx
```

### Validation & guidelines

In order to successfully build the README of a component, the `definition.yaml`
and `tasks.yaml` files must be present in the component configuration directory.

The `definition.yaml` file must contain an array with one object in which the
following fields must be present and comply with the following guidelines:

- `id`.
- `title`.
- `description` - It should contain a single sentence describing the component.
  The template will use it next to the component title (`{{ .Title }}{{
  .Description }}.`) so it must be written in imperative tense.
- `releaseStage` - Must be the string representation of one of the nonzero
  values of `ComponentDefinition.ReleaseStage`, defined in
  [protobufs](https://github.com/instill-ai/protobufs/blob/main/pipeline/pipeline/v1beta/component_definition.proto).
- `type` - Component definitions must contain this field and its value must
  match one of the (string) values, defined in [protobufs](https://github.com/instill-ai/protobufs/blob/main/pipeline/pipeline/v1beta/component_definition.proto).
- `availableTasks` - This array must have at least one value, which should be
  one of the root-level keys in the `tasks.yaml` file.
- `sourceUrl` - Must be a valid URL. It must not end with a slash, as the
  definitions path will be appended.

Certain optional fields modify the document behaviour:

- `public`, when `true`, will set the `draft` property to `false`.
- The content of `prerequisites` will be displayed in
  an info block next to the resource configuration details.
- A table will be built for the `setup` properties described in `setup.yaml`. They
  must contain an `instillUIOrder` field so the row order is deterministic.

### Injecting extra content

Some components might require or benefit from having extra sections in their
documentation. For instance, one might want to dedicate a section to add a guide
to configuring an account in a 3rd party vendor or to explain in details a
particular configuration of a component.

The `extraContents` flag in the `readme` subcommand lets `compogen` inject the
content of a document into the generated file. The content will be added
verbatim, so it should complain with the MDX syntax.

This flag takes a key and a value, where the key specifies under which section
(at the bottom) the content will be added. The value is the path to the content.
The following section IDs are accepted:

- `intro`
- `release`
- `config`
- `setup`
- Any task ID defined in `tasks.yaml` (e.g. `TASK_CHUNK_TEXT`)
- `bottom`

More than one section can be extended with this flag:

```shell
compogen readme path/to/component/config path/to/component/README.mdx \
  --extraContents setup=path/to/component/.compogen/detailed-setup.mdx
  --extraContents TASK_DO_SOMETHING=path/to/component/.compogen/detailed-task.mdx
```

## TODO

- Support `oneOf` schemas for resource properties, present in, e.g., the [REST API](https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/application/restapi/v0/config/definition.yaml#L26) component.
  - We might leverage some Go implementation of JSON schema. Some candidates:
    - [santhosh-tekuri/jsonschema](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v5#Schema)
    - [omissis/go-jsonschema](https://github.com/omissis/go-jsonschema/blob/934012d/pkg/schemas/model.go#L107)
    - [invopop/jsonschema](https://github.com/invopop/jsonschema/blob/a446707/schema.go#L14)
    - [swaggest/jsonschema-go](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Schema)
  - The schema loading carried out by the `component/base` package in
    `LoadDefinition` or `LoadDefinition` might also be
    useful, although it is oriented to transforming the data to a `structpb.Struct`
    rather than to define the object structure.
- In the "supported tasks" tables, provide better documentation for nested
  arrays and objects (currently the type doesn't support nesting).
- If task definitions contain examples for the (required) input and output
  fields, generate param samples as in the [OpenAI component documentation](https://github.com/instill-ai/instill-ai.dev/blob/main/docs/component/ai/openai.en.mdx).
- Automate command module version.

## Next steps

- `compogen validate` might be used validate any component configuration.
- `compogen new` might be used to generate the skeleton of a component.
- In the future we might want to generate documentation in different languages.
This will require some thought.
