import http from "k6/http";
import encoding from "k6/encoding";

import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js"

export function CheckTriggerSyncSingleImageSingleModelInst() {

  group("Pipelines API: Trigger a pipeline for single image and single model instance", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201 (HTTP pipeline)": (r) => r.status === 201,
    });

    var payloadImageURL = {
      task_inputs: [{
        detection: {
          image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === payloadImageURL.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageURL.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        }
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === payloadImageBase64.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageBase64.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === fd.parts.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === fd.parts.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    const fdWrong = new FormData();
    fdWrong.append("file", "some fake binary string that won't work for sure");
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fdWrong.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response status is 422 with wrong request file`]: (r) => r.status === 422,
    });

    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

    var reqGRPC = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncGRPCSingleModelInstRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqGRPC), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201 (gRPC pipeline)": (r) => r.status === 201,
    });

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqGRPC.id}/trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqGRPC.id}/trigger (url) response status is 400 (gRPC pipeline triggered by HTTP)`]: (r) => r.status === 422,
    })

    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqGRPC.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqGRPC.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerSyncMultiImageSingleModelInst() {

  group("Pipelines API: Trigger a pipeline for multiple images and single model instance", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPSingleModelInstRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var payloadImageURL = {
      task_inputs: [
        {
          detection: {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }, {
          detection:
          {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
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

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response output[0].detection_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === payloadImageURL.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageURL.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    var payloadImageBase64 = {
      task_inputs: [
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          },
        },
        {
          detection: {
            image_base64: encoding.b64encode(constant.dogImg, "b"),
          },
        }
      ]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response output[0].detection_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === payloadImageBase64.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === payloadImageBase64.task_inputs.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    fd.append("file", http.file(constant.catImg, "cat.jpg"));
    fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response output[0].detection_outputs.length`]: (r) => r.json().model_instance_outputs[0].task_outputs.length === fd.parts.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response data_mapping_indices.length`]: (r) => r.json().data_mapping_indices.length === fd.parts.length,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task`]: (r) => r.json().model_instance_outputs[0].task === "TASK_DETECTION",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].model_instance`]: (r) => r.json().model_instance_outputs[0].model_instance === constant.detSyncHTTPSingleModelInstRecipe.recipe.model_instances[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects.length`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects.length === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].index == data_mapping_indices[0]`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].index === r.json().data_mapping_indices[0],
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].category`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].category === "test",
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].score`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].score === 1,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart response model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box`]: (r) => r.json().model_instance_outputs[0].task_outputs[0].detection.objects[0].bounding_box !== undefined,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}

export function CheckTriggerSyncMultiImageMultiModelInst() {

  group("Pipelines API: Trigger a pipeline for multiple images and multiple model instances", () => {

    var reqHTTP = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.detSyncHTTPMultiModelInstRecipe
    );

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`, JSON.stringify(reqHTTP), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
    });

    var payloadImageURL = {
      task_inputs: [
        {
          detection:
          {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }, {
          detection:
          {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }, {
          detection:
          {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }, {
          detection:
          {
            image_url: "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageURL), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (url) response model_instance_outputs.length == 2`]: (r) => r.json().model_instance_outputs.length === 2,
    });

    var payloadImageBase64 = {
      task_inputs: [{
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      }, {
        detection: {
          image_base64: encoding.b64encode(constant.dogImg, "b"),
        },
      }]
    };

    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger`, JSON.stringify(payloadImageBase64), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger (base64) response model_instance_outputs.length == 2`]: (r) => r.json().model_instance_outputs.length === 2,
    });

    const fd = new FormData();
    fd.append("file", http.file(constant.dogImg, "dog.jpg"));
    fd.append("file", http.file(constant.catImg, "cat.jpg"));
    fd.append("file", http.file(constant.bearImg, "bear.jpg"));
    fd.append("file", http.file(constant.dogRGBAImg, "dog-rgba.png"));
    check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}/trigger-multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`,
      },
    }), {
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart (multipart) response status is 200`]: (r) => r.status === 200,
      [`POST /v1alpha/pipelines/${reqHTTP.id}/trigger-multipart (multipart) response model_instance_outputs.length == 2`]: (r) => r.json().model_instance_outputs.length === 2,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${reqHTTP.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /v1alpha/pipelines/${reqHTTP.id} response status 204`]: (r) => r.status === 204,
    });

  });

}
