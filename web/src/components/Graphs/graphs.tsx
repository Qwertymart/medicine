'use client';

import {useEffect, useMemo, useState, useCallback} from 'react';
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
    dataType: 'fetal_heart_rate' | 'uterine_contractions';
    title: string;
    color: string;
}

interface CtgDataPoint {
    timestamp: number;
    device_id: string;
    data_type: string;
    value: number;
    time_sec: number;
}

export function Graphs({dataType, title, color}: GraphsProps) {
    const {ctgData, isConnected} = useSession();
    const [indicatorData, setIndicatorData] = useState<IndicatorWidgetData>({data: []});

    const filteredData = useMemo(() => {
        if (!ctgData || ctgData.length === 0) return [];

        return (ctgData as CtgDataPoint[])
            .filter((point) => point.data_type === dataType)
            .sort((a, b) => a.timestamp - b.timestamp);
    }, [ctgData, dataType]);

    const lastValidValue = useMemo(() => {
        const validPoints = filteredData.filter((point) => point.value !== -1);
        return validPoints.length > 0 ? validPoints[validPoints.length - 1].value : null;
    }, [filteredData]);

    const graphData: YagrWidgetData = useMemo(() => {
        if (filteredData.length === 0) {
            return {
                data: {
                    timeline: [],
                    graphs: [],
                },
                libraryConfig: {
                    title: {text: title},
                    axes: {
                        x: {label: 'Time', scale: 'time'},
                        y: {
                            label: dataType === 'fetal_heart_rate' ? 'BPM' : 'Units',
                            range: dataType === 'fetal_heart_rate' ? [0, 200] : [0, 100],
                        },
                    },
                },
            };
        }

        const mainData = filteredData.map((point) => (point.value !== -1 ? point.value : null));

        return {
            data: {
                timeline: filteredData.map((point) => point.timestamp),
                graphs: [
                    {
                        id: 'main',
                        name: title,
                        color,
                        data: mainData,
                        lineWidth: 2,
                        pointSize: 0,
                    },
                ],
            },
            libraryConfig: {
                chart: {
                    series: {
                        type: 'line',
                        spanGaps: false,
                    },
                },
                title: {
                    text: title,
                },
                axes: {
                    x: {label: 'Time', scale: 'time'},
                    y: {
                        label: dataType === 'fetal_heart_rate' ? 'BPM' : 'Units',
                        range: dataType === 'fetal_heart_rate' ? [0, 200] : [0, 100],
                    },
                },
                tooltip: {show: true},
            },
        };
    }, [filteredData, title, color, dataType]);

    const updateIndicator = useCallback(() => {
        const indicator: IndicatorWidgetDataItem = {
            content: {
                current: {
                    value: lastValidValue !== null ? lastValidValue.toString() : 'No signal',
                    color: lastValidValue !== null ? color : '#ff4444',
                },
            },
            color: lastValidValue !== null ? color : '#ff4444',
            title,
            size: 'm',
            nowrap: true,
        };

        setIndicatorData({data: [indicator]});
    }, [lastValidValue, title, color]);

    useEffect(() => {
        updateIndicator();
    }, [updateIndicator]);

    const renderLoadingState = useCallback(
        () => (
            <div
                className={b('container')}
                style={{
                    display: 'flex',
                    flexDirection: 'column',
                    height: 400,
                    gap: '20px',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#f9f9f9',
                    borderRadius: '8px',
                    border: '1px solid #e0e0e0',
                }}
            >
                <div style={{fontSize: 16, fontWeight: 500}}>Ожидание данных КТГ...</div>
                <div style={{fontSize: 14, color: '#666'}}>
                    Запустите сессию для начала мониторинга
                </div>
            </div>
        ),
        [],
    );

    if (!isConnected && ctgData.length === 0) {
        return renderLoadingState();
    }

    return (
        <div
            className={b('container')}
            style={{
                display: 'flex',
                flexDirection: 'column',
                height: 'auto',
                gap: 15,
            }}
        >
            <div
                className={b('chart')}
                style={{
                    flex: 1,
                    border: '1px solid #e0e0e0',
                    borderRadius: 8,
                    padding: 12,
                    background: 'white',
                    minHeight: 300,
                }}
            >
                <ChartKit type="yagr" data={graphData} />
            </div>

            <div
                style={{
                    display: 'grid',
                    gridTemplateColumns: '1fr',
                    gap: 15,
                    height: 100,
                }}
            >
                <div
                    style={{
                        border: '1px solid #e0e0e0',
                        borderRadius: 8,
                        padding: 12,
                        background: 'white',
                        height: '100%',
                    }}
                >
                    <ChartKit type="indicator" data={indicatorData} />
                </div>
            </div>
        </div>
    );
}
