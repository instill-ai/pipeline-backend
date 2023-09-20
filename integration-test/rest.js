import http from "k6/http";

import {
  check,
  group,
} from "k6";

import {
  pipelinePublicHost,
  connectorPublicHost,
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


  group("Connector Backend API: Create a CSV destination connector 1", function () {

    var res = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
      JSON.stringify({
        "id": constant.dstCSVConnID1,
        "connector_definition_name": "connector-definitions/airbyte-destination-csv",
        "configuration": {
          "destination_path": "/local/pipeline-backend-test-1"
        }
      }), header)

    check(res, {
      [`POST /v1alpha/${constant.namespace}/connector-resources response status for creating CSV destination connector 201`]: (r) => r.status === 201,
    })

    http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources/${constant.dstCSVConnID1}/connect`, {}, header)

  });

  group("Connector Backend API: Create a CSV destination connector 2", function () {

    var res = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
      JSON.stringify({
        "id": constant.dstCSVConnID2,
        "connector_definition_name": "connector-definitions/airbyte-destination-csv",
        "configuration": {
          "destination_path": "/local/pipeline-backend-test-2"
        }
      }), header)

    check(res, {
      [`POST /v1alpha/${constant.namespace}/connector-resources response status for creating CSV destination connector 201`]: (r) => r.status === 201,
    })

    http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${constant.dstCSVConnID2}/connect`, {}, header)

  });

  return header
}

export default function (header) {

  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      check(http.request("GET", `${pipelinePublicHost}/v1alpha/health/pipeline`), {
        "GET /health/pipeline response status is 200": (r) => r.status === 200,
      });
    });
  }

  if (!constant.apiGatewayMode) {
    pipelinePrivate.CheckList(header)
    pipelinePrivate.CheckLookUp(header)

  } else {

    pipelinePublicWithJwt.CheckCreate(header)
    pipelinePublicWithJwt.CheckList(header)
    pipelinePublicWithJwt.CheckGet(header)
    pipelinePublicWithJwt.CheckUpdate(header)
    pipelinePublicWithJwt.CheckRename(header)
    pipelinePublicWithJwt.CheckLookUp(header)
    pipelinePublic.CheckCreate(header)
    pipelinePublic.CheckList(header)
    pipelinePublic.CheckGet(header)
    pipelinePublic.CheckUpdate(header)
    pipelinePublic.CheckRename(header)
    pipelinePublic.CheckLookUp(header)

    trigger.CheckTrigger(header)
    triggerAsync.CheckTrigger(header)

  }
}

export function teardown(header) {

  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1alpha/${constant.namespace}/pipelines?page_size=100`, null, header).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/${constant.namespace}/pipelines/${pipeline.id}`, null, header), {
        [`DELETE /v1alpha/${constant.namespace}/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${constant.dstCSVConnID1}`, null, header), {
      [`DELETE /v1alpha/${constant.namespace}/connector-resources/${constant.dstCSVConnID1} response status 204`]: (r) => r.status === 204,
    });
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${constant.dstCSVConnID2}`, null, header), {
      [`DELETE /v1alpha/${constant.namespace}/connector-resources/${constant.dstCSVConnID2} response status 204`]: (r) => r.status === 204,
    });
  });
}
