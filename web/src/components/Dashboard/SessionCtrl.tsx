'use client';

import {CirclePlayFill, StopFill} from '@gravity-ui/icons';
import {Button, Icon, Alert} from '@gravity-ui/uikit';
import block from 'bem-cn-lite';
import {useSession} from './SessionContext';

const b = block('dashboard-sessiontCtrl');

export function SessionControl() {
    const {
        cardId,
        deviceId,
        sessionId,
        activeSession,
        isLoading,
        error,
        startSession,
        stopSession,
        clearError,
    } = useSession();

    return (
        <div className={b('container')}>
            {error && (
                <Alert
                    theme="danger"
                    title="Ошибка"
                    message={error}
                    onClose={clearError}
                    style={{ marginBottom: 10 }}
                />
            )}
            <div
                className={b('buttons')}
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    paddingBottom: 10,
                }}
            >
                <Button
                    view="outlined"
                    size="xl"
                    onClick={startSession}
                    disabled={isLoading || (activeSession?.status === 'active')}
                    loading={isLoading && !(activeSession?.status === 'active')}
                >
                    <Icon data={CirclePlayFill} size={36} />
                    Старт
                </Button>
                <Button
                    view="outlined"
                    size="xl"
                    onClick={stopSession}
                    disabled={isLoading || !(activeSession?.status === 'active')}
                    loading={isLoading && activeSession?.status === 'active'}
                >
                    Стоп
                    <Icon data={StopFill} size={36} />
                </Button>
            </div>

            {activeSession && (
                <div className={b('sessionInfo')}>
                    Активная сессия: {sessionId}
                    <br />
                    Медицинская карта: {cardId}
                    <br />
                    Устройство: {deviceId}
                </div>
            )}
        </div>
    );
}
