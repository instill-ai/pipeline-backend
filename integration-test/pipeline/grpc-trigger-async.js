import grpc from "k6/net/grpc";


import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["../proto/pipeline/pipeline/v1beta"], "pipeline_public_service.proto");

export function CheckTrigger(data) {
  group(
    "Pipelines API: Trigger an async pipeline",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: constant.dbIDPrefix + randomString(10),
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      check(
        client.invoke(
          "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          data.metadata
        ),
        {
          "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(client.invoke(
        "pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
          data: constant.simplePayload.data,
        },
        data.metadata
      ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );


      check(
        client.invoke(
          `pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );

  group(
    "Pipelines API: Trigger an async pipeline with YAML recipe",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          id: constant.dbIDPrefix + randomString(10),
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      check(
        client.invoke(
          "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline",
          {
            parent: `${constant.namespace}`,
            pipeline: reqBody,
          },
          data.metadata
        ),
        {
          "pipeline.pipeline.v1beta.PipelinePublicService/CreateUserPipeline Async GRPC pipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(client.invoke(
        "pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline",
        {
          name: `${constant.namespace}/pipelines/${reqBody.id}`,
          data: constant.simplePayload.data,
        },
        data.metadata
      ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
          [`pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline response has operation id`]:
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );


      check(
        client.invoke(
          `pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${reqBody.id}`,
          },
          data.metadata
        ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}
