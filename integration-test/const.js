import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import encoding from "k6/encoding";

let proto;
let pHost, cHost;
let pPrivatePort, pPublicPort, cPublicPort;

export const apiGatewayMode = (__ENV.API_GATEWAY_URL && true);

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}


export const pipelinePrivateHost = `http://pipeline-backend:3081`;
export const pipelinePublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/vdp` : `http://pipeline-backend:8081`
export const pipelineGRPCPrivateHost = `${__ENV.API_GATEWAY_URL}`;
export const pipelineGRPCPublicHost = `${__ENV.API_GATEWAY_URL}`;
export const connectorPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/vdp` : `http://connector-backend:8082`
export const connectorGRPCPublicHost = `${__ENV.API_GATEWAY_URL}`;
export const mgmtPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/base` : `http://mgmt-backend:8084`


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
