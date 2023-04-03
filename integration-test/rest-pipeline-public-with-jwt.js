import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckCreate() {

  group(`Pipelines API: Create a pipeline [with random "jwt-sub" header]`, () => {

    var reqBody = Object.assign(
      {
        id: randomString(63),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Cannot create a pipeline of a non-exist user
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] POST /v1alpha/pipelines response status is 500`]: (r) => r.status === 500
    });

  });
}

export function CheckList() {

  group(`Pipelines API: List pipelines [with random "jwt-sub" header]`, () => {

    // Cannot list pipelines of a non-exist user
    check(http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines`, null, constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] GET /v1alpha/pipelines response status is 500`]: (r) => r.status === 500
    });
  });
}

export function CheckGet() {

  group(`Pipelines API: Get a pipeline [with random "jwt-sub" header]`, () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Create a pipeline
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    // Cannot get a pipeline of a non-exist user
    check(http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] GET /v1alpha/pipelines/${reqBody.id} response status is 500`]: (r) => r.status === 500
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdate() {

  group(`Pipelines API: Update a pipeline [with random "jwt-sub" header]`, () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params)

    check(resOrigin, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var reqBodyUpdate = Object.assign(
      {
        uid: "output-only-to-be-ignored",
        mode: "MODE_ASYNC",
        name: "pipelines/some-string-to-be-ignored",
        description: randomString(50),
      },
    )

    // Cannot update a pipeline of a non-exist user
    check(http.request("PATCH", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] PATCH /v1alpha/pipelines/${reqBody.id} response status is 500`]: (r) => r.status === 500
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdateState() {

  group(`Pipelines API: Update a pipeline state [with random "jwt-sub" header]`, () => {

    var reqBodySync = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBodySync), constant.params), {
      "POST /v1alpha/pipelines sync pipeline creation response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines sync pipeline creation response pipeline state ACTIVE": (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBodyAsync), constant.params), {
      "POST /v1alpha/pipelines async pipeline creation response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines async pipeline creation response pipeline state ACTIVE": (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    // Cannot activate a pipeline of a non-exist user
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBodySync.id}/activate`, null, constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] POST /v1alpha/pipelines/${reqBodySync.id}/activate response status is 500 for sync pipeline`]: (r) => r.status === 500
    });

    // Cannot deactivate a pipeline of a non-exist user
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBodySync.id}/deactivate`, null, constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] POST /v1alpha/pipelines/${reqBodySync.id}/deactivate response status is 500 for sync pipeline`]: (r) => r.status === 500
    });

    // Delete the pipelines
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBodySync.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBodySync.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckRename() {

  group(`Pipelines API: Rename a pipeline [with random "jwt-sub" header]`, () => {

    var id = randomString(10) 
    var reqBody = Object.assign(
      {
        id: id,
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Create a pipeline
    var res = http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params)

    check(res, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1alpha/pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10)

    // Cannot rename a pipeline of a non-exist user
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${res.json().pipeline.id}/rename`, JSON.stringify(reqBody), constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] POST /v1alpha/pipelines/${res.json().pipeline.id}/rename response status is 500`]: (r) => r.status === 500
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckLookUp() {

  group(`Pipelines API: Look up a pipeline by uid [with random "jwt-sub" header]`, () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.detSyncHTTPSingleModelRecipe
    )

    // Create a pipeline
    var res = http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params)

    check(res, {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    // Cannot look up a pipeline of a non-exist user
    check(http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines/${res.json().pipeline.id}/lookUp`, null, constant.paramsWithJwt), {
      [`[with random "jwt-sub" header] POST /v1alpha/pipelines/${res.json().pipeline.id}/lookUp response status is 500`]: (r) => r.status === 500
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
