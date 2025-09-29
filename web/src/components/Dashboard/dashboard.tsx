'use client';

import block from 'bem-cn-lite';
import {DashKit, DashKitProps} from '@gravity-ui/dashkit';
import {useState, useEffect} from 'react';
import {SessionProvider} from './SessionContext';
import {SessionControl} from './SessionCtrl';
import {Loader, Switch} from '@gravity-ui/uikit';
import {
    FetalHeartRateWidget,
    // UterineContractionsWidget,
    Widget3,
} from '@/components/Dashboard/Widgets';

const b = block('dashboard');

DashKit.setSettings({
    gridLayout: {margin: [40, 40]},
    isMobile: false,
});

DashKit.registerPlugins(
    {
        type: 'fetal-heart-rate',
        defaultLayout: {w: 40, h: 20},
        renderer: FetalHeartRateWidget,
    },
    // {
    //     type: 'uterine-contractions',
    //     defaultLayout: {w: 16, h: 25},
    //     renderer: UterineContractionsWidget,
    // },
    {
        type: 'widget3',
        defaultLayout: {w: 16, h: 20},
        renderer: Widget3,
    },
);

const config: DashKitProps['config'] = {
    salt: '0.46703554571365613',
    counter: 2,
    items: [
        {
            // ТУТ ОБА ГРАФИКА!!!!!!!!!!!!!!!!!!!!!!! не забыть
            id: 'w1',
            data: {},
            type: 'fetal-heart-rate',
            namespace: 'default',
        },
        // {
        //     id: 'w2',
        //     data: {},
        //     type: 'uterine-contractions',
        //     namespace: 'default',
        // },
        {
            id: 'w3',
            data: {},
            type: 'widget3',
            namespace: 'default',
        },
    ],
    layout: [
        {
            h: 15,
            i: 'w1',
            w: 40,
            x: 0,
            y: 0,
        },
        // {
        //     h: 12,
        //     i: 'w2',
        //     w: 16,
        //     x: 17,
        //     y: 0,
        // },
        {
            h: 5,
            i: 'w3',
            w: 16,
            x: 0,
            y: 40,
        },
    ],
    aliases: {},
    connections: [],
};

export function Dashboard() {
    const [mounted, setMounted] = useState(false);
    const [isEditMode, setIsEditMode] = useState<boolean>(false);

    useEffect(() => {
        setMounted(true);
    }, []);

    if (!mounted) {
        return (
            <div
                style={{
                    padding: 10,
                    height: 'auto',
                    width: '100vw',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}
            >
                <Loader size="l" />
                {/* Загрузка дашборда... */}
            </div>
        );
    }

    return (
        <div className={b('wrapper')}>
            <Switch
                className={b('editModeSelector')}
                size="l"
                onChange={() => {
                    setIsEditMode((cur) => !cur);
                }}
                style={{float: 'right', paddingRight: '10vw'}}
            >
                Переставить виджеты
            </Switch>
            <SessionProvider>
                <div
                    className={b('container')}
                    // style={{paddingLeft: 20, height: 'auto', width: '100vw',}}
                    style={{paddingLeft: 20, height: 'auto', width: '90vw'}}
                >
                    <SessionControl />
                    <DashKit config={config} editMode={isEditMode} />
                </div>
            </SessionProvider>
        </div>
    );
}
