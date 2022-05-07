import http from "k6/http";

import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";
import * as pipeline from './rest-pipeline.js';
import * as trigger from './rest-trigger.js';

const pipelineHost = "http://localhost:8080";
const modelHost = "http://localhost:8081";

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
        "POST /v1alpha/models (multipart) github task det response status": (r) => r.status === 201,
        "POST /v1alpha/models/upload (multipart) task det response model.name": (r) => r.json().model.name === `models/${constant.model_id}`,
        "POST /v1alpha/models/upload (multipart) task det response model.uid": (r) => r.json().model.uid !== undefined,
        "POST /v1alpha/models/upload (multipart) task det response model.id": (r) => r.json().model.id === constant.model_id,
        "POST /v1alpha/models/upload (multipart) task det response model.description": (r) => r.json().model.description === model_description,
        "POST /v1alpha/models/upload (multipart) task det response model.model_definition": (r) => r.json().model.model_definition === constant.model_def_name,
        "POST /v1alpha/models/upload (multipart) task det response model.configuration": (r) => r.json().model.configuration !== undefined,
        "POST /v1alpha/models/upload (multipart) task det response model.visibility": (r) => r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/upload (multipart) task det response model.owner": (r) => r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/upload (multipart) task det response model.create_time": (r) => r.json().model.create_time !== undefined,
        "POST /v1alpha/models/upload (multipart) task det response model.update_time": (r) => r.json().model.update_time !== undefined,
      });

      check(http.post(`${modelHost}/v1alpha/models/${constant.model_id}/instances/latest:deploy`, {}, {
        headers: {
          "Content-Type": "application/json"
        },
      }), {
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response status`]: (r) => r.status === 200,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.name`]: (r) => r.json().instance.name === `models/${constant.model_id}/instances/latest`,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.uid`]: (r) => r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.id`]: (r) => r.json().instance.id === "latest",
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.state`]: (r) => r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.task`]: (r) => r.json().instance.task === "TASK_DETECTION",
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.model_definition`]: (r) => r.json().instance.model_definition === constant.model_def_name,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.create_time`]: (r) => r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.update_time`]: (r) => r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${constant.model_id}/instances/latest:deploy online task det response instance.configuration`]: (r) => r.json().instance.configuration !== undefined,
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

  trigger.CheckTriggerImageDirect()
}

export function teardown(data) {
  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${modelHost}/v1alpha/models/${constant.model_id}`, null, {
      headers: { "Content-Type": "application/json" }
    }), {
      [`DELETE /v1alpha/models/${constant.model_id} response status is 200`]: (r) => r.status === 200,
    });
  });
}
