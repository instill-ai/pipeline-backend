import http from "k6/http";
import encoding from "k6/encoding";

import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckTriggerImageDirect() {

  var reqBody = Object.assign(
    {
      display_name: randomString(10),
      description: randomString(50),
      status: "STATUS_ACTIVATED",
    },
    constant.detectionRecipe
  );

  group("Pipelines API: Trigger a pipeline", () => {

    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
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
        },
        {
          image_url: "https://artifacts.instill.tech/dog.jpg",
        },
      ],
    };

    check(http.request("POST", `${pipelineHost}/pipelines/${reqBody.display_name}/outputs`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response status is 200`]: (r) => r.status === 200,
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response output.detection_outputs.length`]: (r) => r.json().output.detection_outputs.length === payloadImageURL.inputs.length,
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response output.detection_outputs[0].bounding_box_objects.length`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects.length === 1,
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].score === 1,
      [`POST /pipelines/${reqBody.display_name}/outputs (url) response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
    });

    var payloadImageBase64 = {
      inputs: [
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
        {
          imageBase64: encoding.b64encode(constant.dogImg, "b"),
        },
      ],
    };

    check(http.request("POST", `${pipelineHost}/pipelines/${reqBody.display_name}/outputs`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response output.detection_outputs.length`]: (r) => r.json().output.detection_outputs.length === payloadImageBase64.inputs.length,
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response output.detection_outputs[0].bounding_box_objects.length`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects.length === 1,
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].score === 1,
      [`POST /pipelines/${reqBody.display_name}/outputs (base64) response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg));
    fd.append("file", http.file(constant.dogImg));
    fd.append("file", http.file(constant.dogImg));
    check(http.request("POST", `${pipelineHost}/pipelines/${reqBody.display_name}/upload/outputs`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /pipelines/${reqBody.display_name}/outputs (multipart) response status is 200`]: (r) => r.status === 200,
      [`POST /pipelines/${reqBody.display_name}/outputs (multipart) response output.detection_outputs.length`]: (r) => r.json().output.detection_outputs.length === fd.parts.length,
      [`POST /pipelines/${reqBody.display_name}/outputs (multipart) response output.detection_outputs[0].bounding_box_objects.length`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects.length === 1,
      [`POST /pipelines/${reqBody.display_name}/outputs (multipart) response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) => r.json().output.detection_outputs[0].bounding_box_objects[0].score === 1,
    });

  });

  // Delete the pipeline
  check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.display_name}`, null, {
    headers: {
      "Content-Type": "application/json",
    },
  }), {
    [`DELETE /pipelines/${reqBody.display_name} response status 204`]: (r) => r.status === 204,
  });
}
