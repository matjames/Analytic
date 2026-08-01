import React from 'react';
import type { AppProps } from 'next/app';
import Head from 'next/head';
import '@styles/globals.css';
import { AuthProvider } from '@context/AuthContext';
import { LauncherProvider } from '@context/LauncherContext';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <Head>
        <meta charSet="utf-8" />
        <meta
          name="viewport"
          content="width=device-width, initial-scale=1, maximum-scale=5"
        />
        <meta httpEquiv="X-UA-Compatible" content="ie=edge" />
        <meta name="theme-color" content="#0ea5e9" />
      </Head>

      <AuthProvider mockMode={true}>
        <LauncherProvider mockMode={true}>
          <Component {...pageProps} />
        </LauncherProvider>
      </AuthProvider>
    </>
  );
}
