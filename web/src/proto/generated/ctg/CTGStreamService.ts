// Original file: ../client_grpc/proto/ctg_simple.proto

import type * as grpc from '@grpc/grpc-js'
import type { MethodDefinition } from '@grpc/proto-loader'
import type { CTGDataResponse as _ctg_CTGDataResponse, CTGDataResponse__Output as _ctg_CTGDataResponse__Output } from '../ctg/CTGDataResponse';
import type { StreamRequest as _ctg_StreamRequest, StreamRequest__Output as _ctg_StreamRequest__Output } from '../ctg/StreamRequest';

export interface CTGStreamServiceClient extends grpc.Client {
  StreamCTGData(argument: _ctg_StreamRequest, metadata: grpc.Metadata, options?: grpc.CallOptions): grpc.ClientReadableStream<_ctg_CTGDataResponse__Output>;
  StreamCTGData(argument: _ctg_StreamRequest, options?: grpc.CallOptions): grpc.ClientReadableStream<_ctg_CTGDataResponse__Output>;
  streamCtgData(argument: _ctg_StreamRequest, metadata: grpc.Metadata, options?: grpc.CallOptions): grpc.ClientReadableStream<_ctg_CTGDataResponse__Output>;
  streamCtgData(argument: _ctg_StreamRequest, options?: grpc.CallOptions): grpc.ClientReadableStream<_ctg_CTGDataResponse__Output>;
  
}

export interface CTGStreamServiceHandlers extends grpc.UntypedServiceImplementation {
  StreamCTGData: grpc.handleServerStreamingCall<_ctg_StreamRequest__Output, _ctg_CTGDataResponse>;
  
}

export interface CTGStreamServiceDefinition extends grpc.ServiceDefinition {
  StreamCTGData: MethodDefinition<_ctg_StreamRequest, _ctg_CTGDataResponse, _ctg_StreamRequest__Output, _ctg_CTGDataResponse__Output>
}
