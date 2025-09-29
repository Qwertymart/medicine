export const Wrapper = ({children}: {children: React.ReactNode}) => {
    return (
        <div
            style={{
                padding: '10px',
                background: '#6e6262fa',
                height: '100%',
                border: '1px solid black',
                borderRadius: '3%'
            }}
        >
            {children}
        </div>
    );
};
