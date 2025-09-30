'use client';

import {Card, Text, Button} from '@gravity-ui/uikit';
import {Wrapper} from '../Wrapper';
import {useState} from 'react';
import block from 'bem-cn-lite';
import {useSession} from '../../SessionContext';

const b = block('widget3');

export const Widget3 = () => {
    const [trendText, setTrendText] = useState<string>('');
    const [summaryText, setSummaryText] = useState<string>('');
    const [loading, setLoading] = useState<boolean>(false);
    const {cardId, isConnected} = useSession();

    const handlePredict = async () => {
        setLoading(true);
        try {
            const requestData = {
                card_id: cardId,
                t_sec: 960,
            };

            const response = await fetch('/api/v1/ml/predict', {
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

    const handleHealthCheck = async () => {
        try {
            const response = await fetch('/api/v1/ml/health');
            const data = await response.json();
            console.log('Health status:', data);
        } catch (error) {
            console.error('Health check error:', error);
        }
    };

    return (
        <Wrapper>
            {isConnected ? (
                <>
                    <Text variant="header-2" className={b('title', {no_signal: false})}>
                        Результат
                    </Text>

                    <Card view="filled" className={b('card')}>
                        <div className={b('content')}>
                            <div className={b('controls')}>
                                <Button view="action" onClick={handlePredict} loading={loading}>
                                    Получить предсказание
                                </Button>
                                <Button view="outlined" onClick={handleHealthCheck}>
                                    Проверить здоровье сервиса
                                </Button>
                            </div>

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
