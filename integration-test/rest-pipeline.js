import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

  group("Pipelines API: Create a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(63),
        description: randomString(50),
        state: "STATE_ACTIVE",
      },
      constant.detectionRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })
    check(resOrigin, {
      "POST /pipelines response status is 201": (r) => r.status === 201,
      "POST /pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
      "POST /pipelines response pipeline uid": (r) => helper.isUUID(r.json().pipeline.uid),
      "POST /pipelines response pipeline id": (r) => r.json().pipeline.id === reqBody.id,
      "POST /pipelines response pipeline description": (r) => r.json().pipeline.description === reqBody.description,
      "POST /pipelines response pipeline recipe": (r) => r.json().pipeline.recipe !== undefined,
      "POST /pipelines response pipeline state": (r) => r.json().pipeline.state === "STATE_UNSPECIFIED",
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

    reqBody.id = null
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with null id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = "abcd?*&efg!"
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with non-RFC-1034 naming id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = randomString(64)
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with > 63-character id response status is 400": (r) => r.status === 400,
    });

    reqBody.id = "🧡💜我愛潤物科技💚💙"
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines with non-ASCII id response status is 400": (r) => r.status === 400,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${resOrigin.json().pipeline.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${resOrigin.json().pipeline.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckList() {

  group("Pipelines API: List pipelines", () => {

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
      [`GET /pipelines response pipelines.length == 10`]: (r) => r.json().pipelines.length == 10,
      [`GET /pipelines response pipelines[0] no recipe`]: (r) => r.json().pipelines[0].recipe === null,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=VIEW_FULL`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=VIEW_FULL response pipelines[0] has recipe`]: (r) => r.json().pipelines[0].recipe !== undefined,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?view=VIEW_BASIC`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?view=VIEW_BASIC response pipelines[0] has no recipe`]: (r) => r.json().pipelines[0].recipe === null,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?page_size=3`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?page_size=3 response pipelines.length == 3`]: (r) => r.json().pipelines.length == 3,
    });

    check(http.request("GET", `${pipelineHost}/pipelines?page_size=101`, null, {
      headers: {
        "Content-Type": "application/json",
      }
    }), {
      [`GET /pipelines?page_size=101 response pipelines.length == 100`]: (r) => r.json().pipelines.length == 100,
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

  group("Pipelines API: Get a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
        state: "STATE_ACTIVE",
      },
      constant.detectionRecipe
    )

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

  group("Pipelines API: Update a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        mode: "MODE_ASYNC",
        state: "STATE_INACTIVE",
      },
      constant.detectionRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })

    check(resOrigin, {
      "POST /pipelines response status is 201": (r) => r.status === 201,
    });

    var detectionRecipeUpdate = constant.detectionRecipe
    detectionRecipeUpdate.recipe.source = "connectors/gRPC"
    detectionRecipeUpdate.recipe.destination = "connectors/gRPC"

    var reqBodyUpdate = Object.assign(
      {
        uid: "output-only-to-be-ignored",
        mode: "MODE_ASYNC",
        state: "STATE_ACTIVE",
        name: "pipelines/some-string-to-be-ignored",
        description: randomString(50),
      },
      detectionRecipeUpdate
    )

    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response status is 200`]: (r) => r.status === 200,
      [`PATCH /pipelines/${reqBody.id} response pipeline name (OUTPUT_ONLY)`]: (r) => r.json().pipeline.name === `pipelines/${resOrigin.json().pipeline.id}`,
      [`PATCH /pipelines/${reqBody.id} response pipeline uid (OUTPUT_ONLY)`]: (r) => r.json().pipeline.uid === resOrigin.json().pipeline.uid,
      [`PATCH /pipelines/${reqBody.id} response pipeline id (IMMUTABLE)`]: (r) => r.json().pipeline.id === resOrigin.json().pipeline.id,
      [`PATCH /pipelines/${reqBody.id} response pipeline mode (OUTPUT_ONLY)`]: (r) => r.json().pipeline.mode === resOrigin.json().pipeline.mode,
      [`PATCH /pipelines/${reqBody.id} response pipeline state (OUTPUT_ONLY)`]: (r) => r.json().pipeline.state === resOrigin.json().pipeline.state,
      [`PATCH /pipelines/${reqBody.id} response pipeline description (OPTIONAL)`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
      [`PATCH /pipelines/${reqBody.id} response pipeline recipe (REQUIRED)`]: (r) => helper.deepEqual(r.json().pipeline.recipe, reqBodyUpdate.recipe),
      [`PATCH /pipelines/${reqBody.id} response pipeline create_time (OUTPUT_ONLY)`]: (r) => new Date(r.json().pipeline.create_time).getTime() > new Date().setTime(0),
      [`PATCH /pipelines/${reqBody.id} response pipeline update_time (OUTPUT_ONLY)`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date().setTime(0),
      [`PATCH /pipelines/${reqBody.id} response pipeline update_time > create_time`]: (r) => new Date(r.json().pipeline.update_time).getTime() > new Date(r.json().pipeline.create_time).getTime()
    });

    reqBodyUpdate.description = ""
    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response pipeline empty description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
    });

    reqBodyUpdate.description = randomString(10)
    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`,
      JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response pipeline non-empty description`]: (r) => r.json().pipeline.description === reqBodyUpdate.description,
    });

    reqBodyUpdate.id = randomString(10)
    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response status for updating IMMUTABLE field with different id is 400`]: (r) => r.status === 400,
    });

    reqBodyUpdate.id = reqBody.id
    check(http.request("PATCH", `${pipelineHost}/pipelines/${reqBody.id}`, JSON.stringify(reqBodyUpdate), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`PATCH /pipelines/${reqBody.id} response status for updating IMMUTABLE field with the same id is 200`]: (r) => r.status === 200,
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
    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${reqBody.id} response status 204`]: (r) => r.status === 204,
    });

  });
}

export function CheckUpdateState() {

  group("Pipelines API: Update a pipeline state", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        mode: "MODE_ASYNC",
        state: "STATE_INACTIVE",
      },
      constant.detectionRecipe
    )

    // Create a pipeline
    check(http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      "POST /pipelines response status is 201": (r) => r.status === 201,
      "POST /pipelines response pipeline state UNSPECIFIED": (r) => r.json().pipeline.state === "STATE_UNSPECIFIED",
    });

    check(http.request("POST", `${pipelineHost}/pipelines/${reqBody.id}:activate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /pipelines/${reqBody.id}:rename response pipeline state ACTIVE"`]: (r) => r.json().pipeline.state === "STATE_ACTIVE",
    });

    check(http.request("POST", `${pipelineHost}/pipelines/${reqBody.id}:deactivate`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /pipelines/${reqBody.id}:rename response pipeline state INACTIVE"`]: (r) => r.json().pipeline.state === "STATE_INACTIVE",
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

export function CheckRename() {

  group("Pipelines API: Rename a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(10),
        mode: "MODE_ASYNC",
        state: "STATE_INACTIVE",
      },
      constant.detectionRecipe
    )

    // Create a pipeline
    var resOrigin = http.request("POST", `${pipelineHost}/pipelines`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    })

    check(resOrigin, {
      "POST /pipelines response status is 201": (r) => r.status === 201,
      "POST /pipelines response pipeline name": (r) => r.json().pipeline.name === `pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10)
    check(http.request("POST", `${pipelineHost}/pipelines/${resOrigin.json().pipeline.id}:rename`, JSON.stringify(reqBody), {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`POST /pipelines/${resOrigin.json().pipeline.id}:rename response status is 200"`]: (r) => r.status === 200,
      [`POST /pipelines/${resOrigin.json().pipeline.id}:rename response pipeline new name"`]: (r) => r.json().pipeline.name === `pipelines/${reqBody.new_pipeline_id}`,
      [`POST /pipelines/${resOrigin.json().pipeline.id}:rename response pipeline new id"`]: (r) => r.json().pipeline.id === reqBody.new_pipeline_id,
    });

    // Delete the pipeline
    check(http.request("DELETE", `${pipelineHost}/pipelines/${reqBody.new_pipeline_id}`, null, {
      headers: {
        "Content-Type": "application/json",
      },
    }), {
      [`DELETE /pipelines/${reqBody.new_pipeline_id} response status 204`]: (r) => r.status === 204,
    });

  });

}
