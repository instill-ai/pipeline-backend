import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import encoding from "k6/encoding";

let proto;
let pHost, cHost;
let pPrivatePort, pPublicPort, cPublicPort;

if (
  (__ENV.API_GATEWAY_VDP_HOST && !__ENV.API_GATEWAY_VDP_PORT) ||
  (!__ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT)
) {
  fail(
    "both API_GATEWAY_HOST and API_GATEWAY_VDP_PORT should be properly configured."
  );
}

export const apiGatewayMode =
  __ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT;

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (
    __ENV.API_GATEWAY_PROTOCOL !== "http" &&
    __ENV.API_GATEWAY_PROTOCOL != "https"
  ) {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL");
  }
  proto = __ENV.API_GATEWAY_PROTOCOL;
} else {
  proto = "http";
}

if (apiGatewayMode) {
  // internal mode for accessing api-gateway from container
  pHost = cHost = __ENV.API_GATEWAY_VDP_HOST;
  pPrivatePort = 3081;
  pPublicPort = cPublicPort = __ENV.API_GATEWAY_VDP_PORT;
} else {
  // direct microservice mode
  pHost = "pipeline-backend";
  cHost = "connector-backend";
  pPrivatePort = 3081;
  pPublicPort = 8081;
  cPublicPort = 8082;
}

export const pipelinePrivateHost = `${proto}://${pHost}:${pPrivatePort}`;
export const pipelinePublicHost = `${proto}://${pHost}:${pPublicPort}`;
export const pipelineGRPCPrivateHost = `${pHost}:${pPrivatePort}`;
export const pipelineGRPCPublicHost = `${pHost}:${pPublicPort}`;
export const connectorPublicHost = `${proto}://${cHost}:${cPublicPort}`;
export const connectorGRPCPublicHost = `${cHost}:${cPublicPort}`;

export const dogImg = encoding.b64encode(
  open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b")
);
export const catImg = encoding.b64encode(
  open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/cat.jpg`, "b")
);
export const bearImg = encoding.b64encode(
  open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/bear.jpg`, "b")
);
export const dogRGBAImg = encoding.b64encode(
  open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog-rgba.png`, "b")
);

export const namespace = "users/instill-ai"

export const params = {
  headers: {
    "Content-Type": "application/json",
  },
  timeout: "10s",
};

export const paramsGrpc = {
  metadata: {
    "Content-Type": "application/json",
  },
  timeout: "10s",
};

const randomUUID = uuidv4();
export const paramsGRPCWithJwt = {
  metadata: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
};

export const paramsHTTPWithJwt = {
  headers: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
};

export const dstCSVConnID1 = "some-cool-name-for-dst-csv-connector-1";
export const dstCSVConnID2 = "some-cool-name-for-dst-csv-connector-2";

export const simpleRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        id: "start",
        definition_name: "operator-definitions/start-operator",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "text"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end-operator",
        configuration: {
          metadata: {
            answer: {
              title: "Answer"
            }
          },
          input: {
            answer: "{ start.input }"
          }
        }
      },
      {
        id: "d01",
        resource_name: `users/instill-ai/connector-resources/${dstCSVConnID1}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            text: "{ start.input }"
          }
        }
      },
      {
        id: "d02",
        resource_name: `users/instill-ai/connector-resources/${dstCSVConnID2}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            text: "{ start.input }"
          }
        }
      },
    ],
  },
};

export const simpleRecipeWithoutCSV = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        id: "start",
        definition_name: "operator-definitions/start-operator",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "text"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end-operator",
        configuration: {
          metadata: {
            answer: {
              title: "Answer"
            }
          },
          input: {
            answer: "{ start.input }"
          }
        }
      },
    ],
  },
};

export const simpleRecipeDupId = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        id: "start",
        definition_name: "operator-definitions/start-operator",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "text"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end-operator",
        configuration: {
          metadata: {
            answer: {
              title: "Answer"
            }
          },
          input: {
            answer: "{ start.input }"
          }
        }
      },
      {
        id: "d01",
        resource_name: `users/instill-ai/connector-resources/${dstCSVConnID1}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            text: "{ start.input }"
          }
        }
      },
      {
        id: "d01",
        resource_name: `users/instill-ai/connector-resources/${dstCSVConnID2}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            text: "{ start.input }"
          }
        }
      },
    ],
  },
};

export const simplePayload = {
  inputs: [
    {
      input: "a",
    },
  ],
};
