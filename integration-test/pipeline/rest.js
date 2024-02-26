import http from "k6/http";

import {
  check,
  group,
} from "k6";

import {
  pipelinePublicHost,
} from "./const.js";

import * as constant from "./const.js";
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
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }


  group("Connector Backend API: Create a CSV destination connector 1", function () {

    var res = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
      JSON.stringify({
        "id": constant.dstCSVConnID1,
        "connector_definition_name": "connector-definitions/airbyte-destination",
        "configuration": {
          "destination": "airbyte-destination-csv",
          "destination_path": "/local/pipeline-backend-test-1"
        }
      }), header)

    check(res, {
      [`POST /v1beta/${constant.namespace}/connectors response status for creating CSV destination connector 201`]: (r) => r.status === 201,
    })

    http.request("POST", `${pipelinePublicHost}/v1beta/connectors/${constant.dstCSVConnID1}/connect`, {}, header)

  });

  group("Connector Backend API: Create a CSV destination connector 2", function () {

    var res = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors`,
      JSON.stringify({
        "id": constant.dstCSVConnID2,
        "connector_definition_name": "connector-definitions/airbyte-destination",
        "configuration": {
          "destination": "airbyte-destination-csv",
          "destination_path": "/local/pipeline-backend-test-2"
        }
      }), header)

    check(res, {
      [`POST /v1beta/${constant.namespace}/connectors response status for creating CSV destination connector 201`]: (r) => r.status === 201,
    })

    http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${constant.dstCSVConnID2}/connect`, {}, header)

  });
  var resp = http.request("GET", `${constant.mgmtPublicHost}/v1beta/user`, {}, {headers: {"Authorization": `Bearer ${loginResp.json().access_token}`}})
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
    pipelinePrivate.CheckList(data)
    pipelinePrivate.CheckLookUp(data)

  } else {

    pipelinePublicWithJwt.CheckCreate(data)
    pipelinePublicWithJwt.CheckList(data)
    pipelinePublicWithJwt.CheckGet(data)
    pipelinePublicWithJwt.CheckUpdate(data)
    pipelinePublicWithJwt.CheckRename(data)
    pipelinePublicWithJwt.CheckLookUp(data)
    pipelinePublic.CheckCreate(data)
    pipelinePublic.CheckList(data)
    pipelinePublic.CheckGet(data)
    pipelinePublic.CheckUpdate(data)
    pipelinePublic.CheckRename(data)
    pipelinePublic.CheckLookUp(data)

    trigger.CheckTrigger(data)
    triggerAsync.CheckTrigger(data)

  }
}

export function teardown(data) {

  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?page_size=100`, null, data.header).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipeline.id}`, null, data.header), {
        [`DELETE /v1beta/${constant.namespace}/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${constant.dstCSVConnID1}`, null, data.header), {
      [`DELETE /v1beta/${constant.namespace}/connectors/${constant.dstCSVConnID1} response status 204`]: (r) => r.status === 204,
    });
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/connectors/${constant.dstCSVConnID2}`, null, data.header), {
      [`DELETE /v1beta/${constant.namespace}/connectors/${constant.dstCSVConnID2} response status 204`]: (r) => r.status === 204,
    });
  });
}
