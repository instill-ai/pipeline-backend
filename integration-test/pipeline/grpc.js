import grpc from "k6/net/grpc";
import http from "k6/http";

import { check, group } from "k6";

import * as pipeline from "./grpc-pipeline-public.js";
import * as pipelineWithJwt from "./grpc-pipeline-public-with-jwt.js";
import * as pipelinePrivate from "./grpc-pipeline-private.js";
import * as trigger from "./grpc-trigger.js";
import * as triggerAsync from "./grpc-trigger-async.js";

const client = new grpc.Client();
const mgmtClient = new grpc.Client();

client.load(["../proto/pipeline/pipeline/v1beta"], "pipeline_public_service.proto");
client.load(["../proto/core/mgmt/v1beta"], "mgmt_public_service.proto");

import * as constant from "./const.js";

export let options = {
  setupTimeout: "300s",
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  client.connect(constant.pipelineGRPCPublicHost, {
    plaintext: true,
    timeout: "300s",
  });
  mgmtClient.connect(constant.mgmtGRPCPublicHost, {
    plaintext: true,
    timeout: "300s",
  });

  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1beta/auth/login`, JSON.stringify({
    "username": constant.defaultUsername,
    "password": constant.defaultPassword,
  }))

  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1beta/auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var metadata = {
    "metadata": {
      "Authorization": `Bearer ${loginResp.json().accessToken}`
    },
    "timeout": "600s",
  }

  var resp = client.invoke(
    "core.mgmt.v1beta.MgmtPublicService/GetAuthenticatedUser",
    {},
    metadata
  );
  client.close();
  mgmtClient.close();
  return {metadata: metadata, expectedOwner: resp.message.user};
}

export default function (data) {
  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });
      check(
        client.invoke(
          "pipeline.pipeline.v1beta.PipelinePublicService/Liveness",
          {}
        ),
        {
          "GET /health/pipeline response status is StatusOK": (r) =>
            r.status === grpc.StatusOK,
        }
      );
      client.close();
    });
  }

  if (!constant.apiGatewayMode) {
    pipelinePrivate.CheckList(data);
    pipelinePrivate.CheckLookUp(data);
    return;
  }

  pipelineWithJwt.CheckCreate(data);
  pipelineWithJwt.CheckList(data);
  pipelineWithJwt.CheckGet(data);
  pipelineWithJwt.CheckUpdate(data);
  pipelineWithJwt.CheckRename(data);
  pipelineWithJwt.CheckLookUp(data);
  pipeline.CheckCreate(data);
  pipeline.CheckList(data);
  pipeline.CheckGet(data);
  pipeline.CheckUpdate(data);
  pipeline.CheckRename(data);
  pipeline.CheckLookUp(data);

  trigger.CheckTrigger(data);
  triggerAsync.CheckTrigger(data);
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    for (const pipeline of client.invoke(
      "pipeline.pipeline.v1beta.PipelinePublicService/ListUserPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 1000,
      },
      data.metadata
    ).message.pipelines) {
      check(
        client.invoke(
          `pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipeline.id}`,
          },
          data.metadata
        ),
        {
          [`pipeline.pipeline.v1beta.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    client.close();
  });

  client.connect(constant.pipelineGRPCPublicHost, {
    plaintext: true,
  });

  client.close();
}
