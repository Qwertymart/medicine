'use client';

import {useEffect, useMemo, useState} from 'react';
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

interface CtgDataPoint {
    timestamp: number;
    device_id: string;
    data_type: string;
    value: number;
    time_sec: number;
}

export function Graphs({dataType, title, color}: GraphsProps) {
    const {ctgData, isConnected} = useSession();
    const [indicatorData, setIndicatorData] = useState<IndicatorWidgetData>({
        data: [],
    });

    const dataTypeProto = useMemo(() => {
        return dataType === 'fetalHeartRate' ? 'fetal_heart_rate' : 'uterine_contractions';
    }, [dataType]);

    const filteredData = useMemo(() => {
        const filtered = (ctgData as CtgDataPoint[])
            .filter((point) => point.data_type === dataTypeProto)
            .sort((a, b) => a.timestamp - b.timestamp);

        console.log(`Filtered ${dataType} data:`, {
            total: filtered.length,
            normal: filtered.filter((p) => p.value !== -1).length,
            problems: filtered.filter((p) => p.value === -1).length,
            sample: filtered.slice(-5),
        });

        return filtered;
    }, [ctgData, dataTypeProto]);

    const lastValidValue = useMemo(() => {
        const validPoints = filteredData.filter((point) => point.value !== -1);
        return validPoints.length > 0 ? validPoints[validPoints.length - 1].value : null;
    }, [filteredData]);

    const graphData: YagrWidgetData = useMemo(() => {
        const graphData = filteredData.map((point) => (point.value !== -1 ? point.value : null));

        return {
            data: {
                timeline: filteredData.map((point) => point.timestamp),
                graphs: [
                    {
                        id: dataType,
                        name: title,
                        color: color,
                        data: graphData,
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
                    },
                    y: {
                        label: dataType === 'fetalHeartRate' ? 'BPM' : 'Units',
                        range: dataType === 'fetalHeartRate' ? [0, 200] : [0, 100],
                    },
                },
                tooltip: {
                    show: true,
                },
            },
        };
    }, [filteredData, title, color, dataType]);

    useEffect(() => {
        const indicator: IndicatorWidgetDataItem = {
            content: {
                current: {
                    value: lastValidValue !== null ? lastValidValue.toString() : 'No signal',
                    color: lastValidValue !== null ? color : '#ff4444',
                },
            },
            color: lastValidValue !== null ? color : '#ff4444',
            title: title,
            size: 'm',
            nowrap: true,
        };

        setIndicatorData({data: [indicator]});
    }, [lastValidValue, title, color]);

    // const dataStats = useMemo(() => {
    //     const total = filteredData.length;
    //     const valid = filteredData.filter((p) => p.value !== -1).length;
    //     const problems = filteredData.filter((p) => p.value === -1).length;

    //     return {total, valid, problems};
    // }, [filteredData]);

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
                    background: '#f9f9f9',
                    borderRadius: '8px',
                    border: '1px solid #e0e0e0',
                }}
            >
                <div style={{fontSize: '16px', fontWeight: '500'}}>Ожидание данных КТГ...</div>
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
                    minHeight: '300px',
                }}
            >
                <ChartKit type="yagr" data={graphData} />
            </div>

            <div
                style={{
                    display: 'grid',
                    gridTemplateColumns: '1fr 1fr',
                    gap: '15px',
                    height: '100px',
                }}
            >
                {/* Индикатор */}
                <div
                    style={{
                        border: '1px solid #e0e0e0',
                        borderRadius: '8px',
                        padding: '12px',
                        background: 'white',
                        height: '100%',
                    }}
                >
                    <ChartKit type="indicator" data={indicatorData} />
                </div>
            </div>

            <div
                style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '12px',
                    background: isConnected ? '#e8f5e8' : '#fff3cd',
                    borderRadius: '6px',
                    fontSize: '14px',
                    border: `1px solid ${isConnected ? '#d4edda' : '#ffeaa7'}`,
                }}
            >
                <span>
                    Status: <strong>{isConnected ? 'Connected' : 'Disconnected'}</strong>
                </span>
                <span style={{color: '#666'}}>
                    Last update:{' '}
                    {filteredData.length > 0
                        ? new Date(
                              filteredData[filteredData.length - 1].timestamp,
                          ).toLocaleTimeString()
                        : 'No data'}
                </span>
            </div>
        </div>
    );
}
