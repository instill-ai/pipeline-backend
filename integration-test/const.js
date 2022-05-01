export const dogImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");

export const numPipelines = 10

export const model_name = "dummy-det"
export const model_instance_name = "latest"

export const detectionRecipe = {
  recipe: {
    source: {
      name: "HTTP",
    },
    models: [
      {
        name: model_name,
        instance_name: model_instance_name
      }
    ],
    destination: {
      name: "HTTP",
    },
  },
};
