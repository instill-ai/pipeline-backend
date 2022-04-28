import http from "k6/http";

import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as constant from "./const.js";
import * as pipeline from './rest-pipeline.js';
import * as trigger from './rest-trigger.js';

const pipelineHost = "http://localhost:8080";
const modelHost = "http://localhost:8081";

const model_name = constant.detectionModel.name;
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
      fd.append("name", model_name);
      fd.append("description", randomString(20));
      fd.append("task", "TASK_DETECTION");
      fd.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${modelHost}/models/upload`, fd.body(), {
        headers: {
          "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
        },
      }), {
        "POST /models/upload (multipart) det response Status": (r) =>
          r.status === 200, // TODO: update status to 201
        "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
        "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
        "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_DETECTION",
        "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${modelHost}/models/${model_name}/versions/1`, payload, {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
        [`PATCH /models/${model_name}/versions/1 online task cls response model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
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

  trigger.CheckTriggerImageDirect()
}

export function teardown(data) {
  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${modelHost}/models/${model_name}`, null, {
      headers: {
        "Content-Type": "application/json"
      },
    }), {
      "DELETE clean up response status": (r) =>
        r.status === 200 // TODO: update status to 201
    });
  });
}
