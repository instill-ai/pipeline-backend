import http from "k6/http";

import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { pipelinePublicHost } from "./const.js";

import * as constant from "./const.js";

const recipeWithoutSetup = `
version: v1beta
variable:
  recipients:
    format: array:string
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


export function CheckTrigger(data) {
  var collectionPath = `/v1beta/namespaces/${constant.defaultUsername}/pipelines`;

  function resourcePath(id) {
    return `${collectionPath}/${id}`;
  }

  function triggerPath(id) {
    return resourcePath(id) + "/trigger";
  }

  group("Pipelines API: Trigger a pipeline", () => {
    var reqHTTP = Object.assign(
      {
        id: randomString(10),
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
        id: randomString(10),
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
