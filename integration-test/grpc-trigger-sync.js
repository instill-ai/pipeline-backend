import grpc from 'k6/net/grpc';
import encoding from "k6/encoding";

import {
  check,
  group
} from "k6";
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

export function CheckTriggerSyncSingleImageSingleModel() {

  group("Pipelines API: Trigger a pipeline for single image and single model", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqGRPC = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detSyncGRPCSingleModelRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqGRPC
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline GRPC pipeline response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs.length`]: (r) => r.message.modelOutputs[0].taskOutputs.length === payloadImageURL.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageURL.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].task`]: (r) => r.message.modelOutputs[0].task === "TASK_DETECTION",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].model`]: (r) => r.message.modelOutputs[0].model === constant.model_name,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects.length`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects.length === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].index == dataMappingIndices[0]`]: (r) => r.message.modelOutputs[0].taskOutputs[0].index === r.message.dataMappingIndices[0],
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].category`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].category === "test",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].score`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].score === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox !== undefined,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs.length`]: (r) => r.message.modelOutputs[0].taskOutputs.length === payloadImageBase64.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageBase64.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].task`]: (r) => r.message.modelOutputs[0].task === "TASK_DETECTION",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].model`]: (r) => r.message.modelOutputs[0].model === constant.model_name,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects.length`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects.length === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].index == dataMappingIndices[0]`]: (r) => r.message.modelOutputs[0].taskOutputs[0].index === r.message.dataMappingIndices[0],
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].category`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].category === "test",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].score`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].score === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox !== undefined,
    });

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqGRPC.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline ${reqGRPC.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    var reqHTTP = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detSyncHTTPSingleModelRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqHTTP
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (HTTP pipeline) response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqHTTP.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (HTTP pipeline triggered by gRPC) response StatusFailedPrecondition": (r) => r.status === grpc.StatusFailedPrecondition,
    })

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqHTTP.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });

}

export function CheckTriggerSyncMultiImageSingleModel() {

  group("Pipelines API: Trigger a pipeline for multiple images and single model", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqGRPC = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detSyncGRPCSingleModelRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqGRPC
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (GRPC pipeline) response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      },
      {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      },
      {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }
      ]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs.length`]: (r) => r.message.modelOutputs[0].taskOutputs.length === payloadImageURL.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageURL.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].task`]: (r) => r.message.modelOutputs[0].task === "TASK_DETECTION",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].model`]: (r) => r.message.modelOutputs[0].model === constant.model_name,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects.length`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects.length === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].index == dataMappingIndices[0]`]: (r) => r.message.modelOutputs[0].taskOutputs[0].index === r.message.dataMappingIndices[0],
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].category`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].category === "test",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].score`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].score === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox !== undefined,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      },
      {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      }
      ]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs.length`]: (r) => r.message.modelOutputs[0].taskOutputs.length === payloadImageBase64.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageBase64.task_inputs.length,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].task`]: (r) => r.message.modelOutputs[0].task === "TASK_DETECTION",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].model`]: (r) => r.message.modelOutputs[0].model === constant.model_name,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects.length`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects.length === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].index == dataMappingIndices[0]`]: (r) => r.message.modelOutputs[0].taskOutputs[0].index === r.message.dataMappingIndices[0],
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].category`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].category === "test",
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].score`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].score === 1,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox`]: (r) => r.message.modelOutputs[0].taskOutputs[0].detection.objects[0].boundingBox !== undefined,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqGRPC.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}

export function CheckTriggerSyncMultiImageMultiModel() {

  group("Pipelines API: Trigger a pipeline for multiple images and multiple models", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqGRPC = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detSynGRPCMultiModelRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqGRPC
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline (GRPC pipeline) response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (url) response modelOutputs.length`]: (r) => r.message.modelOutputs.length === 2,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      }, {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline', {
      name: `pipelines/${reqGRPC.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerSyncPipeline (base64) response modelOutputs.length`]: (r) => r.message.modelOutputs.length === 2,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqGRPC.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}
