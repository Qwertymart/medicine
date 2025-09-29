import {Card, Text} from '@gravity-ui/uikit';
import {useSession} from '../../SessionContext';
import {Graphs} from '../../../Graphs';
import { Wrapper } from '../Wrapper';

export const UterineContractionsWidget = () => {
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
        <Wrapper>
            <Text variant="header-2" style={{marginBottom: '10px'}}>
                Сокращения матки
            </Text>

            <Graphs dataType="uterine_contractions" title="Сокращения матки" color="#ff2d87" />

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
        </Wrapper>
    );
};
