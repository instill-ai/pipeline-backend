import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTrigger() {

  group("Pipelines API: Trigger a pipeline for single image and single model", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/${constant.namespace}/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/${constant.namespace}/pipelines response status is 201 (HTTP pipeline)": (r) => r.status === 201,
    });

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/${constant.namespace}/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(constant.simplePayload), constant.params), {
      [`POST /v1alpha/${constant.namespace}/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
    });


    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/${constant.namespace}/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/${constant.namespace}/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
