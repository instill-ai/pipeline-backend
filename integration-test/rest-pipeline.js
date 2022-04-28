import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

  var pipeline = Object.assign(
    {
      name: randomString(10),
      description: randomString(50),
      status: "STATUS_ACTIVATED",
    },
    constant.detectionRecipe
  );

  group("Pipelines API: Create a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(pipeline), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
      "POST /pipelines response pipeline name": (r) => r.json().pipeline.name === pipeline.name,
      "POST /pipelines response pipeline description": (r) => r.json().pipeline.description === pipeline.description,
    });

    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify({}), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with empty body response status is 400": (r) => r.status === 400,
    });

    check(http.request("POST", `${pipelineHost}/pipelines`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with null body response status is 400": (r) => r.status === 400,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${pipeline.name}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${pipeline.name} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckList() {

  var pipelines = [];
  for (var i = 0; i < constant.numPipelines; i++) {
    pipelines[i] = Object.assign(
      {
        name: randomString(10),
        description: randomString(50),
        status: "STATUS_ACTIVATED",
      },
      constant.detectionRecipe
    );
  }

  group("Pipelines API: List pipelines", () => {

    // Create pipelines
    for (const pipeline of pipelines) {
      check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(pipeline), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`POST /pipelines x${pipelines.length} response status is 201`]: (r) => r.status === 201
      });
    }

    check(http.request("GET", `${pipelineHost}/pipelines`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines response pipelines.length == ${pipelines.length}`]: (r) => r.json().pipelines.length == pipelines.length,
      [`GET /pipelines response pipelines[0] no recipe`]: (r) => r.json().pipelines[0].recipe === undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=PIPELINE_VIEW_FULL`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=PIPELINE_VIEW_FULL response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines?view=PIPELINE_VIEW_FULL response pipelines.length == ${pipelines.length}`]: (r) => r.json().pipelines.length == pipelines.length,
      [`GET /pipelines?view=PIPELINE_VIEW_FULL response pipelines[0] has recipe`]: (r) => r.json().pipelines[0].recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=PIPELINE_VIEW_BASIC`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=PIPELINE_VIEW_BASIC response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines?view=PIPELINE_VIEW_BASIC response pipelines.length == ${pipelines.length}`]: (r) => r.json().pipelines.length == pipelines.length,
      [`GET /pipelines?view=PIPELINE_VIEW_BASIC response pipelines[0] has no recipe`]: (r) => r.json().pipelines[0].recipe === undefined,
    });

    // Delete the pipelines
    for (const pipeline of pipelines) {
      check(http.request(
        "DELETE",
        `${pipelineHost}/pipelines/${pipeline.name}`,
        JSON.stringify(pipeline), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`DELETE /pipelines x${pipelines.length} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}

export function CheckGet() {

  var pipeline = Object.assign(
    {
      name: randomString(10),
      description: randomString(50),
      status: "STATUS_ACTIVATED",
    },
    constant.detectionRecipe
  );

  group("Pipelines API: Get a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(pipeline), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
    });

    check(http.request("GET", `${pipelineHost}/pipelines/${pipeline.name}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`GET /pipelines/${pipeline.name} response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines/${pipeline.name} response pipeline name`]: (r) => r.json().pipeline.name === pipeline.name,
      [`GET /pipelines/${pipeline.name} response pipeline description`]: (r) => r.json().pipeline.description === pipeline.description,
      [`GET /pipelines/${pipeline.name} response pipeline id`]: (r) => helper.isUUID(r.json().pipeline.id),
      [`GET /pipelines/${pipeline.name} response pipeline recipe`]: (r) => r.json().pipeline.recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines/this-name-does-not-exist`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "GET /pipelines/this-name-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${pipeline.name}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${pipeline.name} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdate() {

  var pipeline = Object.assign(
    {
      name: randomString(10),
      description: randomString(50),
      status: "STATUS_ACTIVATED",
    },
    constant.detectionRecipe
  );

  var updatedPipeline = Object.assign(
    {
      name: randomString(10),
      description: randomString(50),
      status: "STATUS_INACTIVATED",
    },
  );

  group("Pipelines API: Update a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(pipeline), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
    });

    check(
      http.request("PATCH", `${pipelineHost}/pipelines/${pipeline.name}`,
        JSON.stringify(updatedPipeline), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
      [`PATCH /pipelines/${pipeline.name} response status is 200`]: (r) => r.status === 200,
      [`PATCH /pipelines/${pipeline.name} response pipeline name`]: (r) => r.json().pipeline.name === updatedPipeline.name,
      [`PATCH /pipelines/${pipeline.name} response pipeline description`]: (r) => r.json().pipeline.description === updatedPipeline.description,
      [`PATCH /pipelines/${pipeline.name} response pipeline id`]: (r) => helper.isUUID(r.json().pipeline.id),
      [`PATCH /pipelines/${pipeline.name} response pipeline recipe`]: (r) => r.json().pipeline.recipe !== undefined,
    });

    check(http.request("PATCH", `${pipelineHost}/pipelines/this-name-does-not-exist`,
      JSON.stringify(updatedPipeline), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "PATCH /pipelines/this-name-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${updatedPipeline.name}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${updatedPipeline.name} response status 204`]: (r) => r.status === 204,
    });

  });
}
