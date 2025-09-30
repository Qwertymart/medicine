import type {Metadata} from 'next';
import {DEFAULT_BODY_CLASSNAME} from '../components/Wrapper';
import {App} from '../components/App';

import '@gravity-ui/uikit/styles/fonts.css';
import '@gravity-ui/uikit/styles/styles.css';
import '../styles/globals.scss';
import '../styles/widgets.scss';

export const metadata: Metadata = {
    title: 'CTG Control',
    description: 'Make with Love',
};

export default function RootLayout({children}: {children: React.ReactNode}) {
    return (
        <html lang="en">
            <body className={DEFAULT_BODY_CLASSNAME}>
                <App>{children}</App>
            </body>
        </html>
    );
}
