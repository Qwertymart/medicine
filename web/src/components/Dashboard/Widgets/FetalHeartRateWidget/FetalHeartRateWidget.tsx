import {Card, Text} from '@gravity-ui/uikit';
import {useSession} from '../../SessionContext';
import {Graphs} from '../../../Graphs';
import {Wrapper} from '../Wrapper';

export const FetalHeartRateWidget = () => {
    const {activeSession} = useSession();

    return (
        <Wrapper>
            <Text variant="header-2" style={{marginBottom: '10px'}}>
                ЧСС плода
            </Text>
            {activeSession && (
                <Card view="filled" style={{padding: '10px', marginBottom: '10px'}}>
                    <Text variant="body-2">Card ID: {activeSession.card_id}</Text>
                </Card>
            )}
            <div>
                <Graphs dataType="fetal_heart_rate" title="ЧСС плода (уд/мин)" color="#6c59c2" />
            </div>
        </Wrapper>
    );
};
