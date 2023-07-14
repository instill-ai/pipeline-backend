import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTriggerSyncSingleImageSingleModel() {

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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggersync (url) response status is 200`]: (r) => r.status === 200,
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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (base64) response status is 200`]: (r) => r.status === 200,
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart response status is 200`]: (r) => r.status === 200,
    // });

    // const fdWrong = new FormData();
    // fdWrong.append("file", "some fake binary string that won't work for sure");
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fdWrong.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart response status is 422 with wrong request file`]: (r) => r.status === 422,
    // });

    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerSyncMultiImageSingleModel() {

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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (url) response status is 200`]: (r) => r.status === 200,
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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (base64) response status is 200`]: (r) => r.status === 200,
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    // fd.append("file", http.file(constant.catImg, "cat.jpg"));
    // fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    // fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart response status is 200`]: (r) => r.status === 200,
    // });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerSyncMultiImageMultiModel() {

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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (url) response model_outputs.length == 2`]: (r) => r.json().model_outputs.length === 2,
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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSync`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSync (base64) response model_outputs.length == 2`]: (r) => r.json().model_outputs.length === 2,
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    // fd.append("file", http.file(constant.catImg, "cat.jpg"));
    // fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    // fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart (multipart) response status is 200`]: (r) => r.status === 200,
    //   [`POST /v1alpha/pipelines/${reqHTTP.id}/triggerSyncMultipart (multipart) response model_outputs.length == 2`]: (r) => r.json().model_outputs.length === 2,
    // });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
