import {NextResponse} from 'next/server';

const ML_API_URL = process.env.NEXT_PUBLIC_API_ML_URL || 'http://localhost:8052';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        console.log(body);
        const response = await fetch(`${ML_API_URL}/api/v1/ml/predict`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(body),
        });

        const data = await response.json();
        return NextResponse.json(data);
    } catch (error) {
        return NextResponse.json({error: `Internal error ${error}`}, {status: 500});
    }
}
