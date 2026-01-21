import http from "k6/http";
import grpc from "k6/net/grpc";
import encoding from "k6/encoding";

import {
  check,
  group,
} from "k6";

import { pipelinePublicHost } from "./const.js";

import * as componentDefinition from "./rest-component-definition.js";
import * as constant from "./const.js";
import * as integration from "./rest-integration.js";
import * as pipelinePublic from './rest-pipeline-public.js';
import * as pipelinePublicWithBasicAuth from './rest-pipeline-public-with-basic-auth.js';
import * as trigger from './rest-trigger.js';
import * as triggerAsync from './rest-trigger-async.js';

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  // CE edition uses Basic Auth for all authenticated requests
  const basicAuth = encoding.b64encode(`${constant.defaultUsername}:${constant.defaultPassword}`);

  var header = {
    "headers": {
      "Authorization": `Basic ${basicAuth}`,
      "Content-Type": "application/json",
    },
    "timeout": "600s",
  }

  var resp = http.request("GET", `${constant.mgmtPublicHost}/v1beta/user`, {}, { headers: { "Authorization": `Basic ${basicAuth}` } })
  return { header: header, expectedOwner: resp.json().user }
}

export default function (data) {

  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      check(http.request("GET", `${pipelinePublicHost}/v1beta/health/pipeline`), {
        "GET /health/pipeline response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Tests with invalid Basic Auth credentials (should be rejected)
  pipelinePublicWithBasicAuth.CheckCreate(data);
  pipelinePublicWithBasicAuth.CheckList(data);
  pipelinePublicWithBasicAuth.CheckGet(data);
  pipelinePublicWithBasicAuth.CheckUpdate(data);
  pipelinePublicWithBasicAuth.CheckRename(data);

  pipelinePublic.CheckCreate(data);
  pipelinePublic.CheckList(data);
  pipelinePublic.CheckGet(data);
  pipelinePublic.CheckUpdate(data);
  pipelinePublic.CheckRename(data);

  trigger.CheckTrigger(data);
  trigger.CheckPipelineRuns(data);
  triggerAsync.CheckTrigger(data);

  componentDefinition.CheckList(data);

  integration.CheckIntegrations(data);
  integration.CheckConnections(data);
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    var listRes = http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=100`, null, data.header);
    var pipelines = listRes.status === 200 ? listRes.json("pipelines") || [] : [];
    for (const pipeline of pipelines) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipeline.id}`, null, data.header), {
        [`DELETE /v1beta/${constant.namespace}/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Integration API: Delete data created by this test", () => {
    var q = `DELETE FROM connection WHERE id LIKE '${constant.dbIDPrefix}%';`;
    constant.pipelinedb.exec(q);

    q = `DELETE FROM pipeline WHERE id LIKE '${constant.dbIDPrefix}%';`;
    constant.pipelinedb.exec(q);

    constant.pipelinedb.close();
  });
}
