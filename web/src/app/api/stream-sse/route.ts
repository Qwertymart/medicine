import {serverClient} from '@/lib/grpc-client';
import {NextRequest} from 'next/server';

export async function GET(request: NextRequest) {
    try {
        const searchParams = request.nextUrl.searchParams;
        const deviceIdsParam = searchParams.get('device_id');

        const streamRequest: {deviceIds: string[]; dataTypes: string[]} = {
            deviceIds: deviceIdsParam ? deviceIdsParam.split(',') : [],
            dataTypes: ['fetal_heart_rate', 'uterine_contractions'],
        };

        if (streamRequest.deviceIds.length === 0) {
            return new Response(
                JSON.stringify({
                    error: 'Missing required parameter: device_id',
                }),
                {
                    status: 400,
                    headers: {'Content-Type': 'application/json'},
                },
            );
        }

        const encoder = new TextEncoder();

        const readableStream = new ReadableStream({
            async start(controller) {
                let grpcStream: any = null;
                let isStreamActive = true;

                const sendHeartbeat = () => {
                    if (isStreamActive) {
                        try {
                            const heartbeatData = JSON.stringify({
                                type: 'heartbeat',
                                timestamp: new Date().toISOString(),
                                message: 'Stream connected, waiting for data...',
                            });
                            controller.enqueue(encoder.encode(`data: ${heartbeatData}\n\n`));
                        } catch (error) {
                            console.error('Error sending heartbeat:', error);
                        }
                    }
                };

                const heartbeatInterval = setInterval(sendHeartbeat, 300000);

                try {
                    console.log('Starting gRPC stream for devices:', streamRequest.deviceIds);
                    console.log('Stream request payload:', JSON.stringify(streamRequest, null, 2));

                    grpcStream = serverClient.streamCtgData(streamRequest);

                    console.log('gRPC stream created, waiting for data...');

                    const dataTimeout = setTimeout(() => {
                        console.log('No data received for 30 seconds, possible connection issue');
                        const timeoutMessage = JSON.stringify({
                            type: 'warning',
                            timestamp: new Date().toISOString(),
                            message: 'No data received from devices - connection may be stalled',
                        });
                        controller.enqueue(encoder.encode(`data: ${timeoutMessage}\n\n`));
                    }, 30000);

                    grpcStream.on('data', (chunk: any) => {
                        try {
                            clearTimeout(dataTimeout);

                            console.log('Received data chunk:', chunk);
                            console.log('Chunk type:', typeof chunk);
                            console.log('Chunk keys:', chunk ? Object.keys(chunk) : 'null');

                            if (chunk && typeof chunk === 'object') {
                                console.log('Data content:', {
                                    device_id: chunk.device_id,
                                    data_type: chunk.data_type,
                                    value: chunk.value,
                                    time_sec: chunk.time_sec,
                                });
                            }

                            const data = JSON.stringify({
                                timestamp: new Date().toISOString(),
                                type: 'data',
                                data: chunk,
                            });
                            controller.enqueue(encoder.encode(`data: ${data}\n\n`));
                        } catch (error) {
                            console.error('Error processing chunk:', error);
                        }
                    });

                    grpcStream.on('status', (status: any) => {
                        console.log('gRPC stream status:', status);
                    });

                    grpcStream.on('metadata', (metadata: any) => {
                        console.log('gRPC stream metadata:', metadata);
                    });

                    grpcStream.on('error', (error: any) => {
                        console.error('gRPC stream error:', error);
                        console.error('Error details:', {
                            code: error.code,
                            details: error.details,
                            message: error.message,
                        });
                        clearTimeout(dataTimeout);
                        clearInterval(heartbeatInterval);
                        isStreamActive = false;

                        const errorData = JSON.stringify({
                            type: 'error',
                            timestamp: new Date().toISOString(),
                            message: error.message || 'Stream connection error',
                            code: error.code || 'UNKNOWN',
                        });
                        controller.enqueue(encoder.encode(`data: ${errorData}\n\n`));
                        controller.close();
                    });

                    request.signal.addEventListener('abort', () => {
                        console.log('Client disconnected, closing gRPC stream');
                        clearInterval(heartbeatInterval);
                        isStreamActive = false;
                        if (grpcStream) {
                            grpcStream.cancel();
                        }
                        controller.close();
                    });
                } catch (error) {
                    console.error('Error creating gRPC stream:', error);
                    clearInterval(heartbeatInterval);
                    isStreamActive = false;
                    const errorData = JSON.stringify({
                        type: 'error',
                        timestamp: new Date().toISOString(),
                        message: 'Failed to create stream connection',
                        error: error instanceof Error ? error.message : 'Unknown error',
                    });
                    controller.enqueue(encoder.encode(`data: ${errorData}\n\n`));
                    controller.close();
                }
            },

            cancel() {
                console.log('Stream cancelled by client');
            },
        });

        return new Response(readableStream, {
            headers: {
                'Content-Type': 'text/event-stream',
                'Cache-Control': 'no-cache, no-transform',
                Connection: 'keep-alive',
                'Access-Control-Allow-Origin': '*',
                'Access-Control-Allow-Methods': 'GET, OPTIONS',
                'X-Accel-Buffering': 'no',
                'Transfer-Encoding': 'chunked',
            },
        });
    } catch (error) {
        console.error('Error in stream endpoint:', error);
        return new Response(
            JSON.stringify({
                error: 'Internal server error',
            }),
            {
                status: 500,
                headers: {'Content-Type': 'application/json'},
            },
        );
    }
}

export async function OPTIONS() {
    return new Response(null, {
        status: 200,
        headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, OPTIONS',
            'Access-Control-Allow-Headers': 'Content-Type, Cache-Control',
        },
    });
}
