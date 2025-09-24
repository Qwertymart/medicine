import type * as grpc from '@grpc/grpc-js';
import type { MessageTypeDefinition } from '@grpc/proto-loader';

import type { CTGDataResponse as _ctg_CTGDataResponse, CTGDataResponse__Output as _ctg_CTGDataResponse__Output } from './ctg/CTGDataResponse';
import type { CTGStreamServiceClient as _ctg_CTGStreamServiceClient, CTGStreamServiceDefinition as _ctg_CTGStreamServiceDefinition } from './ctg/CTGStreamService';
import type { StreamRequest as _ctg_StreamRequest, StreamRequest__Output as _ctg_StreamRequest__Output } from './ctg/StreamRequest';

type SubtypeConstructor<Constructor extends new (...args: any) => any, Subtype> = {
  new(...args: ConstructorParameters<Constructor>): Subtype;
};

export interface ProtoGrpcType {
  ctg: {
    CTGDataResponse: MessageTypeDefinition<_ctg_CTGDataResponse, _ctg_CTGDataResponse__Output>
    CTGStreamService: SubtypeConstructor<typeof grpc.Client, _ctg_CTGStreamServiceClient> & { service: _ctg_CTGStreamServiceDefinition }
    StreamRequest: MessageTypeDefinition<_ctg_StreamRequest, _ctg_StreamRequest__Output>
  }
}

