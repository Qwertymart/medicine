'use client';

import {useEffect, useMemo, useState, useCallback, useRef} from 'react';
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

function useMockStream(dataType: string, isConnected: boolean) {
    const [streamData, setStreamData] = useState<CtgDataPoint[]>([]);
    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    const timeCounterRef = useRef(0);
    const lastValueRef = useRef(dataType === 'fetal_heart_rate' ? 140 : 20);

    useEffect(() => {
        if (isConnected) {
            if (intervalRef.current) {
                clearInterval(intervalRef.current);
                intervalRef.current = null;
            }
            return;
        }

        setStreamData([]);
        timeCounterRef.current = 0;

        const initialData: CtgDataPoint[] = [];
        const now = Date.now();
        const baseValue = dataType === 'fetal_heart_rate' ? 140 : 20;

        for (let i = 120; i >= 0; i--) {
            const timestamp = now - i * 1000;
            const value = generateDataPoint(dataType, baseValue, 120 - i);
            initialData.push({
                timestamp,
                device_id: 'mock_device',
                data_type: dataType,
                value: value,
                time_sec: 120 - i,
            });
            lastValueRef.current = value;
        }

        setStreamData(initialData);
        timeCounterRef.current = 120;

        intervalRef.current = setInterval(() => {
            timeCounterRef.current += 1;
            const timestamp = Date.now();

            const newValue = generateDataPoint(dataType, baseValue, timeCounterRef.current);
            lastValueRef.current = newValue;

            setStreamData((prev) => {
                const newDataPoint = {
                    timestamp,
                    device_id: 'mock_device',
                    data_type: dataType,
                    value: newValue,
                    time_sec: timeCounterRef.current,
                };

                const updatedData = [...prev, newDataPoint];
                if (updatedData.length > 600) {
                    return updatedData.slice(-600);
                }
                return updatedData;
            });
        }, 1000);

        return () => {
            if (intervalRef.current) {
                clearInterval(intervalRef.current);
                intervalRef.current = null;
            }
        };
    }, [dataType, isConnected]);

    return streamData;
}

function generateDataPoint(dataType: string, baseValue: number, timeSec: number): number {
    if (dataType === 'fetal_heart_rate') {
        let value = baseValue;

        value += Math.sin(timeSec * 0.1) * 8;
        value += (Math.random() - 0.5) * 6;

        if (timeSec % 150 === 0) {
            value += 15 + Math.random() * 10;
        }
        if (timeSec % 300 === 0) {
            value -= 10 + Math.random() * 8;
        }

        if (Math.random() > 0.98) {
            return -1;
        }

        return Math.max(100, Math.min(180, Math.round(value)));
    } else {
        let value = baseValue;

        value += Math.sin(timeSec * 0.05) * 3;
        value += (Math.random() - 0.5) * 2;

        const contractionPhase = timeSec % 210;
        if (contractionPhase < 60) {
            value += (contractionPhase / 60) * 35;
        } else if (contractionPhase < 90) {
            value += 35 + Math.random() * 15;
        } else if (contractionPhase < 120) {
            value += ((120 - contractionPhase) / 30) * 35;
        }

        return Math.max(0, Math.min(100, Math.round(value)));
    }
}

export function Graphs({dataType, title, color}: GraphsProps) {
    // const mockStreamData = useMockStream(dataType, false);
    // const isConnected = true;
    // const ctgData = mockStreamData;
    const {ctgData, isConnected} = useSession();

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
                        x: {
                            label: 'Time',
                            scale: 'time',
                        },
                        y: {
                            label: dataType === 'fetal_heart_rate' ? 'BPM' : 'Units',
                            range: dataType === 'fetal_heart_rate' ? [0, 200] : [0, 100],
                        },
                    },
                } as any,
            };
        }

        const mainData = filteredData.map((point) => (point.value !== -1 ? point.value : null));

        // окно ленты(последние 10 минут)
        const now = Date.now();
        const timeWindow = 10 * 60 * 1000;

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
                    x: {
                        label: 'Time',
                        scale: 'time',
                        range: [now - timeWindow, now] as [number, number],
                    },
                    y: {
                        label: dataType === 'fetal_heart_rate' ? 'BPM' : 'Units',
                        range: dataType === 'fetal_heart_rate' ? [0, 200] : [0, 100],
                    },
                },
                tooltip: {show: true},
            } as any,
        };
    }, [filteredData, title, color, dataType]);

    const [indicatorData, setIndicatorData] = useState<IndicatorWidgetData>({data: []});

    const updateIndicator = useCallback(() => {
        const indicator: IndicatorWidgetDataItem = {
            content: {
                current: {
                    value:
                        lastValidValue !== null
                            ? lastValidValue.toFixed(2).toString()
                            : 'No signal',
                    color: lastValidValue !== null ? color : '#ff4444',
                },
            },
            color: lastValidValue !== null ? color : '#ff4444',
            title,
            size: 's',
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
                    flexDirection: 'row',
                    height: '30vh',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#f9f9f9',
                    borderRadius: '8px',
                    border: '1px solid #e0e0e0',
                }}
            >
                <div style={{fontSize: 16, fontWeight: 500}}>Ожидание данных КТГ...</div>
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
                flexDirection: 'row',
                height: '30vh',
            }}
        >
            <div
                className={b('chart')}
                style={{
                    flex: 1,
                    border: '1px solid #e0e0e0',
                    borderRight: 'none',
                    borderRadius: '8px 0 0 8px',
                    padding: 12,
                    background: 'white',
                    minHeight: '30vh',
                    height: '30vh',
                    minWidth: 0,
                    overflow: 'hidden',
                }}
            >
                <ChartKit type="yagr" data={graphData} />
            </div>

            <div
                className={b('indicator_container')}
                style={{
                    width: 150,
                    display: 'flex',
                    flexDirection: 'column',
                    height: '32.1vh',
                }}
            >
                <div
                    className={b('indicator')}
                    style={{
                        border: '1px solid #e0e0e0',
                        borderLeft: 'none',
                        borderRadius: '0 8px 8px 0',
                        padding: 12,
                        background: 'white',
                        height: '40vh',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: 17,
                        fontWeight: 500,
                    }}
                >
                    <ChartKit type="indicator" data={indicatorData} />
                </div>
            </div>
        </div>
    );
}
