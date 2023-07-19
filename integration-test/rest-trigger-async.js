import http from "k6/http";

import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTriggerAsyncSingleImageSingleModel() {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncSingleModelRecipe
  );

  group("Pipelines API: Trigger an async pipeline for single image and single model", () => {

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });
    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

    var payloadImageBase64 = {
      inputs: [
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
    [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}

export function CheckTriggerAsyncMultiImageSingleModel() {
  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncSingleModelRecipe
  );

  group("Pipelines API: Trigger an async pipeline for multiple images and single model", () => {

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });
    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

    var payloadImageBase64 = {
      inputs: [
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
    [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}

export function CheckTriggerAsyncMultiImageMultiModel() {
  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncMultiModelRecipe
  );

  group("Pipelines API: Trigger an async pipeline for multiple images and multiple models", () => {

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });
    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

    var payloadImageBase64 = {
      inputs: [
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });


    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerAsyncMultiImageMultiModelMultipleDestination() {
  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncMultiModelMultipleDestinationRecipe
  );

  group("Pipelines API: Trigger an async pipeline for multiple images and multiple models", () => {

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });
    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

    var payloadImageBase64 = {
      inputs: [
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
        {
          images: [{
            blob: constant.dogImg,
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (base64) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });


    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerAsyncSingleResponse() {
  group("Pipelines API: Trigger an async pipeline and get the result from GetOperation", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detAsyncSingleResponseRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqBody), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });
    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    var resp = http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsync`, JSON.stringify(payloadImageURL), constant.params);
    check(resp, {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (url) response status is 200`]: (r) => r.json().operation.name.startsWith("operations/"),
    });

    for (var i = 0; i < 30; ++i) {
      var resp = http.request("GET", `${pipelinePublicHost}/v1alpha/${resp.json().operation.name}`);
      if (resp.json().operation.done) {
        break
      }
      sleep(1)
    }

    check(http.request("GET", `${pipelinePublicHost}/v1alpha/${resp.json().operation.name}`, null, constant.params), {
      [`GET /v1alpha/pipelines/${resp.json().operation.name} response 200`]:
        (r) => r.status === 200,
      [`GET /v1alpha/pipelines/${resp.json().operation.name} response done = true`]:
        (r) => r.json().operation.done === true,
      [`GET /v1alpha/pipelines/${resp.json().operation.name} response outputs.length = ${payloadImageURL["inputs"].length}`]:
        (r) => r.json().operation.response.outputs.length === payloadImageURL["inputs"].length,
      [`GET /v1alpha/pipelines/${resp.json().operation.name} response outputs[0].images.length = ${payloadImageURL["inputs"][0].images.length}`]:
        (r) => r.json().operation.response.outputs[0].images.length === payloadImageURL["inputs"][0].images.length,
    }
    );
    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
