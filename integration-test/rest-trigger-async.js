import http from "k6/http";
import encoding from "k6/encoding";

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

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageBase64.task_inputs.length,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg), "dog.jpg");
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === fd.parts.length,
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

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        },
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        }, {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageBase64.task_inputs.length,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg), "dog.jpg");
    fd.append("file", http.file(constant.catImg, "cat.jpg"));
    fd.append("file", http.file(constant.bearImg), "bear.jpg");
    fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === fd.parts.length,
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

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }, {
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageURL), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (url) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageURL.task_inputs.length,
    });

    var payloadImageBase64 = {
      task_inputs: [
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        },
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        },
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          }
        }
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async`, JSON.stringify(payloadImageBase64), constant.params), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (base64) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageBase64.task_inputs.length,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    fd.append("file", http.file(constant.catImg, "cat.jpg"));
    fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}/trigger-async-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqBody.id}/trigger-async (multipart) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === fd.parts.length,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqBody.id}`, null, constant.params), {
      [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
