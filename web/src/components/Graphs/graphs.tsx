'use client';

import {useEffect, useState} from 'react';
import ChartKit, {settings} from '@gravity-ui/chartkit';
import {YagrPlugin} from '@gravity-ui/chartkit/yagr';
import {IndicatorPlugin} from '@gravity-ui/chartkit/indicator';
import type {YagrWidgetData} from '@gravity-ui/chartkit/yagr';
import type {IndicatorWidgetData, IndicatorWidgetDataItem} from '@gravity-ui/chartkit/indicator';
import block from 'bem-cn-lite';
import {useSession} from '@/components/Dashboard/SessionContext';

const b = block('graph');

settings.set({plugins: [YagrPlugin, IndicatorPlugin]});

interface GraphsProps {
    dataType: 'fetalHeartRate' | 'uterineContractions';
    title: string;
    color: string;
}

export function Graphs({dataType, title, color}: GraphsProps) {
    const {ctgData, isConnected} = useSession();
    const [indicatorData, setIndicatorData] = useState<IndicatorWidgetData>({
        data: [],
    });

    useEffect(() => {
        console.log(ctgData);
    }, [ctgData]);

    const graphData: YagrWidgetData = {
        data: {
            timeline: ctgData.map((point: {timestamp: string | number | Date}) =>
                new Date(point.timestamp).getTime(),
            ),
            graphs: [
                {
                    id: dataType,
                    name: title,
                    color: color,
                    data: ctgData
                        .filter((point) => point.data_type === dataType)
                        .map((point) => (point.value !== (-1 || undefined) ? point.value : null)),
                },
            ],
        },
        libraryConfig: {
            chart: {
                series: {
                    type: 'line',
                },
            },
            title: {
                text: title,
            },
            axes: {
                x: {
                    label: 'Время',
                },
                y: {
                    label: dataType === 'fetalHeartRate' ? 'уд/мин' : 'ед.',
                },
            },
        },
    };

    useEffect(() => {
        if (ctgData.length > 0) {
            const lastPoint = ctgData[ctgData.length - 1];
            const value = lastPoint[dataType] || 0;

            const indicator: IndicatorWidgetDataItem = {
                content: {
                    current: {
                        value: value,
                    } as {value: string | number} & Record<string, unknown>,
                },
                color: color,
                title: title,
                size: 'm',
                nowrap: true,
            };

            setIndicatorData({data: [indicator]});
        }
    }, [ctgData, dataType, title, color]);

    if (!isConnected && ctgData.length === 0) {
        return (
            <div
                className={b('container')}
                style={{
                    display: 'flex',
                    flexDirection: 'column',
                    height: 400,
                    gap: '20px',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}
            >
                <div>Ожидание данных КТГ...</div>
                <div style={{fontSize: '14px', color: '#666'}}>
                    Запустите сессию для начала мониторинга
                </div>
            </div>
        );
    }

    return (
        <div
            className={b('container')}
            style={{
                display: 'flex',
                flexDirection: 'column',
                height: 'auto',
                gap: '15px',
            }}
        >
            <div
                className={b('chart')}
                style={{
                    flex: 1,
                    border: '1px solid #e0e0e0',
                    borderRadius: '8px',
                    padding: '12px',
                    background: 'white',
                }}
            >
                <ChartKit type="yagr" data={graphData} />
            </div>

            <div
                className={b('indicators')}
                style={{
                    height: '100px',
                }}
            >
                {indicatorData.data?.map((indicator, index) => (
                    <div
                        key={index}
                        style={{
                            border: '1px solid #e0e0e0',
                            borderRadius: '8px',
                            padding: '12px',
                            background: 'white',
                            height: '100%',
                        }}
                    >
                        <ChartKit type="indicator" data={{data: [indicator]}} />
                    </div>
                ))}
            </div>

            <div
                style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '8px',
                    background: '#f5f5f5',
                    borderRadius: '4px',
                    fontSize: '12px',
                }}
            >
                <span>
                    Статус: <strong>{isConnected ? 'Подключено' : 'Отключено'}</strong>
                </span>
            </div>
        </div>
    );
}
