export const dogImg = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");

export const numPipelines = 10

export const detectionModel = {
  name: "dummy-det",
  instance_name: "latest",
  version: 1 // TODO: DEPRECATE THIS
};

export const detectionRecipe = {
  recipe: {
    source: {
      name: "HTTP",
    },
    models: [
      detectionModel,
    ],
    destination: {
      name: "HTTP",
    },
  },
};
