# Setup

mkdir -p pkg/dummy/config
cp definition.yaml pkg/dummy/config/definition.yaml
cp tasks.yaml pkg/dummy/config/tasks.yaml

mkdir -p pkg/dummy/.compogen
cp extra-dummy.mdx pkg/dummy/.compogen/extra-dummy.mdx
cp extra-bottom.mdx pkg/dummy/.compogen/extra-bottom.mdx

# NOK - Wrong files

! compogen readme pkg/dummy/wrong pkg/dummy/README.mdx
cmp stderr want-no-defs

mkdir -p pkg/dummy/wrong
cp definition.yaml pkg/dummy/wrong/definition.yaml
! compogen readme pkg/dummy/wrong pkg/dummy/README.mdx
cmp stderr want-no-tasks

! compogen readme pkg/dummy/config pkg/wrong/README.mdx
cmp stderr want-wrong-target

# OK

compogen readme ./pkg/dummy/config ./pkg/dummy/README.mdx --extraContents TASK_DUMMY=./pkg/dummy/.compogen/extra-dummy.mdx --extraContents bottom=./pkg/dummy/.compogen/extra-bottom.mdx
cmp pkg/dummy/README.mdx want-readme.mdx

-- definition.yaml --
availableTasks:
  - TASK_DUMMY
  - TASK_DUMMIER_THAN_DUMMY
public: true
spec: {}
id: dummy
title: Dummy
type: COMPONENT_TYPE_OPERATOR
description: Perform an action.
releaseStage: RELEASE_STAGE_BETA
sourceUrl: https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/operator/dummy/v0

-- tasks.yaml --
TASK_DUMMY:
  description: Perform a dummy task.
  input:
    properties:
      durna:
        description: Lorem ipsum dolor sit amet, consectetur adipiscing elit
        uiOrder: 0
        title: Durna
        type: string
      parra:
        deprecated: true
        description: Shouldn't appear, it's deprecated
        uiOrder: 1
        title: Parra
        type: string
    required:
      - durna
    title: Input
  output:
    properties:
      orci:
        description: Orci sagittis eu volutpat odio facilisis mauris sit
        uiOrder: 0
        title: Orci
        type: string
    title: Output

TASK_DUMMIER_THAN_DUMMY:
  title: Dummier
  description: This task is dummier than `TASK_DUMMY`.
  input:
    properties:
      cursus:
        description: Cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo
        uiOrder: 0
        title: Cursus
        type: string
    required:
      - cursus
    title: Input
  output:
    properties:
      elementum:
        description: Tellus elementum sagittis vitae et
        uiOrder: 0
        title: Elementum
        type: string
      errors:
        description: Error messages, if any, during the dummy process
        uiOrder: 3
        title: Errors
        items:
          type: string
        type: array
      context:
        description: Free-form metadata
        uiOrder: 4
        required: []
        title: Meta
      atem:
        description: >-
          This object should comply witht he format {"tortor": "something", "arcu": "something else"}
        uiOrder: 1
        title: Atem
        type: object
        properties:
          tortor:
            description: Tincidunt tortor aliquam nulla
            uiOrder: 0
            title: Tincidunt tortor
            type: string
          arcu:
            description: Bibendum arcu vitae elementum curabitur vitae nunc sed velit
            uiOrder: 1
            title: Arcu
            type: string
        required: []
      nullam_non:
        description: Id faucibus nisl tincidunt eget nullam non
        uiOrder: 2
        title: Nullam non
        type: number
    required:
      - elementum
      - atem
      - nullam_non
      - error
    title: Output

-- extra-dummy.mdx --
#### How to use the dummy task

You might be tempted to think than dummier is better than dummy. However,
one might be wise when choosing between them.
-- extra-bottom.mdx --
## Final words

Thanks for reaching this point! No one really reads documentation thoroughly (:
-- want-no-defs --
Error: open pkg/dummy/wrong/definition.yaml: no such file or directory
-- want-no-tasks --
Error: open pkg/dummy/wrong/tasks.yaml: no such file or directory
-- want-wrong-target --
Error: open pkg/wrong/README.mdx: no such file or directory
-- want-invalid-def --
Error: invalid definitions file:
Definitions field has an invalid length
-- want-readme.mdx --
---
title: "Dummy"
lang: "en-US"
draft: false
description: "Learn about how to set up a Dummy component https://github.com/instill-ai/instill-core"
---

The Dummy component is an operator component that allows users to perform an action.
It can carry out the following tasks:
- [Dummy](#dummy)
- [Dummier](#dummier)

## Release Stage

`Beta`

## Configuration

The component definition and tasks are defined in the [definition.yaml](https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/operator/dummy/v0/config/definition.yaml) and [tasks.yaml](https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/operator/dummy/v0/config/tasks.yaml) files respectively.



## Supported Tasks

### Dummy

Perform a dummy task.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Input | Field ID | Type | Description |
| :--- | :--- | :--- | :--- |
| Task ID (required) | `task` | string | `TASK_DUMMY` |
| Durna (required) | `durna` | string | Lorem ipsum dolor sit amet, consectetur adipiscing elit |

</div>


<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Output | Field ID | Type | Description |
| :--- | :--- | :--- | :--- |
| Orci (optional) | `orci` | string | Orci sagittis eu volutpat odio facilisis mauris sit |

</div>
#### How to use the dummy task

You might be tempted to think than dummier is better than dummy. However,
one might be wise when choosing between them.

### Dummier

This task is dummier than `TASK_DUMMY`.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Input | Field ID | Type | Description |
| :--- | :--- | :--- | :--- |
| Task ID (required) | `task` | string | `TASK_DUMMIER_THAN_DUMMY` |
| Cursus (required) | `cursus` | string | Cursus mattis molestie a iaculis at erat pellentesque adipiscing commodo |

</div>


<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Output | Field ID | Type | Description |
| :--- | :--- | :--- | :--- |
| Elementum | `elementum` | string | Tellus elementum sagittis vitae et |
| [Atem](#dummier-atem) | `atem` | object | This object should comply witht he format `{"tortor": "something", "arcu": "something else"}` |
| Nullam Non | `nullam_non` | number | Id faucibus nisl tincidunt eget nullam non |
| Errors (optional) | `errors` | array[string] | Error messages, if any, during the dummy process |
| Meta (optional) | `context` | any | Free-form metadata |

</div>


<details>
<summary> Output Objects in Dummier</summary>

<h4 id="dummier-atem">Atem</h4>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Type | Note |
| :--- | :--- | :--- | :--- |
| Arcu | `arcu` | string | Bibendum arcu vitae elementum curabitur vitae nunc sed velit |
| Tincidunt tortor | `tortor` | string | Tincidunt tortor aliquam nulla |

</div>
</details>

## Final words

Thanks for reaching this point! No one really reads documentation thoroughly (:
