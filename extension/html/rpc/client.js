const sdm = require("./sdm_grpc_web_pb.js");
const extension = require("./extension_grpc_web_pb.js");

const grpcServerAddress = '/';
const extensionClient = new extension.ExtensionHostServicePromiseClient(grpcServerAddress, null, null);
const sdmClient = new sdm.CorePromiseClient(grpcServerAddress, null, null);

module.exports = { extensionClient ,sdmClient};