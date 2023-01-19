import http from "k6/http";

import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelineHost, connectorHost, modelHost } from "./const.js";

import * as constant from "./const.js";
import * as pipeline from './rest-pipeline.js';
import * as triggerSync from './rest-trigger-sync.js';
import * as triggerAsync from './rest-trigger-async.js';

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  group("Connector Backend API: Create a http source connector", function () {

    var res = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
      JSON.stringify({
        "id": "source-http",
        "source_connector_definition": "source-connector-definitions/source-http",
        "connector": {
          "configuration": {}
        }
      }), {
      headers: { "Content-Type": "application/json" },
    })
    check(res, {
      "POST /v1alpha/source-connectors response status for creating HTTP source connector 201": (r) => r.status === 201,
    })

  });

  group("Connector Backend API: Create a http destination connector", function () {

    var res = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
      JSON.stringify({
        "id": "destination-http",
        "destination_connector_definition": "destination-connector-definitions/destination-http",
        "connector": {
          "configuration": {}
        }
      }), {
      headers: { "Content-Type": "application/json" },
    })

    check(res, {
      "POST /v1alpha/destination-connectors response status for creating HTTP destination connector 201": (r) => r.status === 201,
    })

  });

  group("Connector Backend API: Create a gRPC source connector", function () {

    var res = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
      JSON.stringify({
        "id": "source-grpc",
        "source_connector_definition": "source-connector-definitions/source-grpc",
        "connector": {
          "configuration": {}
        }
      }), {
      headers: { "Content-Type": "application/json" },
    })
    check(res, {
      "POST /v1alpha/source-connectors response status for creating gRPC source connector 201": (r) => r.status === 201,
    })

  });

  group("Connector Backend API: Create a gRPC destination connector", function () {

    var res = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
      JSON.stringify({
        "id": "destination-grpc",
        "destination_connector_definition": "destination-connector-definitions/destination-grpc",
        "connector": {
          "configuration": {}
        }
      }), {
      headers: { "Content-Type": "application/json" },
    })

    check(res, {
      "POST /v1alpha/destination-connectors response status for creating gRPC destination connector 201": (r) => r.status === 201,
    })

  });

  group("Connector Backend API: Create a CSV destination connector", function () {

    var res = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
      JSON.stringify({
        "id": constant.dstCSVConnID,
        "destination_connector_definition": "destination-connector-definitions/destination-csv",
        "connector": {
          "configuration": {
            "destination_path": "/local/pipeline-backend-test"
          }
        }
      }), {
      headers: { "Content-Type": "application/json" },
    })

    check(res, {
      "POST /v1alpha/destination-connectors response status for creating CSV destination connector 201": (r) => r.status === 201,
    })

    // Check connector state being updated in 120 secs
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = http.request("GET", `${connectorHost}/v1alpha/destination-connectors/${constant.dstCSVConnID}`)
      if (res.json().destination_connector.connector.state === "STATE_CONNECTED") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

  });

  group("Model Backend API: Deploy a detection model", function () {
    let fd = new FormData();
    let model_description = randomString(20)
    fd.append("id", constant.model_id);
    fd.append("description", model_description);
    fd.append("model_definition", constant.model_def_name);
    fd.append("content", http.file(constant.det_model, "dummy-det-model.zip"));
    let createClsModelRes = http.request("POST", `${modelHost}/v1alpha/models/multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
      },
    })
    check(createClsModelRes, {
      "POST /v1alpha/models/multipart task det response status": (r) => r.status === 201
    });

    // Check model creation finished
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      let res = http.get(`${modelHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
        headers: { "Content-Type": "application/json" },
      })
      if (res.json().operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    var res = http.post(`${modelHost}/v1alpha/models/${constant.model_id}/instances/latest/deploy`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    })

    check(res, {
      [`POST /v1alpha/models/${constant.model_id}/instances/latest/deploy online task det response status`]: (r) => r.status === 200
    });

    // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = http.get(`${modelHost}/v1alpha/models/${constant.model_id}/instances/latest`, {
        headers: { "Content-Type": "application/json" },
      })
      if (res.json().instance.state === "STATE_ONLINE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

  });

}

export default function (data) {

  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      check(http.request("GET", `${pipelineHost}/v1alpha/health/pipeline`), {
        "GET /health/pipeline response status is 200": (r) => r.status === 200,
      });
    });
  }

  pipeline.CheckCreate()
  pipeline.CheckList()
  pipeline.CheckGet()
  pipeline.CheckUpdate()
  pipeline.CheckUpdateState()
  pipeline.CheckRename()
  pipeline.CheckLookUp()

  triggerSync.CheckTriggerSyncSingleImageSingleModelInst()
  triggerSync.CheckTriggerSyncMultiImageSingleModelInst()
  triggerSync.CheckTriggerSyncMultiImageMultiModelInst()

  triggerAsync.CheckTriggerAsyncSingleImageSingleModelInst()
  triggerAsync.CheckTriggerAsyncMultiImageSingleModelInst()
  triggerAsync.CheckTriggerAsyncMultiImageMultiModelInst()

}

export function teardown(data) {

  group("Connector Backend API: Delete the http source connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/source-http`), {
      [`DELETE /v1alpha/source-connectors/source-http response status 204`]: (r) => r.status === 204,
    });
  });

  group("Connector Backend API: Delete the http destination connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/destination-http`), {
      [`DELETE /v1alpha/destination-connectors/destination-http response status 204`]: (r) => r.status === 204,
    });
  });

  group("Connector Backend API: Delete the gRPC source connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/source-grpc`), {
      [`DELETE /v1alpha/source-connectors/source-grpc response status 204`]: (r) => r.status === 204,
    });
  });

  group("Connector Backend API: Delete the gRPC destination connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/destination-grpc`), {
      [`DELETE /v1alpha/destination-connectors/destination-grpc response status 204`]: (r) => r.status === 204,
    });
  });

  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${constant.dstCSVConnID}`), {
      [`DELETE /v1alpha/destination-connectors/${constant.dstCSVConnID} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${modelHost}/v1alpha/models/${constant.model_id}`, null, {
      headers: { "Content-Type": "application/json" }
    }), {
      [`DELETE /v1alpha/models/${constant.model_id} response status is 204`]: (r) => r.status === 204,
    });
  });

}
