let proto
let pHost, cHost, mHost
let pPort, cPort, mPort

if (__ENV.MODE == "api-gateway") {
  // api-gateway mode
  proto = "http"
  pHost = cHost = mHost = "api-gateway"
  pPort = cPort = mPort = 8080
} else if (__ENV.MODE == "localhost") {
  // localhost mode for GitHub Actions
  proto = "http"
  pHost = cHost = mHost = "localhost"
  pPort = cPort = mPort = 8080
} else {
  // direct microservice mode
  proto = "http"
  pHost = "pipeline-backend"
  cHost = "connector-backend"
  mHost = "model-backend"
  pPort = 8081
  cPort = 8082
  mPort = 8083
}

export const pipelineHost = `${proto}://${pHost}:${pPort}`;
export const pipelineGRPCHost = `${pHost}:${pPort}`;
export const connectorHost = `${proto}://${cHost}:${cPort}`;
export const connectorGRPCHost = `${cHost}:${cPort}`;
export const modelHost = `${proto}://${mHost}:${mPort}`;

export const dogImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");
export const catImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/cat.jpg`, "b");
export const bearImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/bear.jpg`, "b");
export const dogRGBAImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog-rgba.png`, "b");

export const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
export const model_def_name = "model-definitions/local"
export const model_id = "dummy-det"
export const model_instance_id = "latest"

export const detSyncHTTPSingleModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-http",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`,
    ],
    destination: "destination-connectors/destination-http"
  },
};

export const detSyncGRPCSingleModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-grpc",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`,
    ],
    destination: "destination-connectors/destination-grpc"
  },
};

export const detSyncHTTPMultiModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-http",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`,
      `models/${model_id}/instances/${model_instance_id}`,
    ],
    destination: "destination-connectors/destination-http"
  },
};

export const detSynGRPCMultiModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-grpc",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`,
      `models/${model_id}/instances/${model_instance_id}`,
    ],
    destination: "destination-connectors/destination-grpc"
  },
};

export const dstCSVConnID = "some-cool-name-for-dst-csv-connector"

export const detAsyncSingleModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-http",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`
    ],
    destination: `destination-connectors/${dstCSVConnID}`
  },
};

export const detAsyncMultiModelInstRecipe = {
  recipe: {
    source: "source-connectors/source-http",
    model_instances: [
      `models/${model_id}/instances/${model_instance_id}`,
      `models/${model_id}/instances/${model_instance_id}`,
    ],
    destination: `destination-connectors/${dstCSVConnID}`
  },
};
