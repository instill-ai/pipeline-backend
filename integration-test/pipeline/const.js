import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import encoding from "k6/encoding";

let proto;

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
export const pipelinePublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/vdp` : `http://api-gateway:8080/vdp`
export const mgmtPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/core` : `http://api-gateway:8080/core`
export const pipelineGRPCPrivateHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}` : `pipeline-backend:3081`;
export const pipelineGRPCPublicHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}` : `api-gateway:8080`;

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

export const namespace = "users/admin"
export const defaultUsername = "admin"
export const defaultPassword = "password"

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

export const dstCSVConnID1 = "some_cool_name_for_connector_1";
export const dstCSVConnID2 = "some_cool_name_for_connector_2";

export const simpleRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        id: "start",
        definition_name: "operator-definitions/start",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "string",
              instillFormat: "string"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end",
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
        resource_name: `users/instill-ai/connectors/${dstCSVConnID1}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            data: {
              text: "{ start.input }"
            }
          }
        }
      },
      {
        id: "d02",
        resource_name: `users/instill-ai/connectors/${dstCSVConnID2}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            data: {
              text: "{ start.input }"
            }
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
        definition_name: "operator-definitions/start",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "string",
              instillFormat: "string"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end",
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
        definition_name: "operator-definitions/start",
        configuration: {
          metadata: {
            input: {
              title: "Input",
              type: "string",
              instillFormat: "string"
            }
          }
        }
      },
      {
        id: "end",
        definition_name: "operator-definitions/end",
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
        resource_name: `users/instill-ai/connectors/${dstCSVConnID1}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            data: {
              text: "{ start.input }"
            }
          }
        }
      },
      {
        id: "d01",
        resource_name: `users/instill-ai/connectors/${dstCSVConnID2}`,
        definition_name: "connector-definitions/airbyte-destination-csv",
        configuration: {
          input: {
            data: {
              text: "{ start.input }"
            }
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
