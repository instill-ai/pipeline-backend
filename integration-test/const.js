export const dogImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");

export const numPipelines = 10

export const model_id = "dummy-det"
export const model_instance_id = "latest"

export const detectionRecipe = {
  recipe: {
    source: "connectors/http",
    model_instances: [`models/${model_id}/instances/${model_instance_id}`],
    destination: "connectors/http"
  },
};
