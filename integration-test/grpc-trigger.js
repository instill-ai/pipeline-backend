import grpc from "k6/net/grpc";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");

export function CheckTriggerSingleImageSingleModel() {
  group(
    "Pipelines API: Trigger a pipeline for single image and single model",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSyncGRPCSimpleRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqGRPC,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqGRPC.id}`,
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
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]:
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
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqGRPC.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline ${reqGRPC.id} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      var reqHTTP = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSyncHTTPSimpleRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqHTTP,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (HTTP pipeline) response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqGRPC.id}`,
        }
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqHTTP.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (HTTP pipeline triggered by gRPC) response StatusFailedPrecondition":
            (r) => r.status === grpc.StatusFailedPrecondition,
        }
      );

      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqHTTP.id}`,
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

export function CheckTriggerMultiImageSingleModel() {
  group(
    "Pipelines API: Trigger a pipeline for multiple images and single model",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSyncGRPCSimpleRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqGRPC,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (GRPC pipeline) response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqGRPC.id}`,
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
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]:
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
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqGRPC.id}`,
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

export function CheckTriggerMultiImageMultiModel() {
  group(
    "Pipelines API: Trigger a pipeline for multiple images and multiple models",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.detSynGRPCMultiModelRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline",
          {
            pipeline: reqGRPC,
          }
        ),
        {
          "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (GRPC pipeline) response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/ActivatePipeline",
        {
          name: `pipelines/${reqGRPC.id}`,
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
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageURL["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response modelOutputs.length`]:
            (r) => r.message.outputs.length === 2,
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
        ],
      };

      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline",
          {
            name: `pipelines/${reqGRPC.id}`,
            inputs: payloadImageBase64["inputs"],
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response modelOutputs.length`]:
            (r) => r.message.outputs.length === 2,
        }
      );

      // Delete the pipeline
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${reqGRPC.id}`,
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
