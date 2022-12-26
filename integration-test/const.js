let pHost = __ENV.HOST ? `${__ENV.HOST}` : "pipeline-backend"
let cHost = __ENV.HOST ? `${__ENV.HOST}` : "connector-backend"
let mHost = __ENV.HOST ? `${__ENV.HOST}` : "model-backend"

let pPort = 8081
let cPort = 8082
let mPort = 8083

if (__ENV.HOST == "api-gateway") { pHost = cHost = mHost = "api-gateway" }
if (__ENV.HOST == "api-gateway") { pPort = cPort = mPort = 8080 }

export const pipelineHost = `http://${pHost}:${pPort}`;
export const connectorHost = `http://${cHost}:${cPort}`;
export const modelHost = `http://${mHost}:${mPort}`;

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
