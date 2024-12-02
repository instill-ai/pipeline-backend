import grpc from "k6/net/grpc";

import { check } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { describe } from 'https://jslib.k6.io/k6chaijs/4.3.4.3/index.js';

import * as constant from "./const.js";

const clientPrivate = new grpc.Client();
clientPrivate.load(
  ["../proto/vdp/pipeline/v1beta"],
  "pipeline_private_service.proto"
);

const clientPublic = new grpc.Client();
clientPublic.load(
  ["../proto/vdp/pipeline/v1beta"],
  "pipeline_public_service.proto"
);

export function CheckLookUpConnection(data) {
  var connectionID = constant.dbIDPrefix + randomString(8);
  var integrationID = "openai";

  describe("Integration API: Look up connection", () => {
    clientPrivate.connect(constant.pipelineGRPCPrivateHost, {plaintext: true});
    clientPublic.connect(constant.pipelineGRPCPublicHost, {plaintext: true});

    var conn = {
      id: connectionID,
      integrationId: integrationID,
      method: "METHOD_DICTIONARY",
      setup: {"api-key": "sk-one-2-III"},
    };

    var createPath = "vdp.pipeline.v1beta.PipelinePublicService/CreateNamespaceConnection";
    var createReq = clientPublic.invoke(createPath, {
        namespaceId: constant.defaultUsername,
        connection: conn,
    }, data.metadata);

    check(createReq, {
      [`${createPath} (${connectionID}) response StatusOK`]: (r) => r.status === grpc.StatusOK,
    });

    var connectionUID = createReq.message.connection.uid;

    var lookUpPath = "vdp.pipeline.v1beta.PipelinePrivateService/LookUpConnectionAdmin";
    var lookUpReq = clientPrivate.invoke(lookUpPath, {uid: connectionUID}, {});
    check(lookUpReq, {
      [`${lookUpPath} (${connectionUID}) response StatusOK`]: (r) => r.status === grpc.StatusOK,
      [`${lookUpPath} (${connectionUID}) response contains ID`]: (r) => r.message.connection.id === connectionID,
    });
  });
}
