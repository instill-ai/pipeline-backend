import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
      state: "STATE_ACTIVE",
    },
    constant.detectionRecipe
  )

  group("Pipelines API: Create a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
      "POST /pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
      "POST /pipelines response pipeline uid": (r) => helper.isUUID(r.json().pipeline.uid),
      "POST /pipelines response pipeline id": (r) => r.json().pipeline.id === reqBody.id,
      "POST /pipelines response pipeline description": (r) => r.json().pipeline.description === reqBody.description,
      "POST /pipelines response pipeline recipe": (r) => r.json().pipeline.recipe !== undefined,
      "POST /pipelines response pipeline state": (r) => r.json().pipeline.state === "STATE_ACTIVE",
      "POST /pipelines response pipeline mode": (r) => r.json().pipeline.mode === "MODE_SYNC",
      "POST /pipelines response pipeline create_time": (r) => new Date(r.json().pipeline.create_time).getTime() > new Date().setTime(0),
      "POST /pipelines response pipeline update_time": (r) => new Date(r.json().pipeline.update_time).getTime() > new Date().setTime(0)
    });

    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines re-create the same id response status is 400": (r) => r.status === 400
    });

    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines re-create the same id after deletion response status is 201": (r) => r.status === 201
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
    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckList() {

  var reqBodies = [];
  for (var i = 0; i < constant.numPipelines; i++) {
    reqBodies[i] = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
        state: "STATE_ACTIVE",
      },
      constant.detectionRecipe
    )
  }

  group("Pipelines API: List pipelines", () => {

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`POST /pipelines x${reqBodies.length} response status is 201`]: (r) => r.status === 201
      });
    }

    check(http.request("GET", `${pipelineHost}/pipelines`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines response pipelines.length == ${reqBodies.length}`]: (r) => r.json().pipelines.length == reqBodies.length,
      [`GET /pipelines response pipelines[0] no recipe`]: (r) => r.json().pipelines[0].recipe === null,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=VIEW_FULL`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=VIEW_FULL response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines?view=VIEW_FULL response pipelines.length == ${reqBodies.length}`]: (r) => r.json().pipelines.length == reqBodies.length,
      [`GET /pipelines?view=VIEW_FULL response pipelines[0] has recipe`]: (r) => r.json().pipelines[0].recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=VIEW_BASIC`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=VIEW_BASIC response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines?view=VIEW_BASIC response pipelines.length == ${reqBodies.length}`]: (r) => r.json().pipelines.length == reqBodies.length,
      [`GET /pipelines?view=VIEW_BASIC response pipelines[0] has no recipe`]: (r) => r.json().pipelines[0].recipe === null,
    });

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(http.request(
        "DELETE",
        `${pipelineHost}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBody), {
        headers: {
          "Content-Type": "application/json",
        },
      }), {
        [`DELETE /pipelines x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}

export function CheckGet() {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
      state: "STATE_ACTIVE",
    },
    constant.detectionRecipe
  )

  group("Pipelines API: Get a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
    });

    check(http.request("GET", `${pipelineHost}/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`GET /pipelines/${reqBody.id} response status is 200`]: (r) => r.status === 200,
      [`GET /pipelines/${reqBody.id} response pipeline name`]: (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
      [`GET /pipelines/${reqBody.id} response pipeline uid`]: (r) => helper.isUUID(r.json().pipeline.uid),
      [`GET /pipelines/${reqBody.id} response pipeline id`]: (r) => r.json().pipeline.id === reqBody.id,
      [`GET /pipelines/${reqBody.id} response pipeline description`]: (r) => r.json().pipeline.description === reqBody.description,
      [`GET /pipelines/${reqBody.id} response pipeline recipe`]: (r) => r.json().pipeline.recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines/this-id-does-not-exist`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "GET /pipelines/this-id-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${pipeline.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdate() {

  var reqBody = Object.assign(
    {
      id: randomString(10),
      description: randomString(50),
      state: "STATE_ACTIVE",
    },
    constant.detectionRecipe
  )

  group("Pipelines API: Update a pipeline", () => {

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
    });

    var reqBodyUpdate = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
        state: "STATE_INACTIVE",
      },
    )

    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response status is 200`]: (r) => r.status === 200,
      [`PATCH /pipelines/${reqBody.id} response pipeline uid`]: (r) => helper.isUUID(r.json().pipeline.uid),
      [`PATCH /pipelines/${reqBody.id} response pipeline id`]: (r) => r.json().pipeline.id === reqBodyUpdate.id,
      [`PATCH /pipelines/${reqBody.id} response pipeline description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
      [`PATCH /pipelines/${reqBody.id} response pipeline recipe`]: (r) => r.json().pipeline.recipe !== undefined,
      [`PATCH /pipelines/${reqBody.id} response pipeline state`]: (r) => r.json().pipeline.state === "STATE_INACTIVE",
      [`PATCH /pipelines/${reqBody.id} response pipeline mode`]: (r) => r.json().pipeline.mode === "MODE_SYNC",
      [`PATCH /pipelines/${reqBody.id} response pipeline name`]: (r) => r.json().pipeline.name === `pipelines/${reqBodyUpdate.id}`,
      [`PATCH /pipelines/${reqBody.id} response pipeline create_time`]: (r) => new Date(r.json().pipeline.create_time).getTime() > new Date().setTime(0),
      [`PATCH /pipelines/${reqBody.id} response pipeline update_time`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date().setTime(0),
      [`PATCH /pipelines/${reqBody.id} response pipeline update_time > create_time`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date(r.json().pipeline.create_time).getTime()
    });

    reqBodyUpdate.description = ""

    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBodyUpdate.id}`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBodyUpdate.id} response status is 200`]: (r) => r.status === 200,
      [`PATCH /pipelines/${reqBodyUpdate.id} response pipeline description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
    });

    check(http.request("PATCH", `${pipelineHost}/pipelines/this-id-does-not-exist`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "PATCH /pipelines/this-id-does-not-exist response status is 404": (r) => r.status === 404,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBodyUpdate.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${reqBodyUpdate.id} response status 204`]: (r) => r.status === 204,
    });

  });
}
