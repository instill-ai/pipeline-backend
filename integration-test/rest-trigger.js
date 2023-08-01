import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTriggerSingleImageSingleModel() {

  group("Pipelines API: Trigger a pipeline for single image and single model", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSimpleRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/pipelines response status is 201 (HTTP pipeline)": (r) => r.status === 201,
    });

    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
    });

    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerMultiImageSingleModel() {

  group("Pipelines API: Trigger a pipeline for multiple images and single model", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSimpleRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/activate`, {}, constant.params)

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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
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
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
    });


    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerMultiImageMultiModel() {

  group("Pipelines API: Trigger a pipeline for multiple images and multiple models", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPMultiModelRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/activate`, {}, constant.params)

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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_outputs.length == 2`]: (r) => r.json().model_outputs.length === 2,
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
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_outputs.length == 2`]: (r) => r.json().model_outputs.length === 2,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerWithDependency() {

  group("Pipelines API: Trigger a pipeline with dependency setting", () => {

    var reqHTTP = {
      id: randomString(10),
      description: randomString(50),
      recipe: {
        version: "v1alpha",
        components: [
          {
            id: "s01",
            resource_name: "connectors/start-operator",
            dependencies: {},
          },
          {
            id: "d01",
            resource_name: "connectors/end-operator",
            dependencies: {
              images: "[*s01.images]",
            },
          },
        ],
      },
    }

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/pipelines response status is 201 (HTTP pipeline)": (r) => r.status === 201,
    });

    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url)  response outputs.length = ${payloadImageURL["inputs"].length}`]:
        (r) => r.json().outputs.length === payloadImageURL["inputs"].length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url)  response outputs[0].images.length = ${payloadImageURL["inputs"][0].images.length}`]:
        (r) => r.json().outputs[0].images.length === payloadImageURL["inputs"][0].images.length,
    });


    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

    var reqHTTP = {
      id: randomString(10),
      description: randomString(50),
      recipe: {
        version: "v1alpha",
        components: [
          {
            id: "s01",
            resource_name: "connectors/start-operator",
            dependencies: {},
          },
          {
            id: "d01",
            resource_name: "connectors/end-operator",
            dependencies: {
              texts: "[*s01.texts]",
              images: "[]",
            },
          },
        ],
      },
    }

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), constant.params), {
      "POST /v1alpha/pipelines response status is 201 (HTTP pipeline)": (r) => r.status === 201,
    });

    http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/activate`, {}, constant.params)

    var payloadImageURL = {
      inputs: [
        {
          images: [{
            url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }],
          texts: ["11", "22"]
        },
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url)  response outputs.length = ${payloadImageURL["inputs"].length}`]:
        (r) => r.json().outputs.length === payloadImageURL["inputs"].length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url)  response outputs[0].texts.length = ${payloadImageURL["inputs"][0].texts.length}`]:
        (r) => r.json().outputs[0].texts.length === payloadImageURL["inputs"][0].texts.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url)  response outputs[0].images.length = 0`]:
        (r) => r.json().outputs[0].images.length === 0,
    });


    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });


  });

}
