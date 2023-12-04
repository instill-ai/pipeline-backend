import http from "k6/http";
import { check, group } from "k6";

import { pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as dataConnectorDefinition from './rest-data-connector-definition.js';
import * as dataConnectorPublic from './rest-data-connector-public.js';
import * as dataConnectorPublicWithJwt from './rest-data-connector-public-with-jwt.js';
import * as dataConnectorPrivate from './rest-data-connector-private.js';

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1alpha/auth/login`, JSON.stringify({
    "username": constant.defaultUsername,
    "password": constant.defaultPassword,
  }))

  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1alpha//auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var header = {
    "headers": {
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }


  group("Connector API: Pre delete all connector", () => {
    for (const connector of http.request("GET", `${pipelinePublicHost}/v1alpha/${constant.namespace}/connectors`, null, header).json("connectors")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/${constant.namespace}/connectors/${connector.id}`, null, header), {
        [`DELETE /v1alpha/${constant.namespace}/connectors/${connector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  return header
}

export default function (header) {

  /*
   * Connector API - API CALLS
   */

  // Health check
  group("Connector API: Health check", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1alpha/health/pipeline`), {
      "GET /health/pipeline response status is 200": (r) => r.status === 200,
    });
  });

  // private API do not expose to public.
  if (!constant.apiGatewayMode) {

    // data connectors
    dataConnectorPrivate.CheckList(header)
    dataConnectorPrivate.CheckLookUp(header)


  } else {

    // data public with jwt-sub
    dataConnectorPublicWithJwt.CheckCreate(header)
    dataConnectorPublicWithJwt.CheckList(header)
    dataConnectorPublicWithJwt.CheckGet(header)
    dataConnectorPublicWithJwt.CheckUpdate(header)
    dataConnectorPublicWithJwt.CheckLookUp(header)
    dataConnectorPublicWithJwt.CheckState(header)
    dataConnectorPublicWithJwt.CheckRename(header)
    dataConnectorPublicWithJwt.CheckExecute(header)
    dataConnectorPublicWithJwt.CheckTest(header)

    // data connector definitions
    dataConnectorDefinition.CheckList(header)
    dataConnectorDefinition.CheckGet(header)

    // data connectors
    dataConnectorPublic.CheckCreate(header)
    dataConnectorPublic.CheckList(header)
    dataConnectorPublic.CheckGet(header)
    dataConnectorPublic.CheckUpdate(header)
    dataConnectorPublic.CheckConnect(header)
    dataConnectorPublic.CheckLookUp(header)
    dataConnectorPublic.CheckState(header)
    dataConnectorPublic.CheckRename(header)
    dataConnectorPublic.CheckExecute(header)
    dataConnectorPublic.CheckTest(header)
  }




}

export function teardown(header) {
  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines?page_size=100`, null, header).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${pipeline.id}`), {
        [`DELETE /v1alpha/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}
