import http from "k6/http";

import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";
import * as pipeline from './rest-pipeline.js';
import * as trigger from './rest-trigger.js';

const pipelineHost = "http://localhost:8081";
const connectorHost = "http://localhost:8082";
const modelHost = "http://localhost:8083";

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  group("Connector Backend API: Create a http source connector", function () {
    check(http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
      JSON.stringify({
        "id": "source-http",
        "source_connector_definition": "source-connector-definitions/source-http",
        "connector": {
          "configuration": JSON.stringify({})
        }
      }), {
      headers: { "Content-Type": "application/json" },
    }), {
      "POST /v1alpha/source-connectors response status for creating directness HTTP source connector 201": (r) => r.status === 201,
    })
  });

  group("Connector Backend API: Create a http destination connector", function () {
    check(http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
      JSON.stringify({
        "id": "destination-http",
        "destination_connector_definition": "destination-connector-definitions/destination-http",
        "connector": {
          "configuration": JSON.stringify({})
        }
      }), {
      headers: { "Content-Type": "application/json" },
    }), {
      "POST /v1alpha/destination-connectors response status for creating directness HTTP destination connector 201": (r) => r.status === 201,
    })
  });

  group("Connector Backend API: Create a CSV destination connector", function () {
    check(http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
      JSON.stringify({
        "id": constant.dstCSVConnID,
        "destination_connector_definition": "destination-connector-definitions/destination-csv",
        "connector": {
          "configuration": JSON.stringify({
            "connection_specification": {
              "supports_incremental": true,
              "connection_specification": {
                "destination_path": "/local"
              },
              "supported_destination_sync_modes": [2, 1]
            }
          })
        }
      }), {
      headers: { "Content-Type": "application/json" },
    }), {
      "POST /v1alpha/destination-connectors response status for creating CSV destination connector 201": (r) => r.status === 201,
    })
  });

  group("Model Backend API: Deploy a detection model", function () {
    let fd = new FormData();
    let model_description = randomString(20)
    fd.append("name", "models/" + constant.model_id);
    fd.append("description", model_description);
    fd.append("model_definition_name", constant.model_def_name);
    fd.append("content", http.file(constant.det_model, "dummy-det-model.zip"));
    check(http.request("POST", `${modelHost}/v1alpha/models/upload`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
      },
    }), {
      "POST /v1alpha/models (multipart) github task det response status": (r) => r.status === 201
    });

    check(http.post(`${modelHost}/v1alpha/models/${constant.model_id}/instances/latest:deploy`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    }), {
      [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response status`]: (r) => r.status === 200
    });

  });

}

export default function (data) {
  let res;

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

  trigger.CheckTriggerImageDirect()
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

  group("Connector Backend API: Delete the csv destination connector", function () {
    check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${constant.dstCSVConnID}`), {
      [`DELETE /v1alpha/destination-connectors/${constant.dstCSVConnID} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${modelHost}/v1alpha/models/${constant.model_id}`, null, {
      headers: { "Content-Type": "application/json" }
    }), {
      [`DELETE /v1alpha/models/${constant.model_id} response status is 200`]: (r) => r.status === 200,
    });
  });

}
