import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';

const PROTO_PATH = path.join(process.cwd(), '../client_grpc/proto', 'ctg_simple.proto');

// Проверяем существование файла
try {
    const fs = require('fs');
    if (!fs.existsSync(PROTO_PATH)) {
        throw new Error(`Proto file not found at: ${PROTO_PATH}`);
    }
    console.log('Proto file found at:', PROTO_PATH);
} catch (error) {
    console.error('Error checking proto file:', error);
}

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);

// ДЕБАГГИНГ: Выводим структуру protoDescriptor для понимания
// TODO: его можно убрать
console.log('Proto descriptor keys:', Object.keys(protoDescriptor));

let ctgStreamService: any;

if (protoDescriptor.CTGStreamService) {
    ctgStreamService = protoDescriptor.CTGStreamService;
} 
else if ((protoDescriptor as any).ctg && (protoDescriptor as any).ctg.CTGStreamService) {
    ctgStreamService = (protoDescriptor as any).ctg.CTGStreamService;
}
else {
    const findService = (obj: any, path: string = ''): any => {
        for (const key in obj) {
            const currentPath = path ? `${path}.${key}` : key;
            if (key === 'CTGStreamService' && typeof obj[key] === 'function') {
                console.log('Found CTGStreamService at:', currentPath);
                return obj[key];
            }
            if (typeof obj[key] === 'object' && obj[key] !== null) {
                const result = findService(obj[key], currentPath);
                if (result) return result;
            }
        }
        return null;
    };
    
    ctgStreamService = findService(protoDescriptor);
}

if (!ctgStreamService) {
    console.error('CTGStreamService not found in proto descriptor. Available keys:');
    console.dir(protoDescriptor, { depth: 3 });
    throw new Error('CTGStreamService not found in the proto file');
}

console.log('CTGStreamService type:', typeof ctgStreamService);

if (typeof ctgStreamService !== 'function') {
    console.error('CTGStreamService is not a constructor:', ctgStreamService);
    throw new Error('CTGStreamService is not a constructor');
}

export const serverClient = new ctgStreamService(
    'localhost:9090',
    grpc.credentials.createInsecure(),
);

export const createServerStream = (requestData: any) => {
    return new Promise((resolve, reject) => {
        if (typeof serverClient.streamCtgData !== 'function') {
            const availableMethods = Object.keys(serverClient.constructor.prototype)
                .filter(key => typeof serverClient[key] === 'function');
            console.error('streamCtgData method not found. Available methods:', availableMethods);
            reject(new Error('streamCtgData method not available on CTGStreamService'));
            return;
        }

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
