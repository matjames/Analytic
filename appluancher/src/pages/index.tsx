import React, { useState } from 'react';
import Head from 'next/head';
import { Application } from '@typings/index';
import { Header } from '@components/Header';
import { AppLauncher } from '@components/AppLauncher';
import { ErrorBoundary } from '@components/ErrorBoundary';
import { useAuth } from '@context/AuthContext';
import { useLauncher } from '@context/LauncherContext';
import { canAccessApp } from '@utils/helpers';
import { Hero } from '@components/landing/Hero';
import { Problem } from '@components/landing/Problem';
import { Solution } from '@components/landing/Solution';
import { Capabilities } from '@components/landing/Capabilities';
import { Industries } from '@components/landing/Industries';
import { AITeam } from '@components/landing/AITeam';
import { WhyChoose } from '@components/landing/WhyChoose';
import { Security } from '@components/landing/Security';
import { CTA } from '@components/landing/CTA';
import { Footer } from '@components/landing/Footer';

export default function Home() {
  const { user } = useAuth();
  const { applications } = useLauncher();
  const [launcherOpen, setLauncherOpen] = useState(false);

  const handleAppLaunch = async (app: Application) => {
    // Track the launch
    try {
      await fetch('/api/analytics/app-launch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          appId: app.id,
          appName: app.name,
          category: app.category,
          userRole: user?.role,
          timestamp: new Date().toISOString(),
        }),
      });
    } catch (error) {
      console.error('Failed to track app launch:', error);
    }

    // Navigate to app in the same browser tab
    window.location.href = app.url;
  };

  return (
    <>
      <Head>
        <title>StatGate - Enterprise Evidence Intelligence Platform</title>
        <meta
          name="description"
          content="StatGate is an Enterprise Evidence Intelligence Platform that unifies data engineering, statistics, artificial intelligence, research management, GIS, reporting, and executive decision support into one intelligent ecosystem."
        />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/icons/logo.png" />
      </Head>

      <ErrorBoundary>
        <div className="min-h-screen bg-white">
          {/* Header with App Launcher */}
          <Header
            onAppLauncherClick={() => setLauncherOpen(!launcherOpen)}
          />

          {/* App Launcher Modal */}
          <AppLauncher
            apps={applications}
            isOpen={launcherOpen}
            onClose={() => setLauncherOpen(false)}
            onAppSelect={handleAppLaunch}
            canAccess={(app) => canAccessApp(app, user)}
          />

          <main>
            <Hero />
            <Problem />
            <Solution />
            <Capabilities />
            <Industries />
            <AITeam />
            <WhyChoose />
            <Security />
            <CTA />
          </main>
          <Footer />
        </div>
      </ErrorBoundary>
    </>
  );
}