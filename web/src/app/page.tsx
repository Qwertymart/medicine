'use client';
import dynamic from 'next/dynamic';

const Dashboard = dynamic(() => import('@/components/Dashboard'), {ssr: false});
// const CTGStreamComponent = dynamic(() => import('@/components/CTGStream')); //, {ssr: false}

export default function Home() {
    return (
        <>
            {/* <CTGStreamComponent /> */}
            <Dashboard />
        </>
    );
}
