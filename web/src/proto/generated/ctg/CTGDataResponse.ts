// Original file: ../client_grpc/proto/ctg_simple.proto


export interface CTGDataResponse {
  'deviceId'?: (string);
  'dataType'?: (string);
  'value'?: (number | string);
  'timeSec'?: (number | string);
}

export interface CTGDataResponse__Output {
  'deviceId': (string);
  'dataType': (string);
  'value': (number);
  'timeSec': (number);
}
