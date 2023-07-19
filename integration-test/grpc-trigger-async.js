import grpc from "k6/net/grpc";


import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");

export function CheckTriggerAsyncSingleImageSingleModel() {
  group(
    "Pipelines API: Trigger an async pipeline for single image and single model",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detAsyncSingleModelRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqBody,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqBody.id}`,
        }
      );

      var payloadImageURL = {
        inputs: [
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
        ],
      };

      check(client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
        {
          name: `pipelines/${reqBody.id}`,
          inputs: payloadImageURL["inputs"],
        }
      ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      var payloadImageBase64 = {
        inputs: [
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckTriggerAsyncMultiImageSingleModel() {
  group(
    "Pipelines API: Trigger an async pipeline for multiple images and single model",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detAsyncSingleModelRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqBody,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqBody.id}`,
        }
      );

      var payloadImageURL = {
        inputs: [
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      var payloadImageBase64 = {
        inputs: [
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckTriggerAsyncMultiImageMultiModel() {
  group(
    "Pipelines API: Trigger an async pipeline for multiple images and multiple models",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detAsyncMultiModelRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqBody,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqBody.id}`,
        }
      );

      var payloadImageURL = {
        inputs: [
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      var payloadImageBase64 = {
        inputs: [
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}

export function CheckTriggerAsyncMultiImageMultiModelMultipleDestination() {
  group(
    "Pipelines API: Trigger an async pipeline for multiple images and multiple models and multiple destinations",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detAsyncMultiModelMultipleDestinationRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqBody,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqBody.id}`,
        }
      );

      var payloadImageURL = {
        inputs: [
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      var payloadImageBase64 = {
        inputs: [
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
          {
            images: [
              {
                blob: constant.dogImg,
              },
            ],
          },
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
          {
            name: `pipelines/${reqBody.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (base64) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}


export function CheckTriggerAsyncSingleResponse() {
  group(
    "Pipelines API: Trigger an async pipeline and get the result from GetOperation",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detAsyncSingleResponseRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqBody,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqBody.id}`,
        }
      );

      var payloadImageURL = {
        inputs: [
          {
            images: [
              {
                url: "https://artifacts.instill.tech/imgs/dog.jpg",
              },
            ],
          },
        ],
      };

      var resp = client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
        {
          name: `pipelines/${reqBody.id}`,
          inputs: payloadImageURL["inputs"],
        }
      )
      check(resp,
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );

      for (var i = 0; i < 30; ++i) {
        var resp = client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/GetOperation",
          {
            name: resp.message.operation.name,
          }
        )
        if (resp.message.operation.done) {
          break
        }
        sleep(1)
      }

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/GetOperation",
          {
            name: resp.message.operation.name,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/GetOperation response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/GetOperation response done = true`]:
            (r) => r.message.operation.done === true,
          [`vdp.pipeline.v1alpha.PipelinePublicService/GetOperation response outputs.length = ${payloadImageURL["inputs"].length}`]:
            (r) => r.message.operation.response.outputs.length === payloadImageURL["inputs"].length,
          [`vdp.pipeline.v1alpha.PipelinePublicService/GetOperation response outputs[0].images.length = ${payloadImageURL["inputs"][0].images.length}`]:
            (r) => r.message.operation.response.outputs[0].images.length === payloadImageURL["inputs"][0].images.length,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqBody.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}
