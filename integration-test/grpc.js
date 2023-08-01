import grpc from "k6/net/grpc";

import { check, group } from "k6";

import * as pipeline from "./grpc-pipeline-public.js";
import * as pipelineWithJwt from "./grpc-pipeline-public-with-jwt.js";
import * as pipelinePrivate from "./grpc-pipeline-private.js";
import * as trigger from "./grpc-trigger.js";
import * as triggerAsync from "./grpc-trigger-async.js";

const client = new grpc.Client();

client.load(["proto/vdp/pipeline/v1alpha"], "pipeline_public_service.proto");
client.load(["proto/vdp/connector/v1alpha"], "connector_public_service.proto");

import * as constant from "./const.js";

export let options = {
  setupTimeout: "300s",
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true,
    timeout: "1800s",
  });

  group("Connector Backend API: Create a source connector", function () {
    var resp = client.invoke(
      "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector",
      {
        connector: {
          id: "start-operator",
          connector_definition_name: "connector-definitions/start-operator",
          configuration: {},
        },
      }
    );
    check(resp, {
      "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response StatusOK":
        (r) => r.status === grpc.StatusOK,
    });
  });

  group(
    "Connector Backend API: Create a destination connector",
    function () {
      check(
        client.invoke(
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector",
          {
            connector: {
              id: "end-operator",
              connector_definition_name:
                "connector-definitions/end-operator",
              configuration: {},
            },
          }
        ),
        {
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
    }
  );

  group(
    "Connector Backend API: Create a CSV destination connector 1",
    function () {
      check(
        client.invoke(
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector",
          {
            connector: {
              id: constant.dstCSVConnID1,
              connector_definition_name:
                "connector-definitions/airbyte-destination-csv",
              configuration: {
                destination_path: "/local/pipeline-backend-test-1",
              },
            },
          },
          constant.paramsGrpc
        ),
        {
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector",
        {
          name: `connectors/${constant.dstCSVConnID1}`,
        }
      );
    }
  );
  group(
    "Connector Backend API: Create a CSV destination connector 2",
    function () {
      check(
        client.invoke(
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector",
          {
            connector: {
              id: constant.dstCSVConnID2,
              connector_definition_name:
                "connector-definitions/airbyte-destination-csv",
              configuration: {
                destination_path: "/local/pipeline-backend-test-2",
              },
            },
          },
          constant.paramsGrpc
        ),
        {
          "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector",
        {
          name: `connectors/${constant.dstCSVConnID2}`,
        }
      );
    }
  );

  client.close();
}

export default function (data) {
  /*
   * Pipelines API - API CALLS
   */

  // Health check
  {
    group("Pipelines API: Health check", () => {
      client.connect(constant.pipelineGRPCPublicHost, {
        plaintext: true,
      });
      check(
        client.invoke(
          "vdp.pipeline.v1alpha.PipelinePublicService/Liveness",
          {}
        ),
        {
          "GET /health/pipeline response status is StatusOK": (r) =>
            r.status === grpc.StatusOK,
        }
      );
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
  pipeline.CheckWatch()

  trigger.CheckTriggerSingleImageSingleModel();
  trigger.CheckTriggerMultiImageSingleModel();
  trigger.CheckTriggerWithDependency();
  triggerAsync.CheckTriggerAsyncSingleImageSingleModel();
  triggerAsync.CheckTriggerAsyncMultiImageSingleModel();
  triggerAsync.CheckTriggerAsyncSingleResponse()

  if (!constant.apiGatewayMode) {
    pipelinePrivate.CheckList()
    pipelinePrivate.CheckLookUp()
    pipelineWithJwt.CheckCreate()
    pipelineWithJwt.CheckList()
    pipelineWithJwt.CheckGet()
    pipelineWithJwt.CheckUpdate()
    pipelineWithJwt.CheckUpdateState()
    pipelineWithJwt.CheckRename()
    pipelineWithJwt.CheckLookUp()
  }
}

export function teardown(data) {
  group("Pipeline API: Delete all pipelines created by this test", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true,
    });

    for (const pipeline of client.invoke(
      "vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines",
      {
        pageSize: 1000,
      },
      {}
    ).message.pipelines) {
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`,
          {
            name: `pipelines/${pipeline.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    client.close();
  });

  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true,
  });

  group("Connector Backend API: Delete the source connector", function () {
    check(
      client.invoke(
        `vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`,
        {
          name: "connectors/start-operator",
        }
      ),
      {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response StatusOK`]:
          (r) => r.status === grpc.StatusOK,
      }
    );
  });

  group(
    "Connector Backend API: Delete the destination connector",
    function () {
      check(
        client.invoke(
          `vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`,
          {
            name: "connectors/end-operator",
          }
        ),
        {
          [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }
  );

  group(
    "Connector Backend API: Delete the csv destination connector 1",
    function () {
      check(
        client.invoke(
          `vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`,
          {
            name: `connectors/${constant.dstCSVConnID1}`,
          }
        ),
        {
          [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }
  );
  group(
    "Connector Backend API: Delete the csv destination connector 2",
    function () {
      check(
        client.invoke(
          `vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`,
          {
            name: `connectors/${constant.dstCSVConnID2}`,
          }
        ),
        {
          [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }
  );

  client.close();
}
