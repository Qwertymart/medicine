import type {CTGStreamServiceClient} from '@/proto/generated/ctg/CTGStreamService';
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';

// ОБЯЗАТЕЛЬНО МОНОРЕПО!!!!!!!!!!!!!!!!!!!!!!!!!!!!!11 Иначе просто не начдётся файл
const PROTO_PATH = path.join(process.cwd(), '../client_grpc/proto', 'ctg_simple.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const ctgStreamService = protoDescriptor.CTGStreamService as any;

export const serverClient: CTGStreamServiceClient = new ctgStreamService(
    'localhost:9090',
    grpc.credentials.createInsecure(),
);

export const createServerStream = (requestData: any) => {
    return new Promise((resolve, reject) => {
        const stream = serverClient.streamCtgData(requestData);
        const responses: any[] = [];

        stream.on('data', (response: any) => {
            console.log('Received stream data:', response);
            responses.push(response);
        });

        stream.on('end', () => {
            console.log('Stream ended');
            resolve(responses);
        });

        stream.on('error', (error: any) => {
            console.error('Stream error:', error);
            reject(error);
        });

        stream.on('status', (status: any) => {
            console.log('Stream status:', status);
        });
    });
};
