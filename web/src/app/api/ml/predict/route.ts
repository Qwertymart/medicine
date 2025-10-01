import {NextResponse} from 'next/server';

export async function POST(request: Request) {
    try {
        const body = await request.json();
        console.log('Request body:', body);

        // ✅ Используйте имя Docker сервиса вместо localhost
        const ML_URL = 'http://ml-service:8052';  // Имя контейнера из docker-compose
        console.log('Calling ML service:', ML_URL);

        const response = await fetch(`${ML_URL}/api/v1/ml/predict`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(body),
        });

        console.log('ML service response status:', response.status);

        if (!response.ok) {
            const errorText = await response.text();
            console.error('ML service error:', response.status, errorText);
            return NextResponse.json({
                error: `ML service returned ${response.status}: ${errorText}`
            }, {status: response.status});
        }

        const data = await response.json();
        console.log('ML service response data:', data);
        return NextResponse.json(data);

    } catch (error) {
        console.error('API route error:', error);
        return NextResponse.json({
            error: `Fetch error: ${error.message}`
        }, {status: 500});
    }
}
