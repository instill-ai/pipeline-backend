import grpc from "k6/net/grpc";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

const client = new grpc.Client();
client.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");

export function CheckTrigger(data) {
  group(
    "Pipelines API: Trigger a pipeline",
    () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });

      var reqGRPC = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqGRPC,
        },
        data.metadata
      );

      check(createRes, {
        "CreateNamespacePipeline response StatusOK":
          (r) => r.status === grpc.StatusOK,
      });

      if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
        console.log("SKIPPED: CheckTrigger - pipeline creation failed");
        client.close();
        return;
      }
      var pipelineId = createRes.message.pipeline.id;

      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/TriggerNamespacePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
            data: constant.simplePayload.data,
          },
          data.metadata
        ),
        {
          "TriggerNamespacePipeline response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          "DeleteNamespacePipeline response StatusOK":
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
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      var createRes = client.invoke(
        "pipeline.v1beta.PipelinePublicService/CreateNamespacePipeline",
        {
          parent: `${constant.namespace}`,
          pipeline: reqGRPC,
        },
        data.metadata
      );

      check(createRes, {
        "CreateNamespacePipeline YAML response StatusOK":
          (r) => r.status === grpc.StatusOK,
      });

      if (createRes.status !== grpc.StatusOK || !createRes.message || !createRes.message.pipeline) {
        console.log("SKIPPED: CheckTrigger YAML - pipeline creation failed");
        client.close();
        return;
      }
      var pipelineId = createRes.message.pipeline.id;

      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/TriggerNamespacePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
            data: constant.simplePayload.data,
          },
          data.metadata
        ),
        {
          "TriggerNamespacePipeline YAML response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );


      check(
        client.invoke(
          "pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline",
          {
            name: `${constant.namespace}/pipelines/${pipelineId}`,
          },
          data.metadata
        ),
        {
          "DeleteNamespacePipeline YAML response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );

      client.close();
    }
  );
}
