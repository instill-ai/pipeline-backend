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

  group("Connector Backend API: Create a CSV destination connector 1", function () {

    var res = http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources`,
      JSON.stringify({
        "id": constant.dstCSVConnID1,
        "connector_definition_name": "connector-definitions/airbyte-destination-csv",
        "configuration": {
          "destination_path": "/local/pipeline-backend-test-1"
        }
      }), constant.params)

    check(res, {
      "POST /v1alpha/connectors response status for creating CSV destination connector 201": (r) => r.status === 201,
    })

    http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources/${constant.dstCSVConnID1}/connect`, {}, constant.params)

  });

  group("Connector Backend API: Create a CSV destination connector 2", function () {

    var res = http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources`,
      JSON.stringify({
        "id": constant.dstCSVConnID2,
        "connector_definition_name": "connector-definitions/airbyte-destination-csv",
        "configuration": {
          "destination_path": "/local/pipeline-backend-test-2"
        }
      }), constant.params)

    check(res, {
      "POST /v1alpha/connectors response status for creating CSV destination connector 201": (r) => r.status === 201,
    })

    http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources/${constant.dstCSVConnID2}/connect`, {}, constant.params)

  });

}

export default function (data) {

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
    pipelinePrivate.CheckList()
    pipelinePrivate.CheckLookUp()

    pipelinePublicWithJwt.CheckCreate()
    pipelinePublicWithJwt.CheckList()
    pipelinePublicWithJwt.CheckGet()
    pipelinePublicWithJwt.CheckUpdate()
    pipelinePublicWithJwt.CheckRename()
    pipelinePublicWithJwt.CheckLookUp()
  }

  pipelinePublic.CheckCreate()
  pipelinePublic.CheckList()
  pipelinePublic.CheckGet()
  pipelinePublic.CheckUpdate()
  pipelinePublic.CheckRename()
  pipelinePublic.CheckLookUp()

  trigger.CheckTrigger()
  triggerAsync.CheckTrigger()


}

export function teardown(data) {

  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines?page_size=100`).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${pipeline.id}`), {
        [`DELETE /v1alpha/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connector-resources/${constant.dstCSVConnID1}`), {
      [`DELETE /v1alpha/connector-resources/${constant.dstCSVConnID1} response status 204`]: (r) => r.status === 204,
    });
  });
  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connector-resources/${constant.dstCSVConnID2}`), {
      [`DELETE /v1alpha/connector-resources/${constant.dstCSVConnID2} response status 204`]: (r) => r.status === 204,
    });
  });
}
