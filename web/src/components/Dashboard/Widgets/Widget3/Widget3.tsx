import {Card, Text} from '@gravity-ui/uikit';
import {Wrapper} from '../Wrapper';

export const Widget3 = () => {
    return (
        <Wrapper>
            <Text variant="header-2" style={{marginBottom: '10px'}}>
                Крутая карточка с резами
            </Text>

            <Card view="filled" style={{padding: '10px', marginTop: '10px'}}>
                <div
                    style={{
                        display: 'grid',
                        gridTemplateColumns: '1fr 1fr',
                        gap: '10px',
                    }}
                >
                    <Text>ТУТ ЧТО_ТО БУДЕТ</Text>
                </div>
            </Card>
        </Wrapper>
    );
};
