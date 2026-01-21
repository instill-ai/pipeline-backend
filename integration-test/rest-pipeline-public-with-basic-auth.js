import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";

export function CheckCreate(data) {
  group(
    `Pipelines API: Create a pipeline [with invalid auth]`,
    () => {
      var reqBody = Object.assign(
        {
          description: randomString(50),
        },
        constant.simplePipelineWithYAMLRecipe
      );

      // Cannot create a pipeline of a non-exist user
      check(
        http.request(
          "POST",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
          JSON.stringify(reqBody),
          constant.paramsHTTPWithInvalidAuth
        ),
        {
          [`[with invalid auth] POST /v1beta/${constant.namespace}/pipelines response status is 401`]:
            (r) => r.status === 401,
        }
      );
    }
  );
}

export function CheckList(data) {
  group(`Pipelines API: List pipelines [with invalid auth]`, () => {
    // Cannot list pipelines with invalid auth
    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        null,
        constant.paramsHTTPWithInvalidAuth
      ),
      {
        [`[with invalid auth] GET /v1beta/${constant.namespace}/pipelines response status is 401`]:
          (r) => r.status === 401,
      }
    );
  });
}

export function CheckGet(data) {
  group(`Pipelines API: Get a pipeline [with invalid auth]`, () => {
    var reqBody = Object.assign(
      {
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    // Create a pipeline
    var createRes = http.request(
      "POST",
      `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
      JSON.stringify(reqBody),
      data.header
    );
    check(createRes, {
      "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
        r.status === 201,
    }
    );

    if (createRes.status !== 201 || !createRes.json().pipeline) {
      console.log(`Failed to create pipeline in CheckGet - status: ${createRes.status}, body: ${createRes.body}, skipping remaining tests`);
      return;
    }
    var pipelineId = createRes.json().pipeline.id;

    // Cannot get a pipeline with invalid auth
    check(
      http.request(
        "GET",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        null,
        constant.paramsHTTPWithInvalidAuth
      ),
      {
        [`[with invalid auth] GET /v1beta/${constant.namespace}/pipelines/{id} response status is 401`]:
          (r) => r.status === 401,
      }
    );

    // Delete the pipeline
    check(
      http.request(
        "DELETE",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
        null,
        data.header
      ),
      {
        [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}

export function CheckUpdate(data) {
  group(
    `Pipelines API: Update a pipeline [with invalid auth]`,
    () => {
      var reqBody = Object.assign(
        {},
        constant.simplePipelineWithYAMLRecipe
      );

      // Create a pipeline
      var resOrigin = http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      );

      check(resOrigin, {
        "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
          r.status === 201,
      });

      var pipelineId = resOrigin.json().pipeline.id;

      var reqBodyUpdate = Object.assign({
        name: "pipelines/some-string-to-be-ignored",
        description: randomString(50),
      });

      // Cannot update a pipeline of a non-exist user
      check(
        http.request(
          "PATCH",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
          JSON.stringify(reqBodyUpdate),
          constant.paramsHTTPWithInvalidAuth
        ),
        {
          [`[with invalid auth] PATCH /v1beta/${constant.namespace}/pipelines/{id} response status is 401`]:
            (r) => r.status === 401,
        }
      );

      // Delete the pipeline
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
          null,
          data.header
        ),
        {
          [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (
            r
          ) => r.status === 204,
        }
      );
    }
  );
}

export function CheckRename(data) {
  group(
    `Pipelines API: Rename a pipeline [with invalid auth]`,
    () => {
      var reqBody = Object.assign(
        {},
        constant.simplePipelineWithYAMLRecipe
      );

      // Create a pipeline
      var res = http.request(
        "POST",
        `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines`,
        JSON.stringify(reqBody),
        data.header
      );

      check(res, {
        "POST /v1beta/${constant.namespace}/pipelines response status is 201": (r) =>
          r.status === 201,
        "POST /v1beta/${constant.namespace}/pipelines response pipeline has name": (r) =>
          r.json().pipeline.name && r.json().pipeline.name.includes("/pipelines/"),
      });

      var pipelineId = res.json().pipeline.id;

      var renameBody = {
        new_pipeline_id: constant.dbIDPrefix + randomString(10),
      };

      // Cannot rename a pipeline of a non-exist user
      check(
        http.request(
          "POST",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}/rename`,
          JSON.stringify(renameBody),
          constant.paramsHTTPWithInvalidAuth
        ),
        {
          [`[with invalid auth] POST /v1beta/${constant.namespace}/pipelines/{id}/rename response status is 401`]: (r) => r.status === 401,
        }
      );

      // Delete the pipeline
      check(
        http.request(
          "DELETE",
          `${pipelinePublicHost}/v1beta/${constant.namespace}/pipelines/${pipelineId}`,
          null,
          data.header
        ),
        {
          [`DELETE /v1beta/${constant.namespace}/pipelines/{id} response status 204`]: (r) =>
            r.status === 204,
        }
      );
    }
  );
}
