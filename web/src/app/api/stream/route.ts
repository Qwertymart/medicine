import {createServerStream} from '@/lib/grpc-client';
import {NextResponse} from 'next/server';

export async function POST(request: Request) {
    try {
        const requestData = await request.json();

        const streamResponse = await createServerStream(requestData);

        return NextResponse.json({
            success: true,
            data: streamResponse,
        });
    } catch (error: any) {
        return NextResponse.json(
            {
                success: false,
                error: error.message,
            },
            {status: 500},
        );
    }
}
