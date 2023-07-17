import grpc from "k6/net/grpc";


import { check, group } from "k6";
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
