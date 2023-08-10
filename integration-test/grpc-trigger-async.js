import grpc from "k6/net/grpc";


import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");

export function CheckTrigger() {
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
        constant.simpleRecipe
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

      check(client.invoke(
        "vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline",
        {
          name: `pipelines/${reqBody.id}`,
          inputs: constant.simplePayload.inputs,
        }
      ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1alpha.PipelinePublicService/TriggerAsyncPipeline (url) response has operation id`]:
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
