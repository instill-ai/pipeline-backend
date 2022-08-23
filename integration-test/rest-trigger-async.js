import http from "k6/http";
import encoding from "k6/encoding";

import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelineHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTriggerAsyncSingleImageSingleModelInst() {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncSingleModelInstRecipe
  );

  group("Pipelines API: Trigger an async pipeline for single image and single model instance", () => {

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var payloadImageURL = {
      inputs: [
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (url) response status is 200`]: (r) => r.status === 200,
    });

    var payloadImageBase64 = {
      inputs: [
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (base64) response status is 200`]: (r) => r.status === 200,
    });

    const fd = new FormData();
    fd.append("file", {data: http.file(constant.dogImg), filename: "dog"});
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (multipart) response status is 200`]: (r) => r.status === 200,
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
    headers: {
      "Content-Type": "application/json",
    },
  }), {
    [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}

export function CheckTriggerAsyncMultiImageSingleModelInst() {
  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncSingleModelInstRecipe
  );

  group("Pipelines API: Trigger an async pipeline for multiple images and single model instance", () => {

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var payloadImageURL = {
      inputs: [
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        },
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        },
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (url) response status is 200`]: (r) => r.status === 200,
    });

    var payloadImageBase64 = {
      inputs: [
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (base64) response status is 200`]: (r) => r.status === 200,
    });

    const fd = new FormData();
    fd.append("file", {data: http.file(constant.dogImg), filename: "dog"});
    fd.append("file", {data: http.file(constant.catImg), filename: "cat"});
    fd.append("file", {data: http.file(constant.bearImg), filename: "bear"});
    fd.append("file", {data: http.file(constant.dogRGBAImg), filename: "dogRGBA"});
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (multipart) response status is 200`]: (r) => r.status === 200,
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
    headers: {
      "Content-Type": "application/json",
    },
  }), {
    [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}

export function CheckTriggerAsyncMultiImageMultiModelInst() {
  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
    },
    constant.detAsyncMultiModelInstRecipe
  );

  group("Pipelines API: Trigger an async pipeline for multiple images and multiple model instances", () => {

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var payloadImageURL = {
      inputs: [
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        },
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        },
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (url) response status is 200`]: (r) => r.status === 200,
    });

    var payloadImageBase64 = {
      inputs: [
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        }
      ]
    };

    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (base64) response status is 200`]: (r) => r.status === 200,
    });

    const fd = new FormData();
    fd.append("file", {data: http.file(constant.dogImg), filename: "dog"});
    fd.append("file", {data: http.file(constant.catImg), filename: "cat"});
    fd.append("file", {data: http.file(constant.bearImg), filename: "bear"});
    fd.append("file", {data: http.file(constant.dogRGBAImg), filename: "dogRGBA"});
    check(http.request("POST", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}:trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqBody.id}:trigger (multipart) response status is 200`]: (r) => r.status === 200,
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${reqBody.id}`, null, {
    headers: {
      "Content-Type": "application/json",
    },
  }), {
    [`DELETE /v1alpha/pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
  });
}
