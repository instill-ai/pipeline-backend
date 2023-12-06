import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";
import * as helper from "./helper.js";

export function CheckCreate(header) {
  group("Pipelines API: Create a pipeline", () => {

    var reqBody = Object.assign(
      {
        id: randomString(32),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var resOrigin = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      header
    );
    check(resOrigin, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline name": (r) =>
        r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline uid": (r) =>
        helper.isUUID(r.json().pipeline.uid),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline id": (r) =>
        r.json().pipeline.id === reqBody.id,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline description": (r) =>
        r.json().pipeline.description === reqBody.description,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline recipe is valid": (r) =>
        helper.validateRecipe(r.json().pipeline.recipe, false),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline user is UUID": (r) =>
        helper.isValidOwner(r.json().pipeline.user),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline create_time": (r) =>
        new Date(r.json().pipeline.create_time).getTime() >
        new Date().setTime(0),
      "POST /v1beta/${constant.namespace}/pipelines response pipeline update_time": (r) =>
        new Date(r.json().pipeline.update_time).getTime() >
        new Date().setTime(0),
    });


    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify({}),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines request body JSON Schema failed status 400": (
          r
        ) => r.status === 400,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines re-create the same id response status is 409":
          (r) => r.status === 409,
      }
    );

    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines re-create the same id after deletion response status is 201":
          (r) => r.status === 201,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify({}),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with empty body response status is 400": (r) =>
          r.status === 400,
      }
    );

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with null body response status is 400": (r) =>
          r.status === 400,
      }
    );

    reqBody.id = null;
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with null id response status is 400": (r) =>
          r.status === 400,
      }
    );

    reqBody.id = "abcd?*&efg!";
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with non-RFC-1034 naming id response status is 400":
          (r) => r.status === 400,
      }
    );

    reqBody.id = randomString(40);
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with > 32-character id response status is 400":
          (r) => r.status === 400,
      }
    );

    reqBody.id = "🧡💜我愛潤物科技💚💙";
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines with non-ASCII id response status is 400": (
          r
        ) => r.status === 400,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${resOrigin.json().pipeline.id
        }`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${resOrigin.json().pipeline.id
          } response status 204`]: (r) => r.status === 204,
      }
    );
  });
}



export function CheckList(header) {
  group("Pipelines API: List pipelines", () => {
    check(http.request("GET", `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`, null, header), {
      [`GET /v1beta/${constant.namespace}/pipelines response status is 200`]: (r) =>
        r.status === 200,
      [`GET /v1beta/${constant.namespace}/pipelines response next_page_token is empty`]: (r) =>
        r.json().next_page_token === "",
      [`GET /v1beta/${constant.namespace}/pipelines response total_size is 0`]: (r) =>
        r.json().total_size == 0,
    });

    const numPipelines = 200;
    var reqBodies = [];
    for (var i = 0; i < numPipelines; i++) {
      reqBodies[i] = Object.assign(
        {
          id: randomString(10),
          description: randomString(50),
        },
        constant.simpleRecipeWithoutCSV
      );
    }

    // Create pipelines
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "POST",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
          JSON.stringify(reqBody),
          header
        ),
        {
          [`POST /v1beta/${constant.namespace}/pipelines x${reqBodies.length} response status is 201`]:
            (r) => r.status === 201,
        }
      );
    }

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines.length == 10`]: (r) =>
          r.json().pipelines.length == 10,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines[0].recipe is null`]: (r) =>
          r.json().pipelines[0].recipe === null,
        [`GET /v1beta/${constant.namespace}/pipelines response total_size == 200`]: (r) =>
          r.json().total_size == 200,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?view=VIEW_FULL`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_FULL response pipelines[0] has recipe`]:
          (r) => r.json().pipelines[0].recipe !== null,
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_FULL response pipelines[0] recipe is valid`]:
          (r) => helper.validateRecipe(r.json().pipelines[0].recipe, false),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?view=VIEW_BASIC`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?view=VIEW_BASIC response pipelines[0].recipe is null`]:
          (r) => r.json().pipelines[0].recipe === null,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?page_size=3`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?page_size=3 response pipelines.length == 3`]: (
          r
        ) => r.json().pipelines.length == 3,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?page_size=101`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?page_size=101 response pipelines.length == 100`]:
          (r) => r.json().pipelines.length == 100,
      }
    );

    var resFirst100 = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?page_size=100`,
      null,
      header
    );
    var resSecond100 = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
      }`,
      null, header
    );
    check(resSecond100, {
      [`GET /v1beta/${constant.namespace}/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response status 200`]: (r) => r.status == 200,
      [`GET /v1beta/${constant.namespace}/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response return 100 results`]: (r) => r.json().pipelines.length == 100,
      [`GET /v1beta/${constant.namespace}/pipelines?page_size=100&page_token=${resFirst100.json().next_page_token
        } response next_page_token is empty`]: (r) =>
          r.json().next_page_token === "",
    });

    // Filtering
    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines response 200`]: (r) => r.status == 200,
        [`GET /v1beta/${constant.namespace}/pipelines response pipelines.length > 0`]: (r) =>
          r.json().pipelines.length > 0,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?filter=create_time>timestamp%28%222000-06-19T23:31:08.657Z%22%29`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?filter=create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response 200`]:
          (r) => r.status == 200,
        [`GET /v1beta/${constant.namespace}/pipelines?filter=create_time%20>%20timestamp%28%222000-06-19T23:31:08.657Z%22%29 response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    var srcConnPermalink = "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3"

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines?filter=recipe.components.definition_name:%22${srcConnPermalink}%22`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines?filter=recipe.components.definition_name:%22${srcConnPermalink}%22 response 200`]:
          (r) => r.status == 200,
        [`GET /v1beta/${constant.namespace}/pipelines?filter=recipe.components.definition_name:%22${srcConnPermalink}%22 response pipelines.length > 0`]:
          (r) => r.json().pipelines.length > 0,
      }
    );

    // Delete the pipelines
    for (const reqBody of reqBodies) {
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
          JSON.stringify(reqBody),
          header
        ),
        {
          [`DELETE /v1beta/${constant.namespace}/pipelines x${reqBodies.length} response status is 204`]:
            (r) => r.status === 204,
        }
      );
    }
  });
}

export function CheckGet(header) {
  group("Pipelines API: Get a pipeline", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
        description: randomString(50),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        header
      ),
      {
        "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
          r.status === 201,
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline name`]: (r) =>
          r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline uid`]: (r) =>
          helper.isUUID(r.json().pipeline.uid),
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline id`]: (r) =>
          r.json().pipeline.id === reqBody.id,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline description`]:
          (r) => r.json().pipeline.description === reqBody.description,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline recipe is null`]:
          (r) => r.json().pipeline.recipe === null,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline user is UUID`]:
          (r) => helper.isValidOwner(r.json().pipeline.user),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}?view=VIEW_FULL`,
        null,
        header
      ),
      {
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (r) =>
          r.status === 200,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline recipe is not null`]:
          (r) => r.json().pipeline.recipe !== null,
        [`GET /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline user is UUID`]:
          (r) => helper.isValidOwner(r.json().pipeline.user),
      }
    );

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/this-id-does-not-exist`,
        null,
        header
      ),
      {
        "GET /v1beta/${constant.namespace}/pipelines/this-id-does-not-exist response status is 404":
          (r) => r.status === 404,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

export function CheckUpdate(header) {
  group("Pipelines API: Update a pipeline", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var resOrigin = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      header
    );

    check(resOrigin, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
    });

    var reqBodyUpdate = Object.assign({
      uid: "output-only-to-be-ignored",
      name: "pipelines/some-string-to-be-ignored",
      description: randomString(50),
    });

    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status is 200`]: (
          r
        ) => r.status === 200,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline name (OUTPUT_ONLY)`]:
          (r) =>
            r.json().pipeline.name ===
            `${constant.namespace}/pipelines/${resOrigin.json().pipeline.id}`,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline uid (OUTPUT_ONLY)`]:
          (r) => r.json().pipeline.uid === resOrigin.json().pipeline.uid,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline id (IMMUTABLE)`]:
          (r) => r.json().pipeline.id === resOrigin.json().pipeline.id,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline state (OUTPUT_ONLY)`]:
          (r) => r.json().pipeline.state === resOrigin.json().pipeline.state,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline description (OPTIONAL)`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline user is UUID`]:
          (r) => helper.isValidOwner(r.json().pipeline.user),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline create_time (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.create_time).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline update_time (OUTPUT_ONLY)`]:
          (r) =>
            new Date(r.json().pipeline.update_time).getTime() >
            new Date().setTime(0),
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline update_time > create_time`]:
          (r) =>
            new Date(r.json().pipeline.update_time).getTime() >
            new Date(r.json().pipeline.create_time).getTime(),
      }
    );

    reqBodyUpdate.description = "";
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.description = randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response pipeline non-empty description`]:
          (r) => r.json().pipeline.description === reqBodyUpdate.description,
      }
    );

    reqBodyUpdate.id = randomString(10);
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status for updating IMMUTABLE field with different id is 400`]:
          (r) => r.status === 400,
      }
    );

    reqBodyUpdate.id = reqBody.id;
    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        [`PATCH /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status for updating IMMUTABLE field with the same id is 200`]:
          (r) => r.status === 200,
      }
    );

    check(
      http.request(
        "PATCH",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/this-id-does-not-exist`,
        JSON.stringify(reqBodyUpdate),
        header
      ),
      {
        "PATCH /v1beta/${constant.namespace}/pipelines/this-id-does-not-exist response status is 404":
          (r) => r.status === 404,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}


export function CheckRename(header) {
  group("Pipelines API: Rename a pipeline", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var res = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      header
    );

    check(res, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
      "POST /v1beta/${constant.namespace}/pipelines response pipeline name": (r) =>
        r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
    });

    reqBody.new_pipeline_id = randomString(10);

    check(
      http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
        }/rename`,
        JSON.stringify(reqBody),
        header
      ),
      {
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response status is 200"`]: (r) => r.status === 200,
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response pipeline new name"`]: (r) =>
            r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        [`POST /v1beta/${constant.namespace}/pipelines/${res.json().pipeline.id
          }/rename response pipeline new id"`]: (r) =>
            r.json().pipeline.id === reqBody.new_pipeline_id,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.new_pipeline_id}`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.new_pipeline_id} response status 204`]:
          (r) => r.status === 204,
      }
    );
  });
}

export function CheckLookUp(header) {
  group("Pipelines API: Look up a pipeline by uid", () => {
    var reqBody = Object.assign(
      {
        id: randomString(10),
      },
      constant.simpleRecipe
    );

    // Create a pipeline
    var res = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      header
    );

    check(res, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) => r.status === 201,
    });

    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/pipelines/${res.json().pipeline.uid
        }/lookUp`,
        null,
        header
      ),
      {
        [`GET /v1beta/pipelines/${res.json().pipeline.uid
          }/lookUp response status is 200"`]: (r) => r.status === 200,
        [`GET /v1beta/pipelines/${res.json().pipeline.uid
          }/lookUp response pipeline new name"`]: (r) =>
            r.json().pipeline.name === `${constant.namespace}/pipelines/${reqBody.id}`,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${reqBody.id}`,
        null,
        header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/${reqBody.id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}
