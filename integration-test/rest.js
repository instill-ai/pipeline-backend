import http from "k6/http";

import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";
import * as pipeline from './rest-pipeline.js';
import * as trigger from './rest-trigger.js';

const pipelineHost = "http://localhost:8080/v1alpha";
const modelHost = "http://localhost:8081";

const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  // Prepare sample model in model-backend
  {
    group("Model Backend API: Create a detection model", function () {
      let fd = new FormData();
      fd.append("name", constant.model_id);
      fd.append("description", randomString(20));
      fd.append("content", http.file(det_model, "dummy-det-model.zip"));

      check(http.request("POST", `${modelHost}/models/upload`, fd.body(), {
        headers: {
          "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
        },
      }), {
        "POST /models/upload (multipart) det response Status": (r) => r.status === 200, // TODO: update status to 201
        "POST /models/upload (multipart) task det response model.name": (r) => r.json().model.name === constant.model_id,
        "POST /models/upload (multipart) task det response model.full_name": (r) => r.json().model.full_name === `local-user/${constant.model_id}`,
        "POST /models/upload (multipart) task det response model.instances.length": (r) => r.json().model.instances.length === 1,
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });

      check(http.request("PATCH", `${modelHost}/models/${constant.model_id}/instances/latest`, payload, {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`PATCH /models/${constant.model_id}/instances/latest online task det response status`]: (r) => r.status === 200, // TODO: update status to 201
        [`PATCH /models/${constant.model_id}/instances/latest online task det response instance.name`]: (r) => r.json().instance.name === "latest",
        [`PATCH /models/${constant.model_id}/instances/latest online task det response instance.model_definition_id`]: (r) => helper.isUUID(r.json().instance.model_definition_id),
        [`PATCH /models/${constant.model_id}/instances/latest online task det response instance.created_at`]: (r) => r.json().instance.created_at !== undefined,
        [`PATCH /models/${constant.model_id}/instances/latest online task det response instance.updated_at`]: (r) => r.json().instance.updated_at !== undefined,
        [`PATCH /models/${constant.model_id}/instances/latest online task det response instance.status`]: (r) => r.json().instance.status === "STATUS_ONLINE",
      });
    });
  }
}

export default function (data) {
  let res;

  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      check(http.request("GET", `${pipelineHost}/health/pipeline`), {
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

  // trigger.CheckTriggerImageDirect()
}

export function teardown(data) {
  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${modelHost}/models/${constant.model_id}`, null, {
      headers: {
        "Content-Type": "application/json"
      },
    }), {
      "DELETE clean up response status": (r) => r.status === 200 // TODO: update status to 204
    });
  });
}
