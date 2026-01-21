import grpc from "k6/net/grpc";
import http from "k6/http";
import encoding from "k6/encoding";

import { check, group } from "k6";

import * as pipeline from "./grpc-pipeline-public.js";
import * as pipelineWithBasicAuth from "./grpc-pipeline-public-with-basic-auth.js";
import * as pipelinePrivate from "./grpc-pipeline-private.js";
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

  // CE edition uses Basic Auth for all authenticated requests
  const basicAuth = encoding.b64encode(`${constant.defaultUsername}:${constant.defaultPassword}`);

  var metadata = {
    "metadata": {
      "Authorization": `Basic ${basicAuth}`
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
  // Tests with invalid Basic Auth credentials (should be rejected)
  pipelineWithBasicAuth.CheckCreate(data);
  pipelineWithBasicAuth.CheckList(data);
  pipelineWithBasicAuth.CheckGet(data);
  pipelineWithBasicAuth.CheckUpdate(data);
  pipelineWithBasicAuth.CheckRename(data);

  pipeline.CheckCreate(data);
  pipeline.CheckList(data);
  pipeline.CheckGet(data);
  pipeline.CheckUpdate(data);
  pipeline.CheckRename(data);

  // Private Service API tests (service-to-service communication)
  pipelinePrivate.CheckLookUpPipelineAdmin(data);
  pipelinePrivate.CheckListPipelinesAdmin(data);

  // Trigger tests (updated to use new Namespace APIs)
  trigger.CheckTrigger(data);
  triggerAsync.CheckTrigger(data);
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    pipelineClient.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    var listRes = pipelineClient.invoke(
      "pipeline.v1beta.PipelinePublicService/ListNamespacePipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 1000,
      },
      data.metadata
    );

    if (listRes.message && listRes.message.pipelines) {
      for (const pipeline of listRes.message.pipelines) {
        var deleteRes = pipelineClient.invoke(
          `pipeline.v1beta.PipelinePublicService/DeleteNamespacePipeline`,
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
    constant.pipelinedb.exec(q);
    constant.pipelinedb.close();
  });
}
