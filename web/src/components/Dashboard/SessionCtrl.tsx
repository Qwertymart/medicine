'use client';

import {CirclePlayFill, StopFill} from '@gravity-ui/icons';
import {Button, Icon, Alert, TextInput} from '@gravity-ui/uikit';
import block from 'bem-cn-lite';
import {useSession} from './SessionContext';
import {useState} from 'react';

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

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

    const [inputCardId, setInputCardId] = useState('');
    const [validationState, setValidationState] = useState<'invalid' | undefined>(undefined);
    const [errorMessage, setErrorMessage] = useState<string | undefined>(undefined);

    const handleInputChange = (value: string) => {
        setInputCardId(value);

        if (value.trim() === '') {
            setValidationState(undefined);
            setErrorMessage(undefined);
        } else if (!UUID_REGEX.test(value)) {
            setValidationState('invalid');
            setErrorMessage(
                'Введите корректный UUID формат (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)',
            );
        } else {
            setValidationState(undefined);
            setErrorMessage(undefined);
        }
    };

    const handleStartSession = () => {
        if (inputCardId.trim() && UUID_REGEX.test(inputCardId)) {
            startSession(inputCardId);
        } else {
            setValidationState('invalid');
            setErrorMessage('Введите корректный UUID перед началом сессии');
        }
    };

    return (
        <div className={b('container')}>
            {error && (
                <Alert
                    theme="danger"
                    title="Ошибка"
                    message={error}
                    onClose={clearError}
                    style={{marginBottom: 10}}
                />
            )}

            <div
                className={b('buttons')}
                style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    paddingBottom: 10,
                    gap: '10px',
                }}
            >
                {!activeSession ? (
                    <>
                        <TextInput
                            value={inputCardId}
                            onUpdate={handleInputChange}
                            placeholder="Введите card_id"
                            size="m"
                            style={{width: '200px'}}
                            validationState={validationState}
                            errorMessage={errorMessage}
                            hasClear
                        />
                        <Button
                            view="outlined"
                            size="xl"
                            onClick={handleStartSession}
                            disabled={isLoading || !inputCardId.trim()}
                            loading={isLoading}
                        >
                            <Icon data={CirclePlayFill} size={36} />
                            Старт
                        </Button>
                    </>
                ) : (
                    <Button
                        view="outlined"
                        size="xl"
                        onClick={stopSession}
                        disabled={isLoading}
                        loading={isLoading}
                    >
                        Стоп
                        <Icon data={StopFill} size={36} />
                    </Button>
                )}
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
