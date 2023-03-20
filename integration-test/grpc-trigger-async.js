import http from "k6/http";
import grpc from 'k6/net/grpc';
import encoding from "k6/encoding";

import {
  check,
  group
} from "k6";
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  pipelinePublicHost
} from "./const.js";

import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

export function CheckTriggerAsyncSingleImageSingleModelInst() {

  group("Pipelines API: Trigger an async pipeline for single image and single model instance", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detAsyncSingleModelInstRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageBase64.task_inputs.length,
    });

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}

export function CheckTriggerAsyncMultiImageSingleModelInst() {


  group("Pipelines API: Trigger an async pipeline for multiple images and single model instance", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detAsyncSingleModelInstRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK": (r) => r.status === grpc.StatusOK,
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
      }]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      },
      {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }, {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }
      ]
    };

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageBase64.task_inputs.length,
    });

    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}

export function CheckTriggerAsyncMultiImageMultiModelInst() {

  group("Pipelines API: Trigger an async pipeline for multiple images and multiple model instances", () => {

    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    var reqBody = Object.assign({
      id: randomString(10),
      description: randomString(50),
    },
      constant.detAsyncMultiModelInstRecipe
    );

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline', {
      pipeline: reqBody
    }), {
      "vdp.pipeline.v1alpha.PipelinePublicService/CreatePipeline Async GRPC pipeline response StatusOK": (r) => r.status === grpc.StatusOK,
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

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageURL["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (url) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
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

    check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline', {
      name: `pipelines/${reqBody.id}`,
      task_inputs: payloadImageBase64["task_inputs"]
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerPipeline (base64) response dataMappingIndices.length`]: (r) => r.message.dataMappingIndices.length === payloadImageBase64.task_inputs.length,
    });

    // Delete the pipeline
    check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
      name: `pipelines/${reqBody.id}`
    }), {
      [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    client.close();
  });
}
