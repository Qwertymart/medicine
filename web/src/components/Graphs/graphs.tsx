'use client';

import {useEffect, useState} from 'react';
import ChartKit, {settings} from '@gravity-ui/chartkit';
import {YagrPlugin} from '@gravity-ui/chartkit/yagr';
import {IndicatorPlugin} from '@gravity-ui/chartkit/indicator';
import type {YagrWidgetData} from '@gravity-ui/chartkit/yagr';
import type {IndicatorWidgetData, IndicatorWidgetDataItem} from '@gravity-ui/chartkit/indicator';
import block from 'bem-cn-lite';

import '@gravity-ui/uikit/styles/styles.scss';

const b = block('graph');

settings.set({plugins: [YagrPlugin, IndicatorPlugin]});

const graphData: YagrWidgetData = {
    data: {
        timeline: [
            1636838612441, 1636925012441, 1637011412441, 1637097812441, 1637184212441,
            1637270612441, 1637357012441, 1637443412441, 1637529812441, 1637616212441,
        ],
        graphs: [
            {
                id: '0',
                name: 'Serie 1',
                color: '#6c59c2',
                data: [25, 52, 89, 72, 39, 49, 82, 59, 36, 5],
            },
            // {
            //     id: '1',
            //     name: 'Serie 2',
            //     color: '#6e8188',
            //     data: [37, 6, 51, 10, 65, 35, 72, 0, 94, 54],
            // },
        ],
    },
    libraryConfig: {
        chart: {
            series: {
                type: 'line',
            },
        },
        title: {
            text: 'line: random 10 pts',
        },
    },
};

export function Graphs() {
    const [indicatorData, setIndicatorData] = useState<IndicatorWidgetData>({
        data: [],
    });

    useEffect(() => {
        const indicators: IndicatorWidgetDataItem[] = graphData.data.graphs.map((graph) => {
            const lastValue = graph.data[graph.data.length - 1] ?? 0;

            return {
                content: {
                    current: {
                        value: Number(lastValue),
                    } as {value: string | number} & Record<string, unknown>,
                },
                color: graph.color || '#000000',
                title: graph.name || 'Unnamed',
                size: 'm',
                nowrap: true,
            };
        });

        setIndicatorData({data: indicators});
    }, []);

    return (
        <div
            className={b('container')}
            style={{
                display: 'flex',
                flexDirection: 'row',
                height: 600,
                gap: '20px',
            }}
        >
            <div className={b('chart')} style={{flex: 1}}>
                <ChartKit type="yagr" data={graphData} />
            </div>

            <div
                className={b('indicators')}
                style={{
                    display: 'flex',
                    gap: '16px',
                    height: '120px',
                }}
            >
                {indicatorData.data?.map((indicator, index) => (
                    <div
                        key={index}
                        style={{
                            flex: 1,
                            border: '1px solid #e0e0e0',
                            borderRadius: '8px',
                            padding: '12px',
                            background: 'white',
                        }}
                    >
                        <ChartKit type="indicator" data={{data: [indicator]}} />
                    </div>
                ))}
            </div>
        </div>
    );
}
