import http from "k6/http";

import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTrigger(data) {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.simpleRecipe
  );

  group("Pipelines API: Trigger an async pipeline for single image and single model", () => {

    check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, JSON.stringify(reqBody), data.header), {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
    });


    check(http.request("POST", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(constant.simplePayload), data.header), {
      [`POST /v1beta/${constant.namespace}/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1beta/${constant.namespace}/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`, null, data.header), {
    [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}
