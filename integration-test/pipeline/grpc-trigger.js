import grpc from "k6/net/grpc";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["../proto/vdp/pipeline/v1beta"], "pipeline_public_service.proto");

export function CheckTrigger(data) {
  group(
    "Pipelines API: Trigger a pipeline",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqGRPC,
          },
          data.metadata
        ),
        {
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipeline",
          {
            name: `${constant.namespace}/pipelines/${reqGRPC.id}`,
            data: constant.simplePayload.data,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqGRPC.id}`,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline ${reqGRPC.id} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );



      client.close();
    }
  );

  group(
    "Pipelines API: Trigger a pipeline with YAML recipe",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqGRPC,
          },
          data.metadata
        ),
        {
          "vdp.pipeline.v1beta.PipelinePublicService/CreateUserPipeline GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      check(
        client.invoke(
          "vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipeline",
          {
            name: `${constant.namespace}/pipelines/${reqGRPC.id}`,
            data: constant.simplePayload.data,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(
        client.invoke(
          `vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqGRPC.id}`,
          },
          data.metadata
        ),
        {
          [`vdp.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline ${reqGRPC.id} response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );



      client.close();
    }
  );
}
