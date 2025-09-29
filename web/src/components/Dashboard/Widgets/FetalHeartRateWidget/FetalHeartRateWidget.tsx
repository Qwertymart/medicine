import {Graphs} from '../../../Graphs';
import {Wrapper} from '../Wrapper';

export const FetalHeartRateWidget = () => {
    return (
        <Wrapper>
            <div>
                <div style={{paddingBottom: '1%'}}>
                    <Graphs
                        dataType="fetal_heart_rate"
                        title="ЧСС плода (уд/мин)"
                        color="#6c59c2"
                    />
                </div>
                <div>
                    <Graphs
                        dataType="uterine_contractions"
                        title="Сокращения матки"
                        color="#ff2d87"
                    />
                </div>
            </div>
        </Wrapper>
    );
};
