import http from "k6/http";
import grpc from 'k6/net/grpc';
import {
  FormData
} from "https://jslib.k6.io/formdata/0.0.2/index.js";

import {
  check,
  group,
  sleep
} from 'k6';
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as pipeline from './grpc-pipeline-public.js';
import * as pipelinePrivate from './grpc-pipeline-private.js';
import * as triggerSync from './grpc-trigger-sync.js';
import * as triggerAsync from './grpc-trigger-async.js';

const client = new grpc.Client();

client.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

import * as constant from "./const.js";


export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  client.connect(constant.connectorGRPCHost, {
    plaintext: true
  });

  group("Connector Backend API: Create a http source connector", function () {

    var resp = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
      source_connector: {
        "id": "source-http",
        "source_connector_definition": "source-connector-definitions/source-http",
        "connector": {
          "configuration": {}
        }
      }
    })
    check(resp, {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector HTTP response StatusOK": (r) => r.status === grpc.StatusOK,
    });

  });

  group("Connector Backend API: Create a http destination connector", function () {

    check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
      destination_connector: {
        "id": "destination-http",
        "destination_connector_definition": "destination-connector-definitions/destination-http",
        "connector": {
          "configuration": {}
        }
      }
    }), {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response StatusOK": (r) => r.status === grpc.StatusOK,
    });

  });

  group("Connector Backend API: Create a gRPC source connector", function () {

    check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
      source_connector: {
        "id": "source-grpc",
        "source_connector_definition": "source-connector-definitions/source-grpc",
        "connector": {
          "configuration": {}
        }
      }
    }), {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector gRPC response StatusOK": (r) => r.status === grpc.StatusOK,
    });

  });

  group("Connector Backend API: Create a gRPC destination connector", function () {

    check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
      destination_connector: {
        "id": "destination-grpc",
        "destination_connector_definition": "destination-connector-definitions/destination-grpc",
        "connector": {
          "configuration": {}
        }
      }
    }), {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector gRPC response StatusOK": (r) => r.status === grpc.StatusOK,
    });

  });

  group("Connector Backend API: Create a CSV destination connector", function () {

    check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
      destination_connector: {
        "id": constant.dstCSVConnID,
        "destination_connector_definition": "destination-connector-definitions/destination-csv",
        "connector": {
          "configuration": {
            "destination_path": "/local/pipeline-backend-test"
          }
        }
      }
    }), {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector CSV response StatusOK": (r) => r.status === grpc.StatusOK,
    });

    // Check connector state being updated in 300 secs
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 300000;
    while (timeoutTime > currentTime) {
      var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
        name: `destination_connector/${constant.dstCSVConnID}`
      })
      if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

  });

  group("Model Backend API: Deploy a detection model", function () {
    let fd = new FormData();
    let model_description = randomString(20)
    fd.append("id", constant.model_id);
    fd.append("description", model_description);
    fd.append("model_definition", constant.model_def_name);
    fd.append("content", http.file(constant.det_model, "dummy-det-model.zip"));
    let createClsModelRes = http.request("POST", `${constant.modelHost}/v1alpha/models/multipart`, fd.body(), {
      headers: {
        "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
      },
    })
    check(createClsModelRes, {
      "POST /v1alpha/models/multipart task det response status": (r) => r.status === 201
    });

    // Check model creation finished
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      let res = http.get(`${constant.modelHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
        headers: {
          "Content-Type": "application/json"
        },
      })
      if (res.json().operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    var res = http.post(`${constant.modelHost}/v1alpha/models/${constant.model_id}/instances/latest/deploy`, {}, {
      headers: {
        "Content-Type": "application/json"
      },
    })

    check(res, {
      [`POST /v1alpha/models/${constant.model_id}/instances/latest/deploy online task det response status`]: (r) => r.status === 200
    });

    // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = http.get(`${constant.modelHost}/v1alpha/models/${constant.model_id}/instances/latest`, {
        headers: {
          "Content-Type": "application/json"
        },
      })
      if (res.json().instance.state === "STATE_ONLINE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

  });

  client.close()
}

export default function (data) {

  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      client.connect(constant.pipelineGRPCHost, {
        plaintext: true
      });
      check(client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/Liveness', {}), {
        "GET /health/pipeline response status is StatusOK": (r) => r.status === grpc.StatusOK,
      });
      client.close();
    });
  }

  pipeline.CheckCreate()
  pipeline.CheckList()
  pipeline.CheckGet()
  pipeline.CheckUpdate()
  pipeline.CheckUpdateState()
  pipeline.CheckRename()
  pipeline.CheckLookUp()

  triggerSync.CheckTriggerSyncSingleImageSingleModelInst()
  triggerSync.CheckTriggerSyncMultiImageSingleModelInst()
  triggerSync.CheckTriggerSyncMultiImageMultiModelInst()

  triggerAsync.CheckTriggerAsyncSingleImageSingleModelInst()
  triggerAsync.CheckTriggerAsyncMultiImageSingleModelInst()
  triggerAsync.CheckTriggerAsyncMultiImageMultiModelInst()

  if (__ENV.MODE != "api-gateway" && __ENV.MODE != "localhost") {
    pipelinePrivate.CheckList()
    pipelinePrivate.CheckGet()
    pipelinePrivate.CheckLookUp()
  }

}

export function teardown(data) {

  client.connect(constant.connectorGRPCHost, {
    plaintext: true
  });
  
  group("Connector Backend API: Delete the http source connector", function () {
    check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
      name: "source-connectors/source-http"
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });
  });

  group("Connector Backend API: Delete the http destination connector", function () {
    check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
      name: "destination-connectors/destination-http"
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });
  });

  group("Connector Backend API: Delete the gRPC source connector", function () {
    check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
      name: "source-connectors/source-grpc"
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });
  });

  group("Connector Backend API: Delete the gRPC destination connector", function () {
    check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
      name: "destination-connectors/destination-grpc"
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });
  });

  group("Connector Backend API: Delete the csv destination connector", function () {
    check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
      name: `destination-connectors/${constant.dstCSVConnID}`
    }), {
      [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });
  });

  client.close();

  group("Model Backend API: Delete the detection model", function () {
    check(http.request("DELETE", `${constant.modelHost}/v1alpha/models/${constant.model_id}`, null, {
      headers: {
        "Content-Type": "application/json"
      }
    }), {
      [`DELETE /v1alpha/models/${constant.model_id} response status is 204`]: (r) => r.status === 204,
    });
  });

  group("Connector API: Delete all pipelines created by this test", () => {

    client.connect(constant.pipelineGRPCHost, {
      plaintext: true
    });

    for (const pipeline of client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
        pageSize: 1000
      }, {}).message.pipelines) {
      check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
        name: `pipelines/${pipeline.id}`
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }

    client.close();
  });
}