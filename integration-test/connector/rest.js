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
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }


  group("Connector API: Pre delete all connector", () => {
    for (const connector of http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`, null, header).json("connectors")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${connector.id}`, null, header), {
        [`DELETE /v1beta/${constant.namespace}/connectors/${connector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  var resp = http.request("GET", `${constant.mgmtPublicHost}/v1beta/user`, {}, {headers: {"Authorization": `Bearer ${loginResp.json().access_token}`}})
  return {header: header, expectedOwner: resp.json().user}
}

export default function (data) {

  /*
   * Connector API - API CALLS
   */

  // Health check
  group("Connector API: Health check", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1beta/health/pipeline`), {
      "GET /health/pipeline response status is 200": (r) => r.status === 200,
    });
  });

  // private API do not expose to public.
  if (!constant.apiGatewayMode) {

    // data connectors
    dataConnectorPrivate.CheckList(data)
    dataConnectorPrivate.CheckLookUp(data)


  } else {

    // data public with Instill-User-Uid
    dataConnectorPublicWithJwt.CheckCreate(data)
    dataConnectorPublicWithJwt.CheckList(data)
    dataConnectorPublicWithJwt.CheckGet(data)
    dataConnectorPublicWithJwt.CheckUpdate(data)
    dataConnectorPublicWithJwt.CheckLookUp(data)
    dataConnectorPublicWithJwt.CheckState(data)
    dataConnectorPublicWithJwt.CheckRename(data)
    dataConnectorPublicWithJwt.CheckExecute(data)
    dataConnectorPublicWithJwt.CheckTest(data)

    // data connector definitions
    dataConnectorDefinition.CheckList(data)
    dataConnectorDefinition.CheckGet(data)

    // data connectors
    dataConnectorPublic.CheckCreate(data)
    dataConnectorPublic.CheckList(data)
    dataConnectorPublic.CheckGet(data)
    dataConnectorPublic.CheckUpdate(data)
    dataConnectorPublic.CheckConnect(data)
    dataConnectorPublic.CheckLookUp(data)
    dataConnectorPublic.CheckState(data)
    dataConnectorPublic.CheckRename(data)
    dataConnectorPublic.CheckExecute(data)
    dataConnectorPublic.CheckTest(data)
  }




}

export function teardown(data) {
  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1beta/pipelines?page_size=100`, null, data.header).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1beta/pipelines/${pipeline.id}`), {
        [`DELETE /v1beta/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}
