import http from "k6/http";
import grpc from "k6/net/grpc";

import {
  check,
  group,
} from "k6";

import { pipelinePublicHost } from "./const.js";

import * as componentDefinition from "./rest-component-definition.js";
import * as constant from "./const.js";
import * as integration from "./rest-integration.js";
import * as pipelinePublic from './rest-pipeline-public.js';
import * as pipelinePublicWithJwt from './rest-pipeline-public-with-jwt.js';
import * as pipelinePrivate from './rest-pipeline-private.js';
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

  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1beta/auth/login`, JSON.stringify({
    "username": constant.defaultUsername,
    "password": constant.defaultPassword,
  }))


  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1beta//auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var header = {
    "headers": {
      "Authorization": `Bearer ${loginResp.json().accessToken}`
    },
    "timeout": "600s",
  }

  var resp = http.request("GET", `${constant.mgmtPublicHost}/v1beta/user`, {}, {headers: {"Authorization": `Bearer ${loginResp.json().accessToken}`}})
  return {header: header, expectedOwner: resp.json().user}
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

  if (!constant.apiGatewayMode) {
    pipelinePrivate.CheckList(data);
    pipelinePrivate.CheckLookUp(data);
    return;

  }

  pipelinePublicWithJwt.CheckCreate(data);
  pipelinePublicWithJwt.CheckList(data);
  pipelinePublicWithJwt.CheckGet(data);
  pipelinePublicWithJwt.CheckUpdate(data);
  pipelinePublicWithJwt.CheckRename(data);
  pipelinePublicWithJwt.CheckLookUp(data);
  pipelinePublic.CheckCreate(data);
  pipelinePublic.CheckList(data);
  pipelinePublic.CheckGet(data);
  pipelinePublic.CheckUpdate(data);
  pipelinePublic.CheckRename(data);
  pipelinePublic.CheckLookUp(data);

  trigger.CheckTrigger(data);
  triggerAsync.CheckTrigger(data);

  componentDefinition.CheckList(data);

  integration.CheckIntegrations();
  integration.CheckConnections(data);
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?pageSize=100`, null, data.header).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipeline.id}`, null, data.header), {
        [`DELETE /v1beta/${constant.namespace}/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Integration API: Delete all connections created by this test", () => {
    var q = `DELETE FROM connection WHERE id LIKE '${constant.dbIDPrefix}%';`;
    constant.db.exec(q);

    q = `DELETE FROM pipeline WHERE id LIKE '${constant.dbIDPrefix}%';`;
    constant.db.exec(q);

    constant.db.close();
  });
}
