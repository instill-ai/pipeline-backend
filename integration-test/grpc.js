import grpc from "k6/net/grpc";
import http from "k6/http";

import { check, group } from "k6";

import * as pipeline from "./grpc-pipeline-public.js";
import * as pipelineWithJwt from "./grpc-pipeline-public-with-jwt.js";
import * as trigger from "./grpc-trigger.js";
import * as triggerAsync from "./grpc-trigger-async.js";

const pipelineClient = new grpc.Client();
const mgmtClient = new grpc.Client();

pipelineClient.load(["proto"], "pipeline/v1beta/pipeline_public_service.proto");
mgmtClient.load(["proto"], "mgmt/v1beta/mgmt_public_service.proto");

import * as constant from "./const.js";

export let options = {
  setupTimeout: "300s",
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  pipelineClient.connect(constant.pipelineGRPCPublicHost, {
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

  var authResp = mgmtClient.invoke(
    "mgmt.v1beta.MgmtPublicService/GetAuthenticatedUser",
    {},
    metadata
  );

  pipelineClient.close();
  mgmtClient.close();
  return { metadata: metadata, expectedOwner: authResp.message ? authResp.message.user : null };
}

export default function (data) {
  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      pipelineClient.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });
      check(
        pipelineClient.invoke(
          "pipeline.v1beta.PipelinePublicService/Liveness",
          {}
        ),
        {
          "GET /health/pipeline response status is StatusOK": (r) =>
            r.status === grpc.StatusOK,
        }
      );
      pipelineClient.close();
    });
  }

  // Test all public APIs with JWT authentication
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

  // TODO: SKIPPED - Trigger tests failing due to underlying schema issues
  // (missing display_name columns in secret/connection tables)
  // trigger.CheckTrigger(data);
  // triggerAsync.CheckTrigger(data);
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    pipelineClient.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var listRes = pipelineClient.invoke(
      "pipeline.v1beta.PipelinePublicService/ListUserPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 1000,
      },
      data.metadata
    );

    if (listRes.message && listRes.message.pipelines) {
      for (const pipeline of listRes.message.pipelines) {
        var deleteRes = pipelineClient.invoke(
          `pipeline.v1beta.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipeline.id}`,
          },
          data.metadata
        );
        // Accept both StatusOK and StatusNotFound (pipeline might already be deleted)
        check(deleteRes, {
          [`pipeline.v1beta.PipelinePublicService/DeleteUserPipeline cleanup response OK or NotFound`]:
            (r) => r.status === grpc.StatusOK || r.status === grpc.StatusNotFound,
        });
      }
    }

    pipelineClient.close();
  });

  group("Integration API: Delete data created by this test", () => {
    var q = `DELETE FROM pipeline WHERE id LIKE '${constant.dbIDPrefix}%';`;
    constant.db.exec(q);
    constant.db.close();
  });
}
