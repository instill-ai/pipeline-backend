import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

let proto

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

export const pipelineGRPCPrivateHost = `pipeline-backend:3081`;
export const pipelineGRPCPublicHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}` : `api-gateway:8080`

export const csvDstDefRscName = "connector-definitions/airbyte-destination"
export const csvDstDefRscPermalink = "connector-definitions/975678a2-5117-48a4-a135-019619dee18e"

export const mySQLDstDefRscName = "connector-definitions/airbyte-destination"
export const mySQLDstDefRscPermalink = "connector-definitions/975678a2-5117-48a4-a135-019619dee18e"

export const namespace = "users/admin"
export const defaultUsername = "admin"
export const defaultPassword = "password"


export const csvDstConfig = {
  "destination": "airbyte-destination-csv",
  "destination_path": "/local/test"
};

export const params = {
  headers: {
    "Content-Type": "application/json",
  },
  timeout: "600s",
};

const randomUUID = uuidv4();
export const paramsGRPCWithJwt = {
  metadata: {
    "Content-Type": "application/json",
    "Instill-User-Uid": randomUUID,
  },
}

export const paramsHTTPWithJwt = {
  headers: {
    "Content-Type": "application/json",
    "Instill-User-Uid": randomUUID,
  },
}

export const clsModelOutputs = [{
  "data": {
    "classification": {
      "category": "person",
      "score": 0.99
    }
  }
}]



export const detectionModelOutputs = [
  {
    "data": {
      "detection": {
        "objects": [
          {
            "bounding_box": { "height": 0, "left": 0, "top": 99.084984, "width": 204.18988 },
            "category": "dog",
            "score": 0.980409
          },
          {
            "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
            "category": "dog",
            "score": 0.9009272
          }
        ]
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    }

  },
  {
    "data": {
      "detection": {
        "objects": [
          {
            "bounding_box": { "height": 402.58002, "left": 0, "top": 99.084984, "width": 204.18988 },
            "category": "dog",
            "score": 0.980409
          },
          {
            "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
            "category": "dog",
            "score": 0.9009272
          }
        ]
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    }
  },
  {
    "data": {
      "detection": {
        "objects": [
          {
            "bounding_box": { "height": 0, "left": 325.7926, "top": 99.084984, "width": 204.18988 },
            "category": "dog",
            "score": 0.980409
          },
          {
            "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
            "category": "dog",
            "score": 0.9009272
          }
        ]
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    }
  }
]

export const detectionEmptyModelOutputs = [{

  "data": {
    "detection": {
      "objects": []
    }
  }

}]


export const keypointModelOutputs = [{
  "data": {
    "keypoint": {
      "objects": [
        {
          "keypoints": [{ "x": 10, "y": 100, "v": 0.6 }, { "x": 11, "y": 101, "v": 0.2 }],
          "score": 0.99
        },
        {
          "keypoints": [{ "x": 20, "y": 10, "v": 0.6 }, { "x": 12, "y": 120, "v": 0.7 }],
          "score": 0.99
        },
      ]
    }
  }

}]

export const ocrModelOutputs = [{
  "data": {
    "ocr": {
      "objects": [
        {
          "bounding_box": { "height": 402.58002, "left": 0, "top": 99.084984, "width": 204.18988 },
          "text": "some text",
          "score": 0.99
        },
        {
          "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
          "text": "some text",
          "score": 0.99
        },
      ],
    }
  }

}]

export const semanticSegModelOutputs = [{
  "data": {
    "semantic_segmentation": {
      "stuffs": [
        {
          "rle": "2918,12,382,33,...",
          "category": "person"
        },
        {
          "rle": "34,18,230,18,...",
          "category": "sky"
        },
        {
          "rle": "34,18,230,18,...",
          "category": "dog"
        }
      ]

    }
  }
}]

export const instSegModelOutputs = [{
  "data": {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
    "structured_data": {
      "instance_segmentation": {
        "objects": [
          {
            "rle": "11,6,35,8,59,10,83,12,107,14,131,16,156,16,180,18,205,18,229,...",
            "score": 0.9996394,
            "bounding_box": {
              "top": 375,
              "left": 166,
              "width": 25,
              "height": 70
            },
            "category": "dog"
          },
          {
            "rle": "11,6,35,8,59,10,83,12,107,14,131,16,156,16,180,18,205,18,229,...",
            "score": 0.9990727,
            "bounding_box": {
              "top": 107,
              "left": 240,
              "width": 27,
              "height": 27
            },
            "category": "car"
          }
        ]
      }
    }
  }
}]

export const textToImageModelOutputs = [{
  "data": {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
    "structured_data": {
      "text_to_image": {
        "images": [
          "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
          "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
          "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
          "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE"
        ]
      }
    }
  }
}]

export const textGenerationModelOutputs = [{
  "data": {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
    "structured_data": {
      "text_generation": {
        "text": "The winds of change are blowing strong, bring new beginnings, righting wrongs. The world around us is constantly turning, and with each sunrise, our spirits are yearning..."
      }
    }
  }
}]

export const unspecifiedModelOutputs = [{
  "data": {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
    "structured_data": {
      "unspecified": {
        "raw_outputs": [
          {
            "name": "some unspecified model output",
            "data_type": "INT8",
            "shape": [3, 3, 3],
            "data": [1, 2, 3, 4, 5, 6, 7]
          },
        ],
      }
    }
  }
}]
