'use client';

import {DashKit, DashKitProps} from '@gravity-ui/dashkit';
import {useState, useEffect} from 'react';
import {Graphs} from '../Graphs';
import block from 'bem-cn-lite';
import {SessionProvider} from './SessionContext';
import {SessionControl} from './SessionCtrl';

const b = block('dashboard');

DashKit.setSettings({
    gridLayout: {margin: [9, 9]},
    isMobile: false,
});

DashKit.registerPlugins(
    {
        type: 'widget1',
        defaultLayout: {w: 16, h: 14},
        renderer: function Widget1() {
            return (
                <div style={{padding: '10px', background: '#ffffff'}}>
                    <h1>Первый виджет</h1>
                    <Graphs />
                </div>
            );
        },
    },
    {
        type: 'widget2',
        defaultLayout: {w: 16, h: 14},
        renderer: function Widget2() {
            return (
                <div style={{padding: '10px', background: '#ffffff'}}>
                    <h1>Второй виджет</h1>
                    <Graphs />
                </div>
            );
        },
    },
    {
        type: 'widget3',
        defaultLayout: {w: 6, h: 4},
        renderer: function Widget3() {
            return <div style={{padding: '10px', background: '#ffffff'}}>Третий виджет</div>;
        },
    },
);
const config: DashKitProps['config'] = {
    salt: '0.46703554571365613',
    counter: 3,
    items: [
        {
            id: 'w1',
            data: {},
            type: 'widget1',
            namespace: 'default',
        },
        {
            id: 'w2',
            data: {},
            type: 'widget2',
            namespace: 'default',
        },
        {
            id: 'w3',
            data: {},
            type: 'widget3',
            namespace: 'default',
        },
    ],
    layout: [
        {
            h: 25,
            i: 'w1',
            w: 16,
            x: 1,
            y: 0,
        },
        {
            h: 25,
            i: 'w2',
            w: 16,
            x: 17,
            y: 0,
        },
        {
            h: 4,
            i: 'w3',
            w: 16,
            x: 1,
            y: 25,
        },
    ],
    aliases: {},
    connections: [],
};

export function Dashboard() {
    const [mounted, setMounted] = useState(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    if (!mounted) {
        return (
            <div
                style={{
                    padding: 20,
                    height: '100vh',
                    width: '100vw',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}
            >
                Загрузка дашборда...
            </div>
        );
    }

    return (
        <SessionProvider>
            <div
                className={b('container')}
                style={{paddingLeft: 20, height: '100vh', width: '100vw'}}
            >
                <SessionControl />
                <DashKit config={config} editMode={true} />
            </div>
        </SessionProvider>
    );
}
