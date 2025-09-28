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

        if (streamRequest.deviceIds.length === 0 || streamRequest.dataTypes.length === 0) {
            return new Response(
                JSON.stringify({
                    error: 'Missing required parameters: deviceIds and dataTypes are required',
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
                try {
                    const grpcStream = serverClient.streamCtgData(streamRequest);

                    grpcStream.on('data', (chunk: any) => {
                        try {
                            const data = JSON.stringify({
                                timestamp: new Date().toISOString(),
                                data: chunk,
                            });
                            controller.enqueue(encoder.encode(`data: ${data}\n\n`));
                        } catch (error) {
                            console.error('Error processing chunk:', error);
                        }
                    });

                    grpcStream.on('end', () => {
                        console.log('gRPC stream ended');
                        controller.enqueue(encoder.encode('data: {"type": "end"}\n\n'));
                        controller.close();
                    });

                    grpcStream.on('error', (error: any) => {
                        console.error('gRPC stream error:', error);
                        const errorData = JSON.stringify({
                            type: 'error',
                            message: error.message,
                        });
                        controller.enqueue(encoder.encode(`data: ${errorData}\n\n`));
                        controller.close();
                    });

                    request.signal.addEventListener('abort', () => {
                        console.log('Client disconnected');
                        grpcStream.cancel();
                        controller.close();
                    });
                } catch (error) {
                    console.error('Error creating gRPC stream:', error);
                    const errorData = JSON.stringify({
                        type: 'error',
                        message: 'Failed to create stream connection',
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
            'Access-Control-Allow-Headers': 'Content-Type',
        },
    });
}
