'use client';

import React, {createContext, useCallback, useContext, useEffect, useRef, useState} from 'react';

interface SessionResponse {
    session_id: string;
    card_id: string;
    device_id: string;
    status: 'active' | 'stopped';
    start_time: string;
    end_time?: string;
    duration: number;
}

interface ApiError {
    error: string;
    details?: string;
}

interface StreamMessage {
    timestamp: string;
    data: any;
    type?: string;
    message?: string;
}

// Интерфейсы для данных КТГ
interface CTGDataPoint {
    timestamp: number;
    fetalHeartRate?: number;
    uterineContractions?: number;
}

interface SessionContextValue {
    activeSession: SessionResponse | null;
    cardId: string | null;
    deviceId: string | null;
    sessionId: string | null;
    isLoading: boolean;
    error: string | null;
    isConnected: boolean;
    ctgData: CTGDataPoint[];
    startSession: (cardId: string) => Promise<void>;
    stopSession: () => Promise<void>;
    refresh: () => void;
    clearError: () => void;
    clearData: () => void;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

const SessionContext = createContext<SessionContextValue | undefined>(undefined);

export const SessionProvider: React.FC<{children: React.ReactNode}> = ({children}) => {
    const [activeSession, setActiveSession] = useState<SessionResponse | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [ctgData, setCtgData] = useState<CTGDataPoint[]>([]);
    const eventSourceRef = useRef<EventSource | null>(null);

    const checkActiveSession = useCallback(async () => {
        try {
            const response = await fetch(`${API_BASE}/sessions/active`);
            if (response.ok) {
                const data = await response.json();
                if (data.sessions && data.sessions.length > 0) {
                    setActiveSession(data.sessions[0]);
                } else {
                    setActiveSession(null);
                }
            }
        } catch (error) {
            setError(error instanceof Error ? error.message : 'Ошибка получения активной сессии');
        }
    }, []);

    const processStreamMessage = useCallback((message: StreamMessage) => {
        if (message.data && message.timestamp) {
            const timestamp = new Date(message.timestamp).getTime();

            setCtgData((prevData) => {
                const newDataPoint: CTGDataPoint = {
                    timestamp,
                    ...message.data,
                };

                const updatedData = [...prevData, newDataPoint].slice(-1000);
                return updatedData;
            });
        }
    }, []);

    const startSession = useCallback(
        async (cardId: string) => {
            if (!cardId.trim()) {
                setError('Введите card_id');
                return;
            }

            setIsLoading(true);
            setError(null);
            setCtgData([]);

            try {
                const response = await fetch(`${API_BASE}/sessions/start`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        device_id: process.env.NEXT_PUBLIC_DEVICE_ID || '',
                        card_id: cardId.trim(),
                    }),
                });

                if (!response.ok) {
                    const errorData: ApiError = await response.json();
                    throw new Error(errorData.details || errorData.error);
                }

                const result = await response.json();
                if (result.data) {
                    setActiveSession(result.data);
                    localStorage.setItem('ctg_session', JSON.stringify(result.data));

                    const startStream = () => {
                        const params = new URLSearchParams({
                            device_id: process.env.NEXT_PUBLIC_DEVICE_ID || '',
                            card_id: cardId.trim(),
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

                                    switch (message.type) {
                                        case 'connected':
                                            console.log(
                                                'Successfully connected to stream for devices:',
                                                message.data?.device_id,
                                            );
                                            setIsConnected(true);
                                            setError(null);
                                            break;

                                        case 'heartbeat':
                                            break;

                                        case 'data':
                                            processStreamMessage(message);
                                            setError(null);
                                            break;

                                        case 'no_data':
                                            console.log(
                                                'No data received from devices:',
                                                message.message,
                                            );
                                            eventSource.close();
                                            setIsConnected(false);
                                            break;

                                        case 'end':
                                            console.log('Stream ended by server');
                                            eventSource.close();
                                            setIsConnected(false);
                                            break;

                                        case 'error':
                                            console.error('Stream error:', message.message);
                                            setError(message.message || 'Stream error occurred');
                                            eventSource.close();
                                            setIsConnected(false);
                                            break;

                                        default:
                                            console.warn(
                                                'Unknown message type:',
                                                message.type,
                                                message,
                                            );
                                            break;
                                    }
                                } catch (parseError) {
                                    console.error(
                                        'Error parsing message:',
                                        parseError,
                                        'Raw event data:',
                                        event.data,
                                    );
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

                    startStream();
                }
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Ошибка запуска сессии');
            } finally {
                setIsLoading(false);
            }
        },
        [processStreamMessage],
    );

    const stopSession = useCallback(async () => {
        if (!activeSession) return;

        setIsLoading(true);
        setError(null);

        try {
            const response = await fetch(`${API_BASE}/sessions/stop/${activeSession.session_id}`, {
                method: 'POST',
            });

            if (!response.ok) {
                const error: ApiError = await response.json();
                throw new Error(error.details || error.error);
            }

            setActiveSession(null);

            if (eventSourceRef.current) {
                eventSourceRef.current.close();
                eventSourceRef.current = null;
            }
            setIsConnected(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Ошибка остановки сессии');
        } finally {
            setIsLoading(false);
        }
    }, [activeSession]);

    const clearData = useCallback(() => {
        setCtgData([]);
    }, []);

    useEffect(() => {
        checkActiveSession();
    }, [checkActiveSession]);

    useEffect(() => {
        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
            }
        };
    }, []);

    const refresh = useCallback(() => checkActiveSession(), [checkActiveSession]);
    const clearError = useCallback(() => setError(null), []);

    return (
        <SessionContext.Provider
            value={{
                activeSession,
                cardId: activeSession?.card_id ?? null,
                deviceId: activeSession?.device_id ?? null,
                sessionId: activeSession?.session_id ?? null,
                isLoading,
                error,
                isConnected,
                ctgData,
                startSession,
                stopSession,
                refresh,
                clearError,
                clearData,
            }}
        >
            {children}
        </SessionContext.Provider>
    );
};

export function useSession() {
    const ctx = useContext(SessionContext);
    if (ctx === undefined) throw new Error('useSession must be used within SessionProvider');
    return ctx;
}
