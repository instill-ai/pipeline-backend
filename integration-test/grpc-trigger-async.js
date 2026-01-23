import grpc from "k6/net/grpc";


import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");

export function CheckTrigger(data) {
  group(
    "Pipelines API: Trigger an async pipeline",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqBody = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreatePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(createRes, {
        "CreatePipeline Async response StatusOK":
          (r) => r.status === grpc.StatusOK,
      });

      if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
        console.log("SKIPPED: CheckTriggerAsync - pipeline creation failed");
        client.close();
        return;
      }
      var pipelineId = createRes.message.pipeline.id;


      check(client.invoke(
        "pipeline.v1beta.PipelinePublicService/TriggerAsyncPipeline",
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
          data: constant.simplePayload.data,
        },
        data.metadata
      ),
        {
          "TriggerAsyncPipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
          "TriggerAsyncPipeline response has operation id":
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );


      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/DeletePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          "DeletePipeline Async response StatusOK":
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
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreatePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqBody,
        },
        data.metadata
      );

      check(createRes, {
        "CreatePipeline Async YAML response StatusOK":
          (r) => r.status === grpc.StatusOK,
      });

      if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
        console.log("SKIPPED: CheckTriggerAsync YAML - pipeline creation failed");
        client.close();
        return;
      }
      var pipelineId = createRes.message.pipeline.id;


      check(client.invoke(
        "pipeline.v1beta.PipelinePublicService/TriggerAsyncPipeline",
        {
          name: `${constant.namespace}/pipelines/${pipelineId}`,
          data: constant.simplePayload.data,
        },
        data.metadata
      ),
        {
          "TriggerAsyncPipeline YAML response StatusOK":
            (r) => r.status === grpc.StatusOK,
          "TriggerAsyncPipeline YAML response has operation id":
            (r) => r.message.operation.name.startsWith("operations/"),
        }
      );


      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/DeletePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          "DeletePipeline Async YAML response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}
