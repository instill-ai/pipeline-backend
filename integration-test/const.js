import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

let proto
let pHost, cHost, mHost
let pPrivatePort, pPublicPort, cPublicPort

if (__ENV.API_GATEWAY_VDP_HOST && !__ENV.API_GATEWAY_VDP_PORT || !__ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT) {
  fail("both API_GATEWAY_HOST and API_GATEWAY_VDP_PORT should be properly configured.")
}

export const apiGatewayMode = (__ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT);

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}

if (apiGatewayMode) {
  // internal mode for accessing api-gateway from container
  pHost = cHost = __ENV.API_GATEWAY_VDP_HOST
  pPrivatePort = 3081
  pPublicPort = cPublicPort = __ENV.API_GATEWAY_VDP_PORT

} else {
  // direct microservice mode
  pHost = "pipeline-backend"
  cHost = "connector-backend"
  mHost = "model-backend"
  pPrivatePort = 3081
  pPublicPort = 8081
  cPublicPort = 8082
}

export const pipelinePrivateHost = `${proto}://${pHost}:${pPrivatePort}`;
export const pipelinePublicHost = `${proto}://${pHost}:${pPublicPort}`;
export const pipelineGRPCPrivateHost = `${pHost}:${pPrivatePort}`;
export const pipelineGRPCPublicHost = `${pHost}:${pPublicPort}`;
export const connectorPublicHost = `${proto}://${cHost}:${cPublicPort}`;
export const connectorGRPCPublicHost = `${cHost}:${cPublicPort}`;

export const dogImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");
export const catImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/cat.jpg`, "b");
export const bearImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/bear.jpg`, "b");
export const dogRGBAImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog-rgba.png`, "b");


export const params = {
  headers: {
    "Content-Type": "application/json",
  },
  timeout: "1800s",
};

export const paramsGrpc = {
  metadata: {
    "Content-Type": "application/json",
  },
  timeout: "1800s",
};

const randomUUID = uuidv4();
export const paramsGRPCWithJwt = {
  metadata: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
}

export const paramsHTTPWithJwt = {
  headers: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
}

export const detSyncHTTPSingleModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-http",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": "connectors/destination-http",
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        }
      },

    ]
  },
};

export const detSyncGRPCSingleModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-grpc",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": "connectors/destination-grpc",
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },

    ]
  },
};

export const detSyncHTTPMultiModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-http",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      // {
      //   "id": "m02",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": "connectors/destination-http",
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },

    ]
  },
};

export const detSynGRPCMultiModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-grpc",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      // {
      //   "id": "m02",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": "connectors/destination-grpc",
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },

    ]
  },
};

export const dstCSVConnID1 = "some-cool-name-for-dst-csv-connector-1"
export const dstCSVConnID2 = "some-cool-name-for-dst-csv-connector-2"

export const detAsyncSingleModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-http",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": `connectors/${dstCSVConnID1}`,
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },
    ]
  },
};

export const detAsyncMultiModelRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-http",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      // {
      //   "id": "m02",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": `connectors/${dstCSVConnID1}`,
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },
    ]
  },
};

export const detAsyncMultiModelMultipleDestinationRecipe = {
  recipe: {
    version: "v1alpha",
    components: [
      {
        "id": "s01",
        "resource_name": "connectors/source-http",
        "dependencies": {},
      },
      // {
      //   "id": "m01",
      //   "resource_name": `connectors/${model_id}`,
      // },
      // {
      //   "id": "m02",
      //   "resource_name": `connectors/${model_id}`,
      // },
      {
        "id": "d01",
        "resource_name": `connectors/${dstCSVConnID1}`,
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },
      {
        "id": "d02",
        "resource_name": `connectors/${dstCSVConnID2}`,
        "dependencies": {
          "structured_data": "{**s01.structured_data}"
        },
      },
    ]
  },
};
