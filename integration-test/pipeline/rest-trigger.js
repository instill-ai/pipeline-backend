import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";

const recipeWithoutSetup = `
version: v1beta
variable:
  recipients:
    type: array:string
output:
  resp:
    title: Response
    value: \${email-0.output.result}
component:
  email-0:
    type: email
    input:
      recipients: \${variable.recipients}
      cc: null
      bcc: null
      subject: "Dummy email"
      message: "Hi I'm testing integrations"
    condition: null
    task: TASK_SEND_EMAIL
`;

var collectionPath = `/v1beta/namespaces/${constant.defaultUsername}/pipelines`;

function resourcePath(id) {
  return `${collectionPath}/${id}`;
}

function triggerPath(id) {
  return resourcePath(id) + "/trigger";
}

export function CheckTrigger(data) {
  group("Pipelines API: Trigger a pipeline", () => {
    var reqHTTP = Object.assign(
      {
        id: constant.dbIDPrefix + randomString(10),
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    check(http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqHTTP), data.header), {
      [`POST ${collectionPath} (${reqHTTP.id}) response status is 201 (HTTP pipeline)`]: (r) => r.status === 201,
    });

    check(http.request("POST", pipelinePublicHost + triggerPath(reqHTTP.id), JSON.stringify(constant.simplePayload), data.header), {
      [`POST ${triggerPath(reqHTTP.id)} response status is 200`]: (r) => r.status === 200,
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(reqHTTP.id), null, data.header), {
      [`DELETE ${resourcePath(reqHTTP.id)} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Pipelines API: Trigger a pipeline with YAML recipe", () => {
    var reqHTTP = Object.assign(
      {
        id: constant.dbIDPrefix + randomString(10),
        description: randomString(50),
      },
      constant.simplePipelineWithYAMLRecipe
    );

    check(http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqHTTP), data.header), {
      [`POST ${collectionPath} (${reqHTTP.id}) response status is 201 (HTTP pipeline)`]: (r) => r.status === 201,
    });

    check(http.request("POST", pipelinePublicHost + triggerPath(reqHTTP.id), JSON.stringify(constant.simplePayload), data.header), {
      [`POST ${triggerPath(reqHTTP.id)} response status is 200`]: (r) => r.status === 200,
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(reqHTTP.id), null, data.header), {
      [`DELETE ${resourcePath(reqHTTP.id)} response status 204`]: (r) => r.status === 204,
    });
  });

  group("Pipelines API: Validate pipeline on trigger", () => {
    const payload = {
      data: [{
        variable: {recipients: ["a", "b"]},
      }],
    };

    const missingConnRecipe = `${recipeWithoutSetup}
    setup: \${connection.my-conn}`;

    var reqMiss = {
      id: constant.dbIDPrefix + `missing-conn`,
      description: randomString(10),
      rawRecipe: missingConnRecipe,
    };

    check(http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqMiss), data.header), {
      [`POST ${collectionPath} (${reqMiss.id}) response status is 201`]: (r) => r.status === 201,
    });

    check(http.request("POST", pipelinePublicHost + triggerPath(reqMiss.id), JSON.stringify(payload), data.header), {
      [`POST ${triggerPath(reqMiss.id)} response status is 400`]: (r) => r.status === 400,
      [`POST ${triggerPath(reqMiss.id)} contains end-user message`]:
        (r) => r.json().message === "Connection my-conn doesn't exist.",
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(reqMiss.id), null, data.header), {
      [`DELETE ${resourcePath(reqMiss.id)} response status 204`]: (r) => r.status === 204,
    });

    const invalidRefRecipe = `${recipeWithoutSetup}
    setup: \${connnnnnection.my-conn}`;

    var reqInvalid = {
      id: constant.dbIDPrefix + `invalid-ref`,
      description: randomString(10),
      rawRecipe: invalidRefRecipe,
    };

    check(http.request("POST", pipelinePublicHost + collectionPath, JSON.stringify(reqInvalid), data.header), {
      [`POST ${collectionPath} (${reqInvalid.id}) response status is 201`]: (r) => r.status === 201,
    });

    check(http.request("POST", pipelinePublicHost + triggerPath(reqInvalid.id), JSON.stringify(payload), data.header), {
      [`POST ${triggerPath(reqInvalid.id)} response status is 400`]: (r) => r.status === 400,
      [`POST ${triggerPath(reqInvalid.id)} contains end-user message`]: (r) =>
        r.json().message === "String setup only supports connection references (${connection.<conn-id>}).",
    });

    check(http.request("DELETE", pipelinePublicHost + resourcePath(reqInvalid.id), null, data.header), {
      [`DELETE ${resourcePath(reqInvalid.id)} response status 204`]: (r) => r.status === 204,
    });
  });
}

const breakableRecipe = `
version: v1.0-alpha
variable:
  jota:
    type: json
component:
  jq:
    type: json
    task: TASK_JQ
    input:
      jq-filter: '.foo'
      json-value: \${variable.jota}
output:
  out:
    value: \${jq.output.results[0]}
`;

export function CheckPipelineRuns(data) {
  group("Pipelines API: View pipeline and component runs", () => {
    const creationReq = {
      id: constant.dbIDPrefix + randomString(10),
      description: randomString(50),
      rawRecipe: breakableRecipe,
    };

    // Create pipeline
    check(
      http.request(
        "POST",
        pipelinePublicHost + collectionPath,
        JSON.stringify(creationReq),
        data.header
      ),
      {
        [`POST ${collectionPath} (${creationReq.id}) response status is 201 (HTTP pipeline)`]: (r) =>
          r.status === 201,
      }
    );

    // Trigger pipeline with error
    const nokPayload = JSON.stringify({
      data: [
        {
          variable: {},
        },
      ],
    });

    const nokResp = http.request(
      "POST",
      pipelinePublicHost + triggerPath(creationReq.id),
      nokPayload,
      data.header
    );
    check(nokResp, {
      [`POST ${triggerPath(creationReq.id)} (NOK) response status is 200`]: (r) =>
        r.status === 200,
      [`POST ${triggerPath(creationReq.id)} (NOK) returns error status`]: (r) =>
        r.json().metadata.traces.jq.statuses[0] === "STATUS_ERROR",
    });

    // Successfully trigger pipeline
    const okPayload = JSON.stringify({
      data: [
        {
          variable: {
            jota: { foo: "bar" },
          },
        },
      ],
    });

    const okResp = http.request(
      "POST",
      pipelinePublicHost + triggerPath(creationReq.id),
      okPayload,
      data.header
    );
    check(okResp, {
      [`POST ${triggerPath(creationReq.id)} (OK) response status is 200`]: (r) =>
        r.status === 200,
      [`POST ${triggerPath(creationReq.id)} (OK) contains result`]: (r) =>
        r.json().outputs[0].out === "bar",
      [`POST ${triggerPath(creationReq.id)} (OK) returns successful status`]: (r) =>
        r.json().metadata.traces.jq.statuses[0] === "STATUS_COMPLETED",
    });

    const pipelineRuns = http.request(
      "GET",
      pipelinePublicHost + resourcePath(creationReq.id) + "/runs",
      null,
      data.header
    );
    check(pipelineRuns, {
      [`GET ${resourcePath(creationReq.id)}/runs response status is 200`]: (r) =>
        r.status === 200,
      [`GET ${resourcePath(creationReq.id)}/runs contains runs`]: (r) =>
        r.json().pipelineRuns.length === 2,
      [`GET ${resourcePath(creationReq.id)}/runs has a successful run`]: (r) =>
        r.json().pipelineRuns[0].status === "RUN_STATUS_COMPLETED",
      [`GET ${resourcePath(creationReq.id)}/runs has a failed run`]: (r) =>
        r.json().pipelineRuns[1].status === "RUN_STATUS_FAILED",
    });

    const okPipelineRunUID = pipelineRuns.json().pipelineRuns[0].pipelineRunUid;

    const okComponentRuns = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/pipeline-runs/${okPipelineRunUID}/component-runs`,
      null,
      data.header
    );
    check(okComponentRuns, {
      [`GET /v1beta/pipeline-runs/${okPipelineRunUID}/component-runs response status is 200`]: (r) =>
        r.status === 200,
      [`GET /v1beta/pipeline-runs/${okPipelineRunUID}/component-runs contains component runs`]: (r) =>
        r.json().componentRuns.length === 1,
      [`GET /v1beta/pipeline-runs/${okPipelineRunUID}/component-runs matches the component ID`]: (r) =>
        r.json().componentRuns[0].componentId === "jq",
      [`GET /v1beta/pipeline-runs/${okPipelineRunUID}/component-runs has a successful component run`]: (r) =>
        r.json().componentRuns[0].status === "RUN_STATUS_COMPLETED",
    });

    const nokPipelineRunUID = pipelineRuns.json().pipelineRuns[1].pipelineRunUid;
    const nokComponentRuns = http.request(
      "GET",
      `${pipelinePublicHost}/v1beta/pipeline-runs/${nokPipelineRunUID}/component-runs`,
      null,
      data.header
    );
    check(nokComponentRuns, {
      [`GET /v1beta/pipeline-runs/${nokPipelineRunUID}/component-runs response status is 200`]: (r) =>
        r.status === 200,
      [`GET /v1beta/pipeline-runs/${nokPipelineRunUID}/component-runs contains component runs`]: (r) =>
        r.json().componentRuns.length === 1,
      [`GET /v1beta/pipeline-runs/${nokPipelineRunUID}/component-runs matches the component ID`]: (r) =>
        r.json().componentRuns[0].componentId === "jq",
      [`GET /v1beta/pipeline-runs/${nokPipelineRunUID}/component-runs has a failed component run`]: (r) =>
        r.json().componentRuns[0].status === "RUN_STATUS_FAILED",
    });

    check(
      http.request(
        "DELETE",
        pipelinePublicHost + resourcePath(creationReq.id),
        null,
        data.header
      ),
      {
        [`DELETE ${resourcePath(creationReq.id)} response status 204`]: (r) =>
          r.status === 204,
      }
    );
  });
}
