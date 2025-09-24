'use client';

import React from 'react';
import {configure, Lang, Theme, ThemeProvider} from '@gravity-ui/uikit';

import {HeartPulse} from '@gravity-ui/icons';
import {AsideHeader} from '@gravity-ui/navigation';

import {DARK, DEFAULT_THEME, Wrapper} from '../Wrapper';

configure({lang: Lang.Ru});

interface AppProps {
    children: React.ReactNode;
}
export const App: React.FC<AppProps> = ({children}) => {
    const [theme, setTheme] = React.useState<Theme>(DEFAULT_THEME);
    const isDark = theme === DARK;
    return (
        <ThemeProvider theme={theme}>
            <AsideHeader
                logo={{icon: HeartPulse, text: 'medicine'}}
                compact={true}
                hideCollapseButton={true}
                renderContent={() => (
                    <Wrapper setTheme={setTheme} isDark={isDark}>
                        {children}
                    </Wrapper>
                )}
            />
        </ThemeProvider>
    );
};
