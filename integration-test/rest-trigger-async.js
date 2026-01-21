import http from "k6/http";

import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTrigger(data) {
  group("Pipelines API: Trigger an async pipeline", () => {
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, JSON.stringify(reqBody), data.header);
    check(createRes, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
    });

    var pipelineId = createRes.json().pipeline ? createRes.json().pipeline.id : null;
    if (!pipelineId) {
      console.log("Failed to create pipeline, skipping async trigger test");
      return;
    }

    check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}/trigger-async`, JSON.stringify(constant.simplePayload), data.header), {
      [`POST /v1beta/${constant.namespace}/pipelines/{id}/trigger-async response status is 200`]: (r) => r.status === 200,
      [`POST /v1beta/${constant.namespace}/pipelines/{id}/trigger-async response has operation name`]: (r) => r.json().operation && r.json().operation.name && r.json().operation.name.startsWith("operations/"),
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`, null, data.header), {
      [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Pipelines API: Trigger an async pipeline with YAML recipe", () => {
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    var createRes = http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, JSON.stringify(reqBody), data.header);
    check(createRes, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201 (YAML)": (r) => r.status === 201,
    });

    var pipelineId = createRes.json().pipeline ? createRes.json().pipeline.id : null;
    if (!pipelineId) {
      console.log("Failed to create pipeline, skipping async trigger test");
      return;
    }

    check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}/trigger-async`, JSON.stringify(constant.simplePayload), data.header), {
      [`POST /v1beta/${constant.namespace}/pipelines/{id}/trigger-async response status is 200 (YAML)`]: (r) => r.status === 200,
      [`POST /v1beta/${constant.namespace}/pipelines/{id}/trigger-async response has operation name (YAML)`]: (r) => r.json().operation && r.json().operation.name && r.json().operation.name.startsWith("operations/"),
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`, null, data.header), {
      [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204 (YAML)`]: (r) => r.status === 204,
    });
  });
}
