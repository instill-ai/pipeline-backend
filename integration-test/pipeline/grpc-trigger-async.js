import grpc from "k6/net/grpc";


import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["../proto/vdp/pipeline/v1beta"], "pipeline_public_service.proto");

export function CheckTrigger(metadata) {
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
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          metadata
        ),
        {
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(client.invoke(
        "vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
          inputs: constant.simplePayload.inputs,
        },
        metadata
      ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline (url) response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline (url) response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );


      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}
