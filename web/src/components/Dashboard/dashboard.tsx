'use client';

import {DashKit, DashKitProps} from '@gravity-ui/dashkit';
import {useState, useEffect} from 'react';
import {Graphs} from '../Graphs';
import block from 'bem-cn-lite';
import {SessionProvider, useSession} from './SessionContext';
import {SessionControl} from './SessionCtrl';
import {Card, Text, Button} from '@gravity-ui/uikit';

const b = block('dashboard');

DashKit.setSettings({
    gridLayout: {margin: [9, 9]},
    isMobile: false,
});

const FetalHeartRateWidget = () => {
    const {startSession, stopSession, activeSession} = useSession();

    return (
        <div style={{padding: '10px', background: '#ffffff', height: '100%'}}>
            <div
                style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '10px',
                }}
            >
                <Text variant="header-2">ЧСС плода</Text>
                {!activeSession ? (
                    <Button view="action" size="s" onClick={startSession}>
                        Начать сессию
                    </Button>
                ) : (
                    <Button view="outlined-danger" size="s" onClick={stopSession}>
                        Остановить
                    </Button>
                )}
            </div>
            <Graphs dataType="fetalHeartRate" title="ЧСС плода (уд/мин)" color="#6c59c2" />
        </div>
    );
};

const UterineContractionsWidget = () => {
    const {ctgData} = useSession();

    const calculateStats = () => {
        if (ctgData.length === 0) return null;

        const heartRates = ctgData.map((p) => p.fetalHeartRate).filter(Boolean) as number[];
        const contractions = ctgData.map((p) => p.uterineContractions).filter(Boolean) as number[];

        return {
            avgHeartRate: heartRates.length
                ? (heartRates.reduce((a, b) => a + b) / heartRates.length).toFixed(1)
                : '0',
            maxContraction: contractions.length ? Math.max(...contractions) : 0,
            avgContraction: contractions.length
                ? (contractions.reduce((a, b) => a + b) / contractions.length).toFixed(1)
                : '0',
        };
    };

    const stats = calculateStats();

    return (
        <div style={{padding: '10px', background: '#ffffff', height: '100%'}}>
            <Text variant="header-2" style={{marginBottom: '10px'}}>
                Сокращения матки
            </Text>

            <Graphs dataType="uterineContractions" title="Сокращения матки" color="#ff2d87" />

            {stats && (
                <Card view="filled" style={{padding: '10px', marginTop: '10px'}}>
                    <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px'}}>
                        <div>
                            <Text variant="body-2">Средняя ЧСС</Text>
                            <Text variant="body-1">{stats.avgHeartRate} уд/мин</Text>
                        </div>
                        <div>
                            <Text variant="body-2">Макс. сокращение</Text>
                            <Text variant="body-1">{stats.maxContraction}</Text>
                        </div>
                    </div>
                </Card>
            )}
        </div>
    );
};

DashKit.registerPlugins(
    {
        type: 'fetal-heart-rate',
        defaultLayout: {w: 16, h: 14},
        renderer: FetalHeartRateWidget,
    },
    {
        type: 'uterine-contractions',
        defaultLayout: {w: 16, h: 14},
        renderer: UterineContractionsWidget,
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
    counter: 2,
    items: [
        {
            id: 'w1',
            data: {},
            type: 'fetal-heart-rate',
            namespace: 'default',
        },
        {
            id: 'w2',
            data: {},
            type: 'uterine-contractions',
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
// 'use client';

// import {DashKit, DashKitProps} from '@gravity-ui/dashkit';
// import {useState, useEffect} from 'react';
// import {Graphs} from '../Graphs';
// import block from 'bem-cn-lite';
// import {SessionProvider, useSession} from './SessionContext';
// import {SessionControl} from './SessionCtrl';

// const b = block('dashboard');

// DashKit.setSettings({
//     gridLayout: {margin: [9, 9]},
//     isMobile: false,
// });

// DashKit.registerPlugins(
//     {
//         type: 'widget1',
//         defaultLayout: {w: 16, h: 14},
//         renderer: function Widget1() {
//             return (
//                 <div style={{padding: '10px', background: '#ffffff'}}>
//                     <h1>Первый виджет</h1>
//                     <Graphs />
//                 </div>
//             );
//         },
//     },
//     {
//         type: 'widget2',
//         defaultLayout: {w: 16, h: 14},
//         renderer: function Widget2() {
//             return (
//                 <div style={{padding: '10px', background: '#ffffff'}}>
//                     <h1>Второй виджет</h1>
//                     <Graphs />
//                 </div>
//             );
//         },
//     },
//     {
//         type: 'widget3',
//         defaultLayout: {w: 6, h: 4},
//         renderer: function Widget3() {
//             return <div style={{padding: '10px', background: '#ffffff'}}>Третий виджет</div>;
//         },
//     },
// );
// const config: DashKitProps['config'] = {
//     salt: '0.46703554571365613',
//     counter: 3,
//     items: [
//         {
//             id: 'w1',
//             data: {},
//             type: 'widget1',
//             namespace: 'default',
//         },
//         {
//             id: 'w2',
//             data: {},
//             type: 'widget2',
//             namespace: 'default',
//         },
//         {
//             id: 'w3',
//             data: {},
//             type: 'widget3',
//             namespace: 'default',
//         },
//     ],
//     layout: [
//         {
//             h: 25,
//             i: 'w1',
//             w: 16,
//             x: 1,
//             y: 0,
//         },
//         {
//             h: 25,
//             i: 'w2',
//             w: 16,
//             x: 17,
//             y: 0,
//         },
//         {
//             h: 4,
//             i: 'w3',
//             w: 16,
//             x: 1,
//             y: 25,
//         },
//     ],
//     aliases: {},
//     connections: [],
// };

// export function Dashboard() {
//     const [mounted, setMounted] = useState(false);
//     const {
//             cardId,
//             deviceId,
//             sessionId,
//             activeSession,
//             isLoading,
//             error,
//             startSession,
//             stopSession,
//             clearError,
//         } = useSession();

//     useEffect(() => {
//         setMounted(true);
//     }, []);

//     if (!mounted) {
//         return (
//             <div
//                 style={{
//                     padding: 20,
//                     height: '100vh',
//                     width: '100vw',
//                     display: 'flex',
//                     alignItems: 'center',
//                     justifyContent: 'center',
//                 }}
//             >
//                 Загрузка дашборда...
//             </div>
//         );
//     }

//     return (
//         <SessionProvider>
//             <div
//                 className={b('container')}
//                 style={{paddingLeft: 20, height: '100vh', width: '100vw'}}
//             >
//                 <SessionControl />
//                 <DashKit config={config} editMode={true} />
//             </div>
//         </SessionProvider>
//     );
// }
