'use client';

import React, {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useRef,
    useState,
    useMemo,
} from 'react';

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

interface CTGDataPoint {
    value: any;
    data_type: string;
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

const MAX_DATA_POINTS = 1000;

export const SessionProvider: React.FC<{children: React.ReactNode}> = ({children}) => {
    const [activeSession, setActiveSession] = useState<SessionResponse | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [isConnected, setIsConnected] = useState(false);
    const [ctgData, setCtgData] = useState<CTGDataPoint[]>([]);
    const eventSourceRef = useRef<EventSource | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const checkActiveSession = useCallback(async () => {
        try {
            setIsLoading(true);
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
            console.error('Error checking active session:', error);
        } finally {
            setIsLoading(false);
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

                const updatedData = [...prevData, newDataPoint];
                if (updatedData.length > MAX_DATA_POINTS) {
                    return updatedData.slice(-MAX_DATA_POINTS);
                }
                return updatedData;
            });
        }
    }, []);

    const cleanupEventSource = useCallback(() => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
            reconnectTimeoutRef.current = null;
        }
        setIsConnected(false);
    }, []);

    const startEventStream = useCallback(
        (cardId: string) => {
            cleanupEventSource();

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
                                console.log('Successfully connected to stream:', message);
                                setIsConnected(true);
                                setError(null);
                                break;

                            case 'heartbeat':
                                setIsConnected(true);
                                break;

                            case 'data':
                                processStreamMessage(message);
                                setError(null);
                                break;

                            case 'no_data':
                                console.log('No data received from devices:', message.message);
                                cleanupEventSource();
                                break;

                            case 'end':
                                console.log('Stream ended by server');
                                cleanupEventSource();
                                break;

                            case 'error':
                                console.error('Stream error:', message.message);
                                setError(message.message || 'Stream error occurred');
                                cleanupEventSource();
                                break;

                            default:
                                console.warn('Unknown message type:', message.type, message);
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
                    setError('Connection error - attempting reconnect...');
                    setIsConnected(false);

                    reconnectTimeoutRef.current = setTimeout(() => {
                        if (activeSession) {
                            startEventStream(activeSession.card_id);
                        }
                    }, 5000);
                };
            } catch (error) {
                console.error('Failed to start stream:', error);
                setError('Failed to start stream');
            }
        },
        [activeSession, cleanupEventSource, processStreamMessage],
    );

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
                    // localStorage.setItem('ctg_session', JSON.stringify(result.data));
                    startEventStream(cardId);
                }
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Ошибка запуска сессии');
                cleanupEventSource();
            } finally {
                setIsLoading(false);
            }
        },
        [cleanupEventSource, startEventStream],
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
            cleanupEventSource();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Ошибка остановки сессии');
        } finally {
            setIsLoading(false);
        }
    }, [activeSession, cleanupEventSource]);

    const clearData = useCallback(() => {
        setCtgData([]);
    }, []);

    // useEffect(() => {
    //     const savedSession = localStorage.getItem('ctg_session');
    //     if (savedSession) {
    //         try {
    //             const session = JSON.parse(savedSession);
    //             setActiveSession(session);
    //         } catch (e) {
    //             console.error('Error parsing saved session:', e);
    //             localStorage.removeItem('ctg_session');
    //         }
    //     }

    //     checkActiveSession();
    // }, [checkActiveSession]);

    useEffect(() => {
        return () => {
            cleanupEventSource();
        };
    }, [cleanupEventSource]);

    const refresh = useCallback(() => {
        checkActiveSession();
    }, [checkActiveSession]);

    const clearError = useCallback(() => setError(null), []);

    const contextValue = useMemo(
        () => ({
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
        }),
        [
            activeSession,
            isLoading,
            error,
            isConnected,
            ctgData,
            startSession,
            stopSession,
            refresh,
            clearError,
            clearData,
        ],
    );

    return <SessionContext.Provider value={contextValue}>{children}</SessionContext.Provider>;
};

export function useSession() {
    const ctx = useContext(SessionContext);
    if (ctx === undefined) throw new Error('useSession must be used within SessionProvider');
    return ctx;
}
