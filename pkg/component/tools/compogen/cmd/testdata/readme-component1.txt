# Setup

mkdir -p pkg/dummy/config
cp definition.json pkg/dummy/config/definition.json
cp setup.json pkg/dummy/config/setup.json
cp tasks.json pkg/dummy/config/tasks.json

mkdir -p pkg/dummy/.compogen
cp extra-setup.mdx pkg/dummy/.compogen/extra-setup.mdx

# OK

compogen readme ./pkg/dummy/config ./pkg/dummy/README.mdx --extraContents setup=./pkg/dummy/.compogen/extra-setup.mdx
cmp pkg/dummy/README.mdx want-readme.mdx

-- definition.json --
{
  "availableTasks": [
    "TASK_DUMMY"
  ],
  "public": true,
  "id": "dummy",
  "title": "Dummy",
  "vendor": "Dummy Inc.",
  "description": "Perform an action.",
  "prerequisites": "An account at [dummy.io](https://dummy.io) is required.",
  "type": "COMPONENT_TYPE_DATA",
  "releaseStage": "RELEASE_STAGE_COMING_SOON",
  "sourceUrl": "https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/data/dummy/v0"
}

-- setup.json --
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "additionalProperties": true,
  "properties": {
    "organization": {
      "description": "Specify which organization is used for the requests",
      "uiOrder": 1,
      "title": "Organization ID",
      "format": "string"
    },
    "api-key": {
      "description": "Fill in your Dummy API key",
      "uiOrder": 0,
      "title": "API Key",
      "format": "string"
    },
    "authentication": {
      "description": "Authentication method to use for the Dummy",
      "uiOrder": 0,
      "oneOf": [
        {
          "properties": {
            "auth-type": {
              "const": "NO_AUTH",
              "description": "No Authentication",
              "uiOrder": 0,
              "order": 0,
              "title": "Auth Format",
              "format": "string"
            }
          },
          "required": [
            "auth-type"
          ],
          "title": "No Auth"
        },
        {
          "properties": {
            "auth-type": {
              "const": "AUTH_1",
              "description": "Auth 1",
              "uiOrder": 0,
              "order": 0,
              "title": "Auth Format",
              "format": "string"
            },
            "auth-way": {
              "description": "ways for Auth 1",
              "acceptFormats": [
                "string"
              ],
              "enum": [
                "header",
                "query"
              ],
              "uiOrder": 1,
              "order": 1,
              "title": "Auth Way",
              "format": "string"
            }
          },
          "required": [
            "auth-type",
            "auth-way"
          ],
          "title": "Auth 1"
        }
      ],
      "order": 1,
      "title": "Authentication",
      "format": "object"
    }
  },
  "required": [
    "api-key"
  ],
  "title": "OpenAI Connection",
  "format": "object"
}

-- tasks.json --
{
  "TASK_DUMMY": {
    "description": "Perform a dummy task.",
    "input": {
      "properties": {
        "durna": {
          "description": "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
          "uiOrder": 0,
          "title": "Durna",
          "format": "string"
        },
        "strategy": {
          "description": "Chunking strategy",
          "uiOrder": 1,
          "properties": {
            "setting": {
              "description": "Chunk Setting",
              "additionalProperties": true,
              "format": "object",
              "title": "Chunk Setting",
              "uiOrder": 0,
              "required": [
                "chunk-method"
              ],
              "oneOf": [
                {
                  "properties": {
                    "chunk-method": {
                      "const": "Token",
                      "format": "string",
                      "title": "Chunk Method",
                      "description": "Chunking based on tokenization.",
                      "uiOrder": 0
                    },
                    "model-name": {
                      "description": "The name of the model used for tokenization.",
                      "enum": [
                        "gpt-4",
                        "gpt-3.5-turbo"
                      ],
                      "uiOrder": 1,
                      "title": "Model",
                      "format": "string"
                    }
                  },
                  "title": "Token",
                  "required": ["chunk-method"],
                  "format": "object",
                  "description": "Language models have a token limit. You should not exceed the token limit. When you split your text into chunks it is therefore a good idea to count the number of tokens. There are many tokenizers. When you count tokens in your text you should use the same tokenizer as used in the language model."
                },
                {
                  "properties": {
                    "chunk-method": {
                      "const": "Markdown",
                      "format": "string",
                      "title": "Chunk Method",
                      "description": "Chunking based on recursive splitting with markdown format.",
                      "uiOrder": 0
                    },
                    "model-name": {
                      "description": "The name of the model used for tokenization.",
                      "enum": [
                        "gpt-4",
                        "gpt-3.5-turbo"
                      ],
                      "uiOrder": 1,
                      "title": "Model",
                      "format": "string"
                    }
                  },
                  "title": "Markdown",
                  "required": ["chunk-method"],
                  "format": "object",
                  "description": "This text splitter is specially designed for Markdown format."
                }
              ]
            }
          },
          "title": "Strategy",
          "required": [
            "setting"
          ],
          "format": "object"
        },
        "dummy-string": {
          "description": "{{dummy_string}}",
          "uiOrder": 0,
          "title": "Dummy String",
          "format": "string"
        },
        "dummy-file": {
          "description": "this is a file",
          "uiOrder": 0,
          "title": "Dummy File",
          "format": "*"
        },
        "dummy-array": {
          "description": "this is an array",
          "uiOrder": 0,
          "title": "Dummy Array",
          "format": "array",
          "items": {
            "format": "*"
          }
        }
      },
      "required": [
        "durna"
      ],
      "title": "Input"
    },
    "output": {
      "properties": {
        "orci": {
          "description": "Orci sagittis eu volutpat odio facilisis mauris sit",
          "format": "string",
          "uiOrder": 0,
          "title": "Orci",
          "format": "string"
        },
        "conversations": {
          "description": "An array of conversations with thread messages",
          "uiOrder": 0,
          "title": "Conversations",
          "format": "array",
          "items": {
            "title": "conversation details",
            "format": "object",
            "properties": {
              "message": {
                "description": "message to start a conversation",
                "uiOrder": 0,
                "title": "Start Conversation Message",
                "format": "string"
              },
              "start-date": {
                "description": "when a conversation starts",
                "uiOrder": 1,
                "title": "Start Date",
                "format": "string"
              },
              "last-date": {
                "description": "Date of the last message",
                "uiOrder": 2,
                "title": "Last Date",
                "format": "string"
              },
              "thread-reply-messages": {
                "description": "replies in a conversation",
                "uiOrder": 0,
                "title": "Replied messages",
                "format": "array",
                "items": {
                  "title": "relied details",
                  "format": "object",
                  "properties": {
                    "message": {
                      "description": "message to reply a conversation",
                      "instillFormat": "string",
                      "uiOrder": 3,
                      "title": "Replied Message",
                      "format": "string"
                    }
                  },
                  "required": [
                    "message"
                  ]
                }
              }
            },
            "required": [
              "message",
              "start-date"
            ]
          }
        }
      },
      "title": "Output"
    }
  }
}
-- extra-setup.mdx --
This is some crucial information about setup: do it before execution.
-- want-readme.mdx --
---
title: "Dummy"
lang: "en-US"
draft: false
description: "Learn about how to set up a VDP Dummy component https://github.com/instill-ai/instill-core"
---

The Dummy component is a data component that allows users to perform an action.
It can carry out the following tasks:
- [Dummy](#dummy)



## Release Stage

`Coming Soon`



## Configuration

The component definition and tasks are defined in the [definition.json](https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/data/dummy/v0/config/definition.json) and [tasks.json](https://github.com/instill-ai/pipeline-backend/pkg/component/blob/main/data/dummy/v0/config/tasks.json) files respectively.




## Setup

<InfoBlock type="info" title="Prerequisites">An account at [dummy.io](https://dummy.io) is required.</InfoBlock>

In order to communicate with Dummy Inc., the following connection details need to be
provided. You may specify them directly in a pipeline recipe as key-value pairs
within the component's `setup` block, or you can create a **Connection** from
the [**Integration Settings**](https://www.instill.tech/docs/vdp/integration)
page and reference the whole `setup` as `setup:
${connection.<my-connection-id>}`.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| API Key (required) | `api-key` | string | Fill in your Dummy API key  |
| [Authentication](#authentication) | `authentication` | object | Authentication method to use for the Dummy  |
| Organization ID | `organization` | string | Specify which organization is used for the requests  |

</div>

This is some crucial information about setup: do it before execution.

<details>
<summary>The <code>authentication</code> Object </summary>

<h4 id="setup-authentication">Authentication</h4>

`authentication` must fulfill one of the following schemas:

<h5 id="setup-no-auth"><code>No Auth</code></h5>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Auth Format | `auth-type` | string |  Must be `"NO_AUTH"`   |
</div>

<h5 id="setup-auth-1"><code>Auth 1</code></h5>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Auth Format | `auth-type` | string |  Must be `"AUTH_1"`   |
| Auth Way | `auth-way` | string |  ways for Auth 1  <br/><details><summary><strong>Enum values</strong></summary><ul><li>`header`</li><li>`query`</li></ul></details>  |
</div>
</details>


## Supported Tasks

### Dummy

Perform a dummy task.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Input | Field ID | Format | Description |
| :--- | :--- | :--- | :--- |
| Task ID (required) | `task` | string | `TASK_DUMMY` |
| Dummy Array | `dummy-array` | array[any] | this is an array |
| Dummy File | `dummy-file` | any | this is a file |
| Dummy String | `dummy-string` | string | `{{dummy_string}}` |
| Durna (required) | `durna` | string | Lorem ipsum dolor sit amet, consectetur adipiscing elit |
| [Strategy](#dummy-strategy) | `strategy` | object | Chunking strategy |
</div>


<details>
<summary> Input Objects in Dummy</summary>

<h4 id="dummy-strategy">Strategy</h4>

Chunking strategy

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| [Chunk Setting](#dummy-chunk-setting) | `setting` | object | Chunk Setting  |
</div>
</details>

<details>
<summary>The <code>setting</code> Object </summary>

<h4 id="dummy-setting">Setting</h4>

`setting` must fulfill one of the following schemas:

<h5 id="dummy-token"><code>Token</code></h5>

Language models have a token limit. You should not exceed the token limit. When you split your text into chunks it is therefore a good idea to count the number of tokens. There are many tokenizers. When you count tokens in your text you should use the same tokenizer as used in the language model.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Chunk Method | `chunk-method` | string |  Must be `"Token"`   |
| Model | `model-name` | string |  The name of the model used for tokenization.  <br/><details><summary><strong>Enum values</strong></summary><ul><li>`gpt-4`</li><li>`gpt-3.5-turbo`</li></ul></details>  |
</div>

<h5 id="dummy-markdown"><code>Markdown</code></h5>

This text splitter is specially designed for Markdown format.

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Chunk Method | `chunk-method` | string |  Must be `"Markdown"`   |
| Model | `model-name` | string |  The name of the model used for tokenization.  <br/><details><summary><strong>Enum values</strong></summary><ul><li>`gpt-4`</li><li>`gpt-3.5-turbo`</li></ul></details>  |
</div>
</details>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Output | Field ID | Format | Description |
| :--- | :--- | :--- | :--- |
| [Conversations](#dummy-conversations) (optional) | `conversations` | array[object] | An array of conversations with thread messages |
| Orci (optional) | `orci` | string | Orci sagittis eu volutpat odio facilisis mauris sit |
</div>

<details>
<summary> Output Objects in Dummy</summary>

<h4 id="dummy-conversations">Conversations</h4>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Last Date | `last-date` | string | Date of the last message |
| Start Conversation Message | `message` | string | message to start a conversation |
| Start Date | `start-date` | string | when a conversation starts |
| [Replied messages](#dummy-replied-messages) | `thread-reply-messages` | array | replies in a conversation |
</div>

<h4 id="dummy-replied-messages">Replied Messages</h4>

<div class="markdown-col-no-wrap" data-col-1 data-col-2>

| Field | Field ID | Format | Note |
| :--- | :--- | :--- | :--- |
| Replied Message | `message` | string | message to reply a conversation |
</div>
</details>



