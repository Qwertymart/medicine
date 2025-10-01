'use client';

import {Card, Text, Button} from '@gravity-ui/uikit';
import {Wrapper} from '../Wrapper';
import {useState, useEffect, useRef} from 'react';
import block from 'bem-cn-lite';
import {useSession} from '../../SessionContext';

const b = block('widget3');

export const Widget3 = () => {
    const [trendText, setTrendText] = useState<string>('');
    const [summaryText, setSummaryText] = useState<string>('');
    const [, setLoading] = useState<boolean>(false);
    const [healthStatus, setHealthStatus] = useState<string>('Проверка...');
    const {cardId, isConnected} = useSession();
    const sessionStartRef = useRef<number | null>(null);

    useEffect(() => {
        const checkHealth = async () => {
            try {
                const response = await fetch('/api/ml/health');
                const data = await response.json();
                if (data) {
                    setHealthStatus('Сервис доступен');
                } else {
                    console.error('{}');
                }
            } catch (error) {
                setHealthStatus('Сервис недоступен');
                console.error('Health check error:', error);
            }
        };

        checkHealth();

        const healthInterval = setInterval(checkHealth, 30000);

        return () => clearInterval(healthInterval);
    }, []);

    useEffect(() => {
        if (!isConnected || !cardId) return;

        sessionStartRef.current = Date.now();

        const fetchPrediction = async () => {
            setLoading(true);
            try {
                const sessionDuration = sessionStartRef.current
                    ? Math.floor((Date.now() - sessionStartRef.current) / 1000)
                    : 0;
                const t_sec = 0 + sessionDuration;

                const requestData = {
                    card_id: cardId,
                    t_sec: t_sec,
                };

                const response = await fetch('/api/ml/predict', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestData),
                });

                const data = await response.json();

                if (data.result) {
                    setTrendText(data.result.trend_text || '');
                    setSummaryText(data.result.summary?.text || '');
                }
            } catch (error) {
                console.error('Prediction error:', error);
            } finally {
                setLoading(false);
            }
        };

        // fetchPrediction();

        const time = 4 * 60 * 1000;

        // const predictionInterval = setInterval(fetchPrediction, 30000);
        let predictionInterval = setTimeout(function tick() {
            fetchPrediction;
            predictionInterval = setTimeout(tick, time); // (*)
        }, time);

        return () => clearInterval(predictionInterval);
    }, [isConnected, cardId]);

    const handleManualPredict = async () => {
        console.log('click');
        console.log(cardId);
        if (!isConnected || !cardId) return;
        // const cardId='0';

        setLoading(true);
        try {
            const sessionDuration = sessionStartRef.current
                ? Math.floor((Date.now() - sessionStartRef.current) / 1000)
                : 0;
            const t_sec = 0 + sessionDuration;

            const requestData = {
                card_id: cardId,
                target_time: t_sec,
            };

            const response = await fetch('/api/ml/predict', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestData),
            });

            const data = await response.json();

            if (data.result) {
                setTrendText(data.result.trend_text || '');
                setSummaryText(data.result.summary?.text || '');
            }
        } catch (error) {
            console.error('Prediction error:', error);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Wrapper>
            {isConnected ? (
                <>
                    <Text variant="header-1" className={b('title')}>
                        Результат
                    </Text>

                    <Card view="filled" className={b('card')}>
                        <div className={b('content')}>
                            <div className={b('status')}>
                                <Text variant="caption-2">Статус: {healthStatus}</Text>
                            </div>
                            
                            {/* <div className={b('controls')}>
                                <Button
                                    view="outlined-action"
                                    onClick={handleManualPredict}
                                    // loading={loading}
                                >
                                    Обновить предсказание
                                </Button>
                            </div>  */}
                            

                            {(trendText || summaryText) && (
                                <div className={b('results')}>
                                    {trendText && (
                                        <div className={b('result-item')}>
                                            <Text variant="subheader-2">Тренд:</Text>
                                            <Text variant="body-2">{trendText}</Text>
                                        </div>
                                    )}
                                    {summaryText && (
                                        <div className={b('result-item')}>
                                            <Text variant="subheader-2">Резюме:</Text>
                                            <Text variant="body-2">{summaryText}</Text>
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    </Card>
                </>
            ) : (
                <>
                    <Text variant="header-1" className={b('title', {no_signal: true})}>
                        No signal
                    </Text>
                </>
            )}
        </Wrapper>
    );
};
