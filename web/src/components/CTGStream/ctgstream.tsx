'use client';

import {useState, useEffect, useRef} from 'react';

interface StreamMessage {
    timestamp: string;
    data: any;
    type?: string;
    message?: string;
}

export function CTGStreamComponent() {
    const [messages, setMessages] = useState<StreamMessage[]>([]);
    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const eventSourceRef = useRef<EventSource | null>(null);

    const startStream = () => {
        const deviceIds = ['device1', 'device2'];
        const dataTypes = ['ecg', 'blood_pressure'];

        const params = new URLSearchParams({
            deviceIds: deviceIds.join(','),
            dataTypes: dataTypes.join(','),
        });

        try {
            setError(null);
            const eventSource = new EventSource(`/api/stream-sse?${params}`);
            eventSourceRef.current = eventSource;

            eventSource.onopen = () => {
                console.log('CTG Stream connected');
                setIsConnected(true);
            };

            eventSource.onmessage = (event) => {
                try {
                    const message: StreamMessage = JSON.parse(event.data);

                    if (message.type === 'end') {
                        console.log('Stream ended by server');
                        eventSource.close();
                        setIsConnected(false);
                        return;
                    }

                    if (message.type === 'error') {
                        setError(message.message || 'Stream error');
                        eventSource.close();
                        setIsConnected(false);
                        return;
                    }

                    setMessages((prev) => [...prev, message]);
                } catch (parseError) {
                    console.error('Error parsing message:', parseError);
                }
            };

            eventSource.onerror = (error) => {
                console.error('Stream connection error:', error);
                setError('Connection error');
                setIsConnected(false);
            };
        } catch (error) {
            console.error('Failed to start stream:', error);
            setError('Failed to start stream');
        }
    };

    const stopStream = () => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
        setIsConnected(false);
    };

    useEffect(() => {
        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
            }
        };
    }, []);

    return (
        <div className="p-4">
            <h2>CTG Data Stream</h2>

            <div className="mb-4">
                <button
                    onClick={startStream}
                    disabled={isConnected}
                    className="bg-blue-500 text-white px-4 py-2 rounded disabled:bg-gray-400"
                >
                    Start CTG Stream
                </button>
                <button
                    onClick={stopStream}
                    disabled={!isConnected}
                    className="bg-red-500 text-white px-4 py-2 rounded ml-2 disabled:bg-gray-400"
                >
                    Stop Stream
                </button>
            </div>

            {error && <div className="text-red-500 mb-4">Error: {error}</div>}

            <div className="stream-data">
                <h3>Received Data:</h3>
                <div className="max-h-96 overflow-y-auto">
                    {messages.map((msg, index) => (
                        <div key={index} className="border-b py-2">
                            <div className="text-sm text-gray-500">
                                {new Date(msg.timestamp).toLocaleTimeString()}
                            </div>
                            <pre className="text-xs">{JSON.stringify(msg.data, null, 2)}</pre>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}
