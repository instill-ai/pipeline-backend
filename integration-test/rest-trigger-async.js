import http from "k6/http";

import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { check, group } from "k6";
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
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg), "dog.jpg");
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (multipart) response status is 200`]: (r) => r.status === 200,
    // });

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
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg), "dog.jpg");
    // fd.append("file", http.file(constant.catImg, "cat.jpg"));
    // fd.append("file", http.file(constant.bearImg), "bear.jpg");
    // fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (multipart) response status is 200`]: (r) => r.status === 200,
    // });

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
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    fd.append("file", http.file(constant.catImg, "cat.jpg"));
    fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsyncMultipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (multipart) response status is 200`]: (r) => r.status === 200,
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
    });

    // const fd = new FormData();
    // fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    // fd.append("file", http.file(constant.catImg, "cat.jpg"));
    // fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    // fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    // check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/triggerAsyncMultipart`, fd.body(), {
    //   headers: {
    //     "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
    //   },
    // }), {
    //   [`POST /v1alpha/pipelines/${reqBody.id}/triggerAsync (multipart) response status is 200`]: (r) => r.status === 200,
    // });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
