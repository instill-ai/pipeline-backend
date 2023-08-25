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
    timeout: "10s",
  });


  group(
    "Connector Backend API: Create a CSV destination connector 1",
    function () {
      check(
        client.invoke(
          "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource",
          {
            parent: `${constant.namespace}`,
            connector_resource: {
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
          "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource",
        {
          name: `${constant.namespace}/connector-resources/${constant.dstCSVConnID1}`,
        }
      );
    }
  );
  group(
    "Connector Backend API: Create a CSV destination connector 2",
    function () {
      check(
        client.invoke(
          "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource",
          {
            parent: `${constant.namespace}`,
            connectorResource: {
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
          "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV response StatusOK":
            (r) => r.status === grpc.StatusOK,
        }
      );
      client.invoke(
        "vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource",
        {
          name: `${constant.namespace}/connector-resources/${constant.dstCSVConnID2}`,
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
  pipeline.CheckRename()
  pipeline.CheckLookUp()

  trigger.CheckTrigger();
  triggerAsync.CheckTrigger();

  if (!constant.apiGatewayMode) {
    pipelinePrivate.CheckList()
    pipelinePrivate.CheckLookUp()
    pipelineWithJwt.CheckCreate()
    pipelineWithJwt.CheckList()
    pipelineWithJwt.CheckGet()
    pipelineWithJwt.CheckUpdate()
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
      "vdp.pipeline.v1alpha.PipelinePublicService/ListUserPipelines",
      {
        parent: `${constant.namespace}`,
        pageSize: 1000,
      },
      {}
    ).message.pipelines) {
      check(
        client.invoke(
          `vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`,
          {
            name: `${constant.namespace}/pipelines/${pipeline.id}`,
          }
        ),
        {
          [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }

    client.close();
  });

  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true,
  });


  group(
    "Connector Backend API: Delete the csv destination connector 1",
    function () {
      check(
        client.invoke(
          `vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`,
          {
            name: `${constant.namespace}/connector-resources/${constant.dstCSVConnID1}`,
          }
        ),
        {
          [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource response StatusOK`]:
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
          `vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`,
          {
            name: `${constant.namespace}/connector-resources/${constant.dstCSVConnID2}`,
          }
        ),
        {
          [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource response StatusOK`]:
            (r) => r.status === grpc.StatusOK,
        }
      );
    }
  );

  client.close();
}
